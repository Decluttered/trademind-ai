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
	"github.com/trademind-ai/trademind/backend/internal/modules/operationlog"
	"github.com/trademind-ai/trademind/backend/internal/modules/product"
	platformdouyin "github.com/trademind-ai/trademind/backend/internal/providers/platform/douyinshop"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// DouyinPlatformSKUCandidate is one platform SKU line for manual binding UI.
type DouyinPlatformSKUCandidate struct {
	PlatformSKUID           string     `json:"platformSkuId"`
	SpecName                string     `json:"specName,omitempty"`
	PriceYuan               float64    `json:"priceYuan,omitempty"`
	Stock                   int        `json:"stock,omitempty"`
	BoundToPublicationSKUID *uuid.UUID `json:"boundToPublicationSkuId,omitempty"`
}

// ManualBindDouyinSKU binds one publication SKU to a chosen platform SKU.
func (s *Service) ManualBindDouyinSKU(c *gin.Context, publicationSKUID uuid.UUID, body DouyinManualBindBody, adminID *uuid.UUID) (*DouyinSKUBindingRow, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("%s: product publish unavailable", platformdouyin.CodeDouyinSKUManualBindFailed)
	}
	ctx := c.Request.Context()
	platformSKUID := strings.TrimSpace(body.PlatformSKUID)
	if platformSKUID == "" {
		return nil, fmt.Errorf("%s: platform sku id required", platformdouyin.CodeDouyinPlatformSKUIDMissing)
	}

	psku, pub, err := s.loadDouyinPublicationSKU(ctx, publicationSKUID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureStoreOperate(c, pub.ShopID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(pub.ExternalProductID) == "" {
		return nil, fmt.Errorf("%s: platform product id missing", platformdouyin.CodeDouyinProductNotBound)
	}

	if err := s.checkDouyinSKUBindingConflict(ctx, pub.ID, publicationSKUID, platformSKUID); err != nil {
		if s.OpLog != nil {
			_ = s.OpLog.Write(c, operationlog.WriteOpts{
				AdminUserID: adminID,
				Action:      "douyin.sku.binding.conflict",
				Resource:    "product_publication_sku",
				ResourceID:  publicationSKUID.String(),
				Status:      "failed",
				Message: fmt.Sprintf("publicationId=%s platformSkuId=%s err=%s",
					pub.ID, platformSKUID, bindingTruncateMsg(err.Error())),
			})
		}
		return nil, err
	}

	oldExt := strings.TrimSpace(psku.ExternalSKUID)
	now := time.Now().UTC()
	updates := map[string]any{
		"external_sku_id": platformSKUID,
		"bind_status":     BindStatusBound,
		"bind_confidence": 100,
		"bind_message":    "手动绑定",
		"last_synced_at":  &now,
		"updated_at":      now,
	}
	if err := s.DB.WithContext(ctx).Model(&ProductPublicationSKU{}).
		Where("id = ? AND publication_id = ?", publicationSKUID, pub.ID).
		Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("%s: %s", platformdouyin.CodeDouyinSKUManualBindFailed, err.Error())
	}

	if s.OpLog != nil {
		msg := fmt.Sprintf("publicationId=%s platformSkuId=%s bindReason=%s",
			pub.ID, platformSKUID, strings.TrimSpace(body.BindReason))
		if oldExt != "" && oldExt != platformSKUID {
			msg += fmt.Sprintf(" previousPlatformSkuId=%s", oldExt)
		}
		if name := strings.TrimSpace(body.PlatformSKUName); name != "" {
			msg += fmt.Sprintf(" platformSkuName=%s", bindingTruncateMsg(name))
		}
		_ = s.OpLog.Write(c, operationlog.WriteOpts{
			AdminUserID: adminID,
			Action:      "douyin.sku.binding.manual_bind",
			Resource:    "product_publication_sku",
			ResourceID:  publicationSKUID.String(),
			Status:      "success",
			Message:     msg,
		})
	}
	douyinmetrics.RecordSKUManualBound()

	row, err := s.buildDouyinBindingRow(ctx, pub, publicationSKUID)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", platformdouyin.CodeDouyinSKUManualBindFailed, err.Error())
	}
	return row, nil
}

// UnbindDouyinSKU clears manual/platform binding for one publication SKU.
func (s *Service) UnbindDouyinSKU(c *gin.Context, publicationSKUID uuid.UUID, body DouyinManualUnbindBody, adminID *uuid.UUID) (*DouyinSKUBindingRow, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("%s: product publish unavailable", platformdouyin.CodeDouyinSKUManualUnbindFailed)
	}
	ctx := c.Request.Context()
	psku, pub, err := s.loadDouyinPublicationSKU(ctx, publicationSKUID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureStoreOperate(c, pub.ShopID); err != nil {
		return nil, err
	}

	oldExt := strings.TrimSpace(psku.ExternalSKUID)
	now := time.Now().UTC()
	updates := map[string]any{
		"external_sku_id": "",
		"bind_status":     BindStatusUnmatched,
		"bind_confidence": 0,
		"bind_message":    "已手动解除绑定",
		"last_synced_at":  &now,
		"updated_at":      now,
	}
	if err := s.DB.WithContext(ctx).Model(&ProductPublicationSKU{}).
		Where("id = ? AND publication_id = ?", publicationSKUID, pub.ID).
		Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("%s: %s", platformdouyin.CodeDouyinSKUManualUnbindFailed, err.Error())
	}

	if s.OpLog != nil {
		reason := strings.TrimSpace(body.Reason)
		if reason == "" {
			reason = "manual_unbind"
		}
		msg := fmt.Sprintf("publicationId=%s reason=%s", pub.ID, reason)
		if oldExt != "" {
			msg += fmt.Sprintf(" previousPlatformSkuId=%s", oldExt)
		}
		_ = s.OpLog.Write(c, operationlog.WriteOpts{
			AdminUserID: adminID,
			Action:      "douyin.sku.binding.manual_unbind",
			Resource:    "product_publication_sku",
			ResourceID:  publicationSKUID.String(),
			Status:      "success",
			Message:     msg,
		})
	}

	row, err := s.buildDouyinBindingRow(ctx, pub, publicationSKUID)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", platformdouyin.CodeDouyinSKUManualUnbindFailed, err.Error())
	}
	return row, nil
}

func (s *Service) loadDouyinPublicationSKU(ctx context.Context, publicationSKUID uuid.UUID) (*ProductPublicationSKU, *ProductPublication, error) {
	var psku ProductPublicationSKU
	if err := s.DB.WithContext(ctx).First(&psku, "id = ?", publicationSKUID).Error; err != nil {
		return nil, nil, err
	}
	pub, err := s.loadDouyinPublication(ctx, psku.PublicationID)
	if err != nil {
		return nil, nil, err
	}
	return &psku, pub, nil
}

func (s *Service) checkDouyinSKUBindingConflict(ctx context.Context, publicationID, excludeSKUID uuid.UUID, platformSKUID string) error {
	platformSKUID = strings.TrimSpace(platformSKUID)
	if platformSKUID == "" {
		return fmt.Errorf("%s: platform sku id required", platformdouyin.CodeDouyinPlatformSKUIDMissing)
	}
	var other ProductPublicationSKU
	err := s.DB.WithContext(ctx).
		Where("publication_id = ? AND id <> ? AND external_sku_id = ?", publicationID, excludeSKUID, platformSKUID).
		First(&other).Error
	if err == nil {
		return fmt.Errorf("%s: platform sku already bound to another local spec", platformdouyin.CodeDouyinSKUBindingConflict)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return fmt.Errorf("check douyin sku binding conflict: %w", err)
}

func (s *Service) buildDouyinBindingRow(ctx context.Context, pub *ProductPublication, publicationSKUID uuid.UUID) (*DouyinSKUBindingRow, error) {
	localRows, err := s.loadLocalSKUsForBinding(ctx, pub.ID, pub.ProductID)
	if err != nil {
		return nil, err
	}
	var local *localSKUForBinding
	for i := range localRows {
		if localRows[i].PublicationSKUID == publicationSKUID {
			local = &localRows[i]
			break
		}
	}
	if local == nil {
		return nil, fmt.Errorf("publication sku not found")
	}
	var row ProductPublicationSKU
	if err := s.DB.WithContext(ctx).First(&row, "id = ? AND publication_id = ?", publicationSKUID, pub.ID).Error; err != nil {
		return nil, err
	}
	out := DouyinSKUBindingRow{
		PublicationSKUID: row.ID,
		ProductSKUID:     row.ProductSKUID,
		SKUCode:          firstNonEmpty(row.SKUCode, local.SKUCode),
		SpecName:         local.SpecName,
		ExternalSKUID:    strings.TrimSpace(row.ExternalSKUID),
		PlatformSKUName:  platformSKUNameFromCache(pub, strings.TrimSpace(row.ExternalSKUID)),
		BindStatus:       strings.TrimSpace(row.BindStatus),
		BindConfidence:   row.BindConfidence,
		BindMessage:      row.BindMessage,
		LastSyncedAt:     row.LastSyncedAt,
		Price:            row.Price,
		Stock:            localStockForBinding(ctx, s.DB, pub.ProductID, row.ProductSKUID),
	}
	return &out, nil
}

func localStockForBinding(ctx context.Context, db *gorm.DB, productID uuid.UUID, productSKUID *uuid.UUID) *int {
	if db == nil || productSKUID == nil {
		return nil
	}
	var sku product.ProductSKU
	if err := db.WithContext(ctx).First(&sku, "id = ? AND product_id = ?", *productSKUID, productID).Error; err != nil {
		return nil
	}
	return sku.Stock
}

func platformSKUNameFromCache(pub *ProductPublication, platformSKUID string) string {
	if pub == nil || platformSKUID == "" {
		return ""
	}
	for _, c := range platformSKUsFromPublicationRaw(pub.RawData) {
		if strings.TrimSpace(c.PlatformSKUID) == platformSKUID {
			return strings.TrimSpace(c.SpecName)
		}
	}
	return ""
}

func platformSKUsFromPublicationRaw(raw datatypes.JSON) []DouyinPlatformSKUCandidate {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	items, ok := m["platformSkus"].([]any)
	if !ok {
		return nil
	}
	out := make([]DouyinPlatformSKUCandidate, 0, len(items))
	for _, it := range items {
		obj, ok := it.(map[string]any)
		if !ok {
			continue
		}
		c := DouyinPlatformSKUCandidate{
			PlatformSKUID: strings.TrimSpace(fmt.Sprint(obj["platformSkuId"])),
			SpecName:      strings.TrimSpace(fmt.Sprint(obj["specName"])),
		}
		if v, ok := obj["priceYuan"].(float64); ok {
			c.PriceYuan = v
		}
		if v, ok := obj["stock"].(float64); ok {
			c.Stock = int(v)
		}
		if c.PlatformSKUID != "" {
			out = append(out, c)
		}
	}
	return out
}

func platformSKUsToCandidates(skus []platformdouyin.PlatformProductSKU) []DouyinPlatformSKUCandidate {
	out := make([]DouyinPlatformSKUCandidate, 0, len(skus))
	for _, p := range skus {
		out = append(out, DouyinPlatformSKUCandidate{
			PlatformSKUID: strings.TrimSpace(p.PlatformSKUID),
			SpecName:      strings.TrimSpace(p.SpecName),
			PriceYuan:     p.PriceYuan,
			Stock:         p.Stock,
		})
	}
	return out
}

func platformSKUsToRaw(skus []platformdouyin.PlatformProductSKU) []map[string]any {
	out := make([]map[string]any, 0, len(skus))
	for _, p := range skus {
		out = append(out, map[string]any{
			"platformSkuId": strings.TrimSpace(p.PlatformSKUID),
			"specName":      strings.TrimSpace(p.SpecName),
			"priceYuan":     p.PriceYuan,
			"stock":         p.Stock,
		})
	}
	return out
}

func mergePublicationRawPlatformSKUs(existing datatypes.JSON, skus []platformdouyin.PlatformProductSKU) datatypes.JSON {
	var m map[string]any
	if len(existing) > 0 && string(existing) != "null" {
		_ = json.Unmarshal(existing, &m)
	}
	if m == nil {
		m = map[string]any{}
	}
	m["platformSkus"] = platformSKUsToRaw(skus)
	b, _ := json.Marshal(m)
	return datatypes.JSON(b)
}

// DouyinInventorySyncReady reports whether all publication SKUs are safe for inventory sync.
func DouyinInventorySyncReady(rows []DouyinSKUBindingRow) (ready bool, reason string) {
	if len(rows) == 0 {
		return false, "该商品还没有刊登 SKU 映射，暂时不能同步库存。"
	}
	for _, r := range rows {
		st := strings.TrimSpace(strings.ToLower(r.BindStatus))
		ext := strings.TrimSpace(r.ExternalSKUID)
		switch st {
		case BindStatusAmbiguous:
			return false, "该商品仍有规格存在多个候选抖店 SKU，请先人工确认绑定后再同步库存。"
		case BindStatusUnmatched:
			return false, "该商品仍有规格未绑定抖店 SKU，暂时不能同步库存。请先完成 SKU 绑定。"
		case BindStatusFailed:
			return false, "该商品仍有规格 SKU 绑定校准失败，请重新校准或手动绑定后再同步库存。"
		case BindStatusBound, BindStatusSkipped:
			if ext == "" {
				return false, "该商品仍有规格未绑定抖店 SKU，暂时不能同步库存。请先完成 SKU 绑定。"
			}
		default:
			if ext == "" {
				return false, "该商品仍有规格未绑定抖店 SKU，暂时不能同步库存。请先完成 SKU 绑定。"
			}
		}
	}
	return true, ""
}

// ValidateDouyinSKUBindingForInventorySync checks one publication SKU row before inventory push.
func ValidateDouyinSKUBindingForInventorySync(platform string, externalSKUID, bindStatus string) error {
	pl := strings.TrimSpace(strings.ToLower(platform))
	if pl != "douyin_shop" {
		return nil
	}
	ext := strings.TrimSpace(externalSKUID)
	st := strings.TrimSpace(strings.ToLower(bindStatus))
	if ext == "" {
		return fmt.Errorf("%s: external sku id missing; please bind douyin sku first", platformdouyin.CodeDouyinSKUBindingRequired)
	}
	switch st {
	case BindStatusAmbiguous:
		return fmt.Errorf("%s: ambiguous sku binding requires manual confirmation", platformdouyin.CodeDouyinSKUBindingAmbiguous)
	case BindStatusUnmatched:
		return fmt.Errorf("%s: sku binding unmatched; please bind douyin sku first", platformdouyin.CodeDouyinSKUBindingRequired)
	case BindStatusFailed:
		return fmt.Errorf("%s: sku binding calibration failed; please recheck or bind manually", platformdouyin.CodeDouyinSKUBindingRequired)
	}
	return nil
}

func annotatePlatformSKUCandidates(cands []DouyinPlatformSKUCandidate, rows []DouyinSKUBindingRow) []DouyinPlatformSKUCandidate {
	bound := map[string]uuid.UUID{}
	for _, r := range rows {
		ext := strings.TrimSpace(r.ExternalSKUID)
		if ext != "" {
			bound[ext] = r.PublicationSKUID
		}
	}
	out := make([]DouyinPlatformSKUCandidate, 0, len(cands))
	for _, c := range cands {
		cp := c
		if id, ok := bound[strings.TrimSpace(c.PlatformSKUID)]; ok {
			idCopy := id
			cp.BoundToPublicationSKUID = &idCopy
		}
		out = append(out, cp)
	}
	return out
}
