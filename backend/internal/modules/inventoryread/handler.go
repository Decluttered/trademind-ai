package inventoryread

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/modules/productioncontrol"
	"github.com/trademind-ai/trademind/backend/internal/pkg/adminperm"
	"github.com/trademind-ai/trademind/backend/internal/pkg/response"
)

type Handler struct{ Service *ManualReadService }

type createRunRequest struct {
	ShopID uuid.UUID `json:"shopId"`
}

type rerunRequest struct {
	ExpectedRevision int `json:"expectedRevision"`
}

func (h *Handler) actor(c *gin.Context, shopID uuid.UUID) (int64, uuid.UUID, bool) {
	if h == nil || h.Service == nil || h.Service.DB == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeServiceUnavailable, "只读库存服务不可用")
		return 0, uuid.Nil, false
	}
	tenantID, err := adminperm.TenantIDFromGin(c)
	if err != nil {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或租户上下文无效")
		return 0, uuid.Nil, false
	}
	principal, err := adminperm.LoadPrincipal(c, h.Service.DB)
	if err != nil || principal == nil || principal.Disabled || !principal.Can(adminperm.PermInventorySyncRun) || !principal.CanOperateStore(shopID) {
		response.Fail(c, http.StatusForbidden, response.CodeForbidden, "无权发起该店铺的只读库存同步")
		return 0, uuid.Nil, false
	}
	return tenantID, principal.UserID, true
}

func bindRunJSON(c *gin.Context, target any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8*1024)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func requestIdentity(c *gin.Context) (string, string, bool) {
	requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
	if requestID == "" {
		requestID = uuid.NewString()
	}
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if len(key) < 8 || len(key) > 200 {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "Idempotency-Key 缺失或无效")
		return "", "", false
	}
	sum := sha256.Sum256([]byte(key))
	return requestID, hex.EncodeToString(sum[:]), true
}

func (h *Handler) Create(c *gin.Context) {
	var req createRunRequest
	if err := bindRunJSON(c, &req); err != nil || req.ShopID == uuid.Nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "请求参数无效")
		return
	}
	tenantID, actorID, ok := h.actor(c, req.ShopID)
	if !ok {
		return
	}
	requestID, idem, ok := requestIdentity(c)
	if !ok {
		return
	}
	out, err := h.Service.Run(c.Request.Context(), RunInput{TenantID: tenantID, ActorID: actorID, ShopID: req.ShopID, RequestID: requestID, IdempotencyHash: idem})
	if err != nil {
		respondRunError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, response.CodeOK, "ok", out)
}

func (h *Handler) Rerun(c *gin.Context) {
	runID, err := uuid.Parse(strings.TrimSpace(c.Param("runId")))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "运行编号无效")
		return
	}
	var req rerunRequest
	if err := bindRunJSON(c, &req); err != nil || req.ExpectedRevision < 1 {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "请求参数无效")
		return
	}
	tenantID, err := adminperm.TenantIDFromGin(c)
	if err != nil {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或租户上下文无效")
		return
	}
	var source struct {
		TenantID         int64
		ShopConnectionID uuid.UUID
	}
	if err := h.Service.DB.WithContext(c.Request.Context()).Table("inventory_sync_runs").Select("tenant_id", "shop_connection_id").Where("tenant_id = ? AND id = ?", tenantID, runID).Scan(&source).Error; err != nil || source.ShopConnectionID == uuid.Nil {
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "运行记录不存在")
		return
	}
	actorTenantID, actorID, ok := h.actor(c, source.ShopConnectionID)
	if !ok || actorTenantID != source.TenantID {
		return
	}
	requestID, idem, ok := requestIdentity(c)
	if !ok {
		return
	}
	out, err := h.Service.Run(c.Request.Context(), RunInput{TenantID: actorTenantID, ActorID: actorID, ShopID: source.ShopConnectionID, RequestID: requestID, IdempotencyHash: idem, SourceRunID: runID, ExpectedRevision: req.ExpectedRevision})
	if err != nil {
		respondRunError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, response.CodeOK, "ok", out)
}

func respondRunError(c *gin.Context, err error) {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) && providerErr != nil {
		switch providerErr.Code {
		case ErrorUnauthorized, ErrorCredentialExpired:
			response.Fail(c, http.StatusForbidden, response.CodeForbidden, "当前安全边界、凭证或店铺范围不允许真实只读调用")
		case ErrorRateLimited:
			response.Fail(c, http.StatusTooManyRequests, response.CodeServiceUnavailable, "平台限流，请稍后人工重试")
		case ErrorInvalidRequest:
			response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "只读库存请求无效")
		default:
			response.Fail(c, http.StatusBadGateway, response.CodeServiceUnavailable, "平台只读调用失败")
		}
		return
	}
	if errors.Is(err, productioncontrol.ErrBlocked) {
		response.Fail(c, http.StatusForbidden, response.CodeForbidden, "当前安全边界已阻断真实只读调用")
		return
	}
	response.Fail(c, http.StatusInternalServerError, response.CodeInternalError, "只读库存同步失败")
}

func Register(group *gin.RouterGroup, h *Handler) {
	api := group.Group("/p10/inventory-read")
	api.POST("/runs", h.Create)
	api.POST("/runs/:runId/rerun", h.Rerun)
}
