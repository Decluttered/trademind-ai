package productpublish

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	douyinmetrics "github.com/trademind-ai/trademind/backend/internal/metrics/douyin"
	"github.com/trademind-ai/trademind/backend/internal/modules/idempotency"
	"github.com/trademind-ai/trademind/backend/internal/modules/operationlog"
	"github.com/trademind-ai/trademind/backend/internal/modules/product"
	"github.com/trademind-ai/trademind/backend/internal/pkg/tasklease"
	platformdouyin "github.com/trademind-ai/trademind/backend/internal/providers/platform/douyinshop"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// DouyinCreateDraftBody POST create-draft request.
type DouyinCreateDraftBody struct {
	ShopID      string     `json:"shopId"`
	PublishMode string     `json:"publishMode"`
	Force       bool       `json:"force"`
	BatchID     *uuid.UUID `json:"-"`
}

type douyinDraftSnapshot struct {
	PublicationID      uuid.UUID                                 `json:"publicationId"`
	ConfigID           string                                    `json:"configId,omitempty"`
	PublishMode        string                                    `json:"publishMode"`
	MappingHash        string                                    `json:"mappingHash,omitempty"`
	Mapping            map[string]any                            `json:"mappingSnapshot,omitempty"`
	FrozenRequest      *platformdouyin.CreateProductDraftRequest `json:"frozenRequest,omitempty"`
	FrozenMapping      *product.DouyinDraftMapping               `json:"frozenMapping,omitempty"`
	ExecutionAttemptID *uuid.UUID                                `json:"executionAttemptId,omitempty"`
	OperationTaskID    *uuid.UUID                                `json:"operationTaskId,omitempty"`
	FrozenPayloadHash  string                                    `json:"frozenPayloadHash,omitempty"`
}

// CreateDouyinDraftTask is retained for API compatibility. Real Douyin writes
// must be created from a frozen and approved operation task.
func (s *Service) CreateDouyinDraftTask(c *gin.Context, productID uuid.UUID, body DouyinCreateDraftBody, adminID *uuid.UUID) (*TaskDTO, error) {
	return nil, ErrDouyinOperationTaskRequired
}

// ProcessDouyinDraftTask executes douyin_shop save_as_platform_draft publish tasks.
func (s *Service) ProcessDouyinDraftTask(ctx context.Context, taskID uuid.UUID, workerID string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("productpublish: no db")
	}
	taskRow, claim, claimed, err := s.tryClaimProductPublishTask(ctx, taskID, workerID, s.publishLeaseTTL())
	if err != nil || !claimed || taskRow == nil {
		return err
	}
	snap, buildRes, bindingErr := s.validateProductionDouyinTask(ctx, taskRow)
	if bindingErr != nil {
		fin := time.Now().UTC()
		_ = s.finishProductPublishTask(ctx, taskID, workerID, claim, map[string]any{
			"status": TaskFailed, "publish_status": StatusPubFailed, "error_code": ErrorDouyinOperationTaskRequired,
			"error_message": ErrDouyinOperationTaskRequired.Error(), "retryable": false, "finished_at": &fin,
		})
		if taskRow.ExecutionAttemptID != nil && *taskRow.ExecutionAttemptID != uuid.Nil && s.OperationResults != nil {
			_ = s.OperationResults.MarkFailed(ctx, *taskRow.ExecutionAttemptID, taskID, ErrorDouyinOperationTaskRequired, ErrDouyinOperationTaskRequired.Error(), false, false, datatypes.JSON([]byte(`{}`)))
		}
		return bindingErr
	}
	if err := s.guardDouyinWorker(ctx, taskID, taskRow.ShopID, platformdouyin.FeatureProductDraft, false, taskRow.CreatedBy); err != nil {
		return err
	}
	cancelRen := s.startPublishLeaseRenewal(ctx, taskID, workerID, claim, s.publishLeaseTTL())
	defer cancelRen()

	_ = s.DB.WithContext(ctx).Model(&ProductPublishTask{}).Where("id = ? AND tenant_id = ?", taskID, taskRow.TenantID).
		Updates(map[string]any{"publish_status": StatusCreatingPlatformDraft}).Error

	fail := func(code, msg string, retryable bool, requestID string, raw map[string]any) error {
		fin := time.Now().UTC()
		rawJSON, _ := json.Marshal(sanitizeRawErrorMap(raw))
		updates := map[string]any{
			"status":             TaskFailed,
			"publish_status":     StatusPubFailed,
			"error_code":         code,
			"error_message":      msg,
			"retryable":          retryable,
			"request_id":         requestID,
			"finished_at":        &fin,
			"platform_raw_error": datatypes.JSON(rawJSON),
		}
		_ = s.finishProductPublishTask(ctx, taskID, workerID, claim, updates)
		if snap, ok := parseDouyinDraftSnapshot(taskRow.Input); ok {
			_ = s.DB.WithContext(ctx).Model(&ProductPublication{}).
				Where("id = ? AND tenant_id = ? AND product_id = ? AND shop_id = ?", snap.PublicationID, taskRow.TenantID, taskRow.ProductID, taskRow.ShopID).
				Updates(map[string]any{"status": StatusPubFailed, "publish_status": StatusPubFailed, "updated_at": fin}).Error
		}
		if s.OpLog != nil {
			_ = s.OpLog.WriteBackground(ctx, operationlog.WriteOpts{
				AdminUserID: taskRow.CreatedBy,
				Action:      "douyin.product.draft.create.failed",
				Resource:    "product_publish_task",
				ResourceID:  taskID.String(),
				Status:      "failed",
				Message:     fmt.Sprintf("taskId=%s code=%s err=%s requestId=%s", taskID, code, truncateMsg(msg), requestID),
			})
		}
		douyinmetrics.RecordProductDraftCreate(false)
		if snap, ok := parseDouyinDraftSnapshot(taskRow.Input); ok && snap.ExecutionAttemptID != nil && *snap.ExecutionAttemptID != uuid.Nil && s.OperationResults != nil {
			resultUnknown := code == platformdouyin.CodeDouyinUnknownResult || code == platformdouyin.CodeDouyinRequestTimeout
			_ = s.OperationResults.MarkFailed(ctx, *snap.ExecutionAttemptID, taskID, code, msg, retryable && !resultUnknown, resultUnknown, datatypes.JSON(rawJSON))
		}
		return fmt.Errorf("%s", msg)
	}

	if err := s.WriteControl.EvaluateWrite(ctx, taskRow.TenantID, taskRow.ShopID, taskRow.ProductID, len(buildRes.APIReq.SpecPricesV2)); err != nil {
		return fail("PRODUCTION_WRITE_BLOCKED", "production write blocked by runtime control", false, "", nil)
	}
	if err := s.OperationResults.MarkRunning(ctx, *snap.ExecutionAttemptID, taskID); err != nil {
		return fail("OPERATION_ATTEMPT_CONFLICT", "operation attempt could not enter running state", false, "", nil)
	}
	if s.OpLog != nil {
		_ = s.OpLog.WriteBackground(ctx, operationlog.WriteOpts{
			AdminUserID: taskRow.CreatedBy,
			Action:      "douyin.product.payload.build",
			Resource:    "product_publish_task",
			ResourceID:  taskID.String(),
			Status:      "success",
			Message:     fmt.Sprintf("taskId=%s productId=%s", taskID, taskRow.ProductID),
		})
	}

	client, _, err := s.Shops.DouyinClientForShopContext(ctx, taskRow.ShopID, taskRow.CreatedBy)
	if err != nil {
		code := inferDouyinPublishErrorCode(err)
		return fail(code, err.Error(), douyinErrRetryable(err), "", nil)
	}

	xctx, cancel := context.WithTimeout(ctx, s.execTimeout())
	defer cancel()

	var res *platformdouyin.PlatformProductResult
	recoveredRes, recovered, recErr := tryRecoverDouyinDraftFromPlatform(xctx, client, taskRow.ShopID.String(), taskRow.ProductID.String())
	if recErr != nil {
		code := inferDouyinPublishErrorCode(recErr)
		return fail(code, recErr.Error(), douyinErrRetryable(recErr), "", nil)
	}
	if recovered {
		res = recoveredRes
		if s.OpLog != nil {
			_ = s.OpLog.WriteBackground(ctx, operationlog.WriteOpts{
				AdminUserID: taskRow.CreatedBy,
				Action:      "douyin.product.draft.recover",
				Resource:    "product_publish_task",
				ResourceID:  taskID.String(),
				Status:      "success",
				Message:     fmt.Sprintf("taskId=%s platformProductId=%s recovered via product.detail", taskID, res.PlatformProductID),
			})
		}
	} else {
		publishVersion := snap.MappingHash
		if publishVersion == "" {
			publishVersion = taskID.String()
		}
		idemKey := idempotency.DouyinProductDraftCreate(taskRow.ShopID.String(), taskRow.ProductID.String(), publishVersion)
		idemJob, replayRes, acqErr := s.acquirePublishIdempotency(ctx, idemKey, []byte(idemKey), "douyin-draft-create")
		if acqErr != nil {
			return fail(inferDouyinPublishErrorCode(acqErr), acqErr.Error(), true, "", nil)
		}
		if replayRes != nil && replayRes.Replay && strings.TrimSpace(replayRes.ResourceID) != "" {
			res = &platformdouyin.PlatformProductResult{PlatformProductID: replayRes.ResourceID, PlatformStatus: "draft"}
		} else {
			var pubErr error
			res, pubErr = client.CreateProductDraft(xctx, taskRow.ShopID.String(), buildRes.APIReq)
			if pubErr != nil {
				code := inferDouyinPublishErrorCode(pubErr)
				var de *platformdouyin.Error
				raw := map[string]any{}
				retryable := false
				requestID := ""
				if errors.As(pubErr, &de) {
					retryable = de.Retryable
					requestID = de.RequestID
					raw = map[string]any{"platformCode": de.PlatformCode, "platformMessage": de.PlatformMessage}
					if de.UnknownResult || de.Code == platformdouyin.CodeDouyinRequestTimeout || de.Code == platformdouyin.CodeDouyinUnknownResult {
						// unknown_result: try recover before failing; do not auto-recreate
						if recoveredRes2, recovered2, recErr2 := tryRecoverDouyinDraftFromPlatform(xctx, client, taskRow.ShopID.String(), taskRow.ProductID.String()); recErr2 == nil && recovered2 && recoveredRes2 != nil {
							res = recoveredRes2
							if idemJob != nil {
								_ = s.completePublishIdempotency(ctx, idemJob, map[string]string{"platformProductId": res.PlatformProductID}, res.PlatformProductID)
							}
						} else {
							if idemJob != nil {
								s.failPublishIdempotency(ctx, idemJob, platformdouyin.CodeDouyinUnknownResult, false)
							}
							s.markDouyinStale(ctx, taskID, platformdouyin.CodeDouyinUnknownResult, platformdouyin.RecoveryResultUnknown, taskRow.CreatedBy)
							return fail(platformdouyin.CodeDouyinUnknownResult, "抖店草稿创建结果未知，请先回查平台草稿箱", false, requestID, raw)
						}
					} else {
						if idemJob != nil {
							s.failPublishIdempotency(ctx, idemJob, code, retryable)
						}
						return fail(code, pubErr.Error(), retryable, requestID, raw)
					}
				} else {
					if idemJob != nil {
						s.failPublishIdempotency(ctx, idemJob, code, retryable)
					}
					return fail(code, pubErr.Error(), retryable, requestID, raw)
				}
			} else if idemJob != nil && res != nil {
				_ = s.completePublishIdempotency(ctx, idemJob, map[string]string{"platformProductId": res.PlatformProductID}, res.PlatformProductID)
			}
		}
	}
	if res == nil || strings.TrimSpace(res.PlatformProductID) == "" {
		return fail(ErrorDouyinCreateProductFailed, "platform did not return product id", true, "", nil)
	}

	return s.completeDouyinDraftSuccess(ctx, taskRow, taskID, workerID, claim, snap, buildRes, res)
}

func (s *Service) completeDouyinDraftSuccess(ctx context.Context, taskRow *ProductPublishTask, taskID uuid.UUID, workerID string, claim *tasklease.ClaimResult, snap douyinDraftSnapshot, buildRes *DouyinPayloadBuildResult, res *platformdouyin.PlatformProductResult) error {
	fin := time.Now().UTC()
	outSnap := map[string]any{
		"platformProductId": res.PlatformProductID,
		"platformStatus":    res.PlatformStatus,
		"requestId":         res.RequestID,
	}
	rawOut, _ := json.Marshal(outSnap)
	updates := map[string]any{
		"status":              TaskSuccess,
		"publish_status":      StatusDraftCreated,
		"platform_product_id": res.PlatformProductID,
		"request_id":          res.RequestID,
		"retryable":           false,
		"error_code":          "",
		"error_message":       "",
		"finished_at":         &fin,
		"output":              datatypes.JSON(rawOut),
		"platform_result":     datatypes.JSON(rawOut),
	}
	rd, _ := json.Marshal(sanitizeRawErrorMap(res.Raw))
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates["locked_by"] = nil
		updates["locked_until"] = nil
		updates["updated_at"] = fin
		taskUpdate := tx.Model(&ProductPublishTask{}).Where("id = ? AND tenant_id = ?", taskID, taskRow.TenantID)
		if claim != nil && strings.TrimSpace(workerID) != "" {
			taskUpdate = taskUpdate.Where("status = ? AND locked_by = ? AND execution_id = ? AND lock_version = ?",
				TaskRunning, workerID, claim.ExecutionID.String(), claim.LeaseVersion)
		}
		result := taskUpdate.Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return tasklease.ErrLeaseLost
		}
		publicationUpdate := tx.Model(&ProductPublication{}).Where("id = ? AND tenant_id = ? AND product_id = ? AND shop_id = ? AND platform = ?",
			snap.PublicationID, taskRow.TenantID, taskRow.ProductID, taskRow.ShopID, "douyin_shop").Updates(map[string]any{
			"external_product_id":  res.PlatformProductID,
			"status":               StatusDraft,
			"publish_status":       StatusDraftCreated,
			"publish_mode":         snap.PublishMode,
			"platform_category_id": buildRes.Payload.CategoryLeafID,
			"raw_data":             datatypes.JSON(rd),
			"last_synced_at":       &fin,
			"updated_at":           fin,
		})
		if publicationUpdate.Error != nil {
			return publicationUpdate.Error
		}
		if publicationUpdate.RowsAffected != 1 {
			return fmt.Errorf("bound publication not found")
		}

		mapping := snap.FrozenMapping
		if mapping == nil {
			return ErrDouyinOperationTaskRequired
		}
		if err := tx.Where("publication_id = ?", snap.PublicationID).Delete(&ProductPublicationSKU{}).Error; err != nil {
			return err
		}
		skuMap := map[string]platformdouyin.SKUMapping{}
		for _, sm := range res.SKUMappings {
			if sm.OuterSKUID != "" {
				skuMap[sm.OuterSKUID] = sm
			}
		}
		for _, sku := range mapping.SKUs {
			row := ProductPublicationSKU{
				PublicationID: snap.PublicationID,
				SKUCode:       strings.TrimSpace(sku.Name),
				Price:         &sku.Price,
				Stock:         sku.Stock,
			}
			if uid, parseErr := uuid.Parse(strings.TrimSpace(sku.LocalSkuID)); parseErr == nil {
				row.ProductSKUID = &uid
			}
			if sm, ok := skuMap[sku.LocalSkuID]; ok {
				row.ExternalSKUID = strings.TrimSpace(sm.PlatformSKUID)
			}
			rdm, marshalErr := json.Marshal(map[string]any{"outerSkuId": sku.LocalSkuID})
			if marshalErr != nil {
				return marshalErr
			}
			row.RawData = datatypes.JSON(rdm)
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	if s.OpLog != nil {
		_ = s.OpLog.WriteBackground(ctx, operationlog.WriteOpts{
			AdminUserID: taskRow.CreatedBy,
			Action:      "douyin.product.draft.create.success",
			Resource:    "product_publish_task",
			ResourceID:  taskID.String(),
			Status:      "success",
			Message:     fmt.Sprintf("taskId=%s platformProductId=%s", taskID, res.PlatformProductID),
		})
	}
	douyinmetrics.RecordProductDraftCreate(true)
	if snap.ExecutionAttemptID != nil && *snap.ExecutionAttemptID != uuid.Nil && s.OperationResults != nil {
		meta, _ := json.Marshal(map[string]any{"platformStatus": res.PlatformStatus, "requestId": res.RequestID})
		if err := s.OperationResults.MarkSucceeded(ctx, *snap.ExecutionAttemptID, taskID, res.PlatformProductID, res.RequestID, datatypes.JSON(meta)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) douyinBuildResultFromSnapshot(ctx context.Context, task *ProductPublishTask, snap douyinDraftSnapshot) (*DouyinPayloadBuildResult, error) {
	if snap.FrozenRequest != nil {
		return &DouyinPayloadBuildResult{Payload: frozenRequestDisplayPayload(*snap.FrozenRequest), APIReq: *snap.FrozenRequest}, nil
	}
	return BuildDouyinProductPayload(ctx, s.DB, task.ProductID, snap.ConfigID)
}

func frozenRequestDisplayPayload(req platformdouyin.CreateProductDraftRequest) *DouyinProductPayload {
	return &DouyinProductPayload{
		OuterProductID: req.OuterProductID, Name: req.Name, CategoryLeafID: req.CategoryLeafID,
		Pic: req.Pic, Description: req.Description, ProductFormat: req.ProductFormat,
		SpecInfo: req.SpecInfo, SpecPricesV2: specPriceMapsToAny(req.SpecPricesV2),
		Commit: false, StartSaleType: 1, FreightID: req.FreightID, Mobile: req.Mobile,
	}
}

func specPriceMapsToAny(values []map[string]any) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func parseDouyinDraftSnapshot(raw datatypes.JSON) (douyinDraftSnapshot, bool) {
	var snap douyinDraftSnapshot
	if len(raw) == 0 {
		return snap, false
	}
	if err := json.Unmarshal(raw, &snap); err != nil || snap.PublicationID == uuid.Nil {
		return snap, false
	}
	return snap, true
}

func inferDouyinPublishErrorCode(err error) string {
	if err == nil {
		return ErrorUnknownDouyinPublish
	}
	var de *platformdouyin.Error
	if errors.As(err, &de) && de.Code != "" {
		return de.Code
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "not authorized") || strings.Contains(msg, "auth expired"):
		return platformdouyin.CodeDouyinStoreNotAuthorized
	case strings.Contains(msg, "category"):
		return platformdouyin.CodeDouyinCategoryMissing
	case strings.Contains(msg, "attribute") || strings.Contains(msg, "required"):
		return platformdouyin.CodeDouyinRequiredAttrMissing
	case strings.Contains(msg, "image"):
		return platformdouyin.CodeDouyinMainImageNotUploaded
	case strings.Contains(msg, "rate"):
		return platformdouyin.CodeDouyinRateLimited
	case strings.Contains(msg, "permission"):
		return platformdouyin.CodeDouyinPermissionDenied
	default:
		return ErrorDouyinCreateProductFailed
	}
}

func douyinErrRetryable(err error) bool {
	var de *platformdouyin.Error
	if errors.As(err, &de) {
		return de.Retryable
	}
	return false
}
