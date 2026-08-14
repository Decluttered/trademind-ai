package productioncontrol

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/pkg/adminperm"
	"github.com/trademind-ai/trademind/backend/internal/pkg/response"
)

type Handler struct{ Service *Service }

type allowlistRequest struct {
	ShopID           uuid.UUID `json:"shopId"`
	Enabled          bool      `json:"enabled"`
	ExpectedRevision int       `json:"expectedRevision"`
}

type grayDraftRequest struct {
	ShopID           uuid.UUID `json:"shopId"`
	MaxSKU           int       `json:"maxSku"`
	ExpectedRevision int       `json:"expectedRevision"`
}

type grayActionRequest struct {
	ExpectedRevision int `json:"expectedRevision"`
}

type grayApprovalRequest struct {
	Role             string `json:"role"`
	ExpectedRevision int    `json:"expectedRevision"`
}

func (h *Handler) actor(c *gin.Context, permission string, shopID uuid.UUID) (Actor, *adminperm.Principal, bool) {
	if h == nil || h.Service == nil || h.Service.DB == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeServiceUnavailable, "生产控制服务不可用")
		return Actor{}, nil, false
	}
	tenantID, err := adminperm.TenantIDFromGin(c)
	if err != nil {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或租户上下文无效")
		return Actor{}, nil, false
	}
	principal, err := adminperm.LoadPrincipal(c, h.Service.DB)
	if err != nil || principal == nil || principal.Disabled || !principal.Can(permission) {
		response.Fail(c, http.StatusForbidden, response.CodeForbidden, "无权执行此操作")
		return Actor{}, nil, false
	}
	if shopID != uuid.Nil && !principal.CanOperateStore(shopID) {
		response.Fail(c, http.StatusForbidden, response.CodeStorePermissionDenied, "无权操作该店铺")
		return Actor{}, nil, false
	}
	requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
	if requestID == "" {
		requestID = uuid.NewString()
	}
	return Actor{TenantID: tenantID, UserID: principal.UserID, RequestID: requestID}, principal, true
}

func bindJSON(c *gin.Context, target any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024)
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

func (h *Handler) Status(c *gin.Context) {
	actor, principal, ok := h.statusActor(c)
	if !ok {
		return
	}
	out, err := h.Service.Status(c.Request.Context(), actor.TenantID, principal.AllowedStoreIDs())
	if err != nil {
		respondControlError(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) statusActor(c *gin.Context) (Actor, *adminperm.Principal, bool) {
	if h == nil || h.Service == nil || h.Service.DB == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeServiceUnavailable, "生产控制服务不可用")
		return Actor{}, nil, false
	}
	tenantID, err := adminperm.TenantIDFromGin(c)
	if err != nil {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或租户上下文无效")
		return Actor{}, nil, false
	}
	principal, err := adminperm.LoadPrincipal(c, h.Service.DB)
	if err != nil || principal == nil || principal.Disabled || (!principal.Can(adminperm.PermInventorySyncRead) && !principal.Can(adminperm.PermOperationTaskAuditRead)) {
		response.Fail(c, http.StatusForbidden, response.CodeForbidden, "无权查看生产运行状态")
		return Actor{}, nil, false
	}
	requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
	if requestID == "" {
		requestID = uuid.NewString()
	}
	return Actor{TenantID: tenantID, UserID: principal.UserID, RequestID: requestID}, principal, true
}

func (h *Handler) UpdateSwitches(c *gin.Context) {
	var req SwitchUpdate
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "请求参数无效")
		return
	}
	actor, _, ok := h.actor(c, adminperm.PermConfigManage, uuid.Nil)
	if !ok {
		return
	}
	out, err := h.Service.UpdateSwitches(c.Request.Context(), actor, req)
	if err != nil {
		respondControlError(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) SetAllowlist(c *gin.Context) {
	var req allowlistRequest
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "请求参数无效")
		return
	}
	actor, _, ok := h.actor(c, adminperm.PermConfigManage, req.ShopID)
	if !ok {
		return
	}
	out, err := h.Service.SetAllowlist(c.Request.Context(), actor, req.ShopID, req.Enabled, req.ExpectedRevision)
	if err != nil {
		respondControlError(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) SaveGrayDraft(c *gin.Context) {
	var req grayDraftRequest
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "请求参数无效")
		return
	}
	actor, _, ok := h.actor(c, adminperm.PermConfigManage, req.ShopID)
	if !ok {
		return
	}
	out, err := h.Service.SaveGrayDraft(c.Request.Context(), actor, req.ShopID, req.MaxSKU, req.ExpectedRevision)
	if err != nil {
		respondControlError(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) PauseGray(c *gin.Context) { h.grayAction(c, "pause") }
func (h *Handler) StopGray(c *gin.Context)  { h.grayAction(c, "stop") }

func (h *Handler) ApproveGray(c *gin.Context) {
	var req grayApprovalRequest
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid request")
		return
	}
	actor, principal, ok := h.actor(c, adminperm.PermConfigManage, uuid.Nil)
	if !ok {
		return
	}
	if principal == nil || !principal.IsAdmin() {
		response.Fail(c, http.StatusForbidden, response.CodeForbidden, "admin approval required")
		return
	}
	out, err := h.Service.ApproveGray(c.Request.Context(), actor, req.Role, req.ExpectedRevision)
	if err != nil {
		respondControlError(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) ActivateGray(c *gin.Context) {
	var req grayActionRequest
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid request")
		return
	}
	actor, principal, ok := h.actor(c, adminperm.PermConfigManage, uuid.Nil)
	if !ok {
		return
	}
	if principal == nil || !principal.IsAdmin() {
		response.Fail(c, http.StatusForbidden, response.CodeForbidden, "admin activation required")
		return
	}
	out, err := h.Service.ActivateGray(c.Request.Context(), actor, req.ExpectedRevision)
	if err != nil {
		respondControlError(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) grayAction(c *gin.Context, action string) {
	var req grayActionRequest
	if err := bindJSON(c, &req); err != nil || req.ExpectedRevision < 1 {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "请求参数无效")
		return
	}
	actor, _, ok := h.actor(c, adminperm.PermConfigManage, uuid.Nil)
	if !ok {
		return
	}
	out, err := h.Service.PauseOrStopGray(c.Request.Context(), actor, action, req.ExpectedRevision)
	if err != nil {
		respondControlError(c, err)
		return
	}
	response.OK(c, out)
}

func respondControlError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrScopeExceeded):
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "范围超过首版 1 个租户、1 个店铺或 100 个规格的限制")
	case errors.Is(err, ErrRevisionConflict):
		response.Fail(c, http.StatusConflict, response.CodeBadRequest, "配置版本冲突，请刷新后重试")
	case errors.Is(err, ErrBlocked):
		response.Fail(c, http.StatusForbidden, response.CodeForbidden, "当前安全边界已阻断该操作")
	default:
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalError, "生产控制操作失败")
	}
}

func Register(group *gin.RouterGroup, h *Handler) {
	api := group.Group("/p10")
	api.GET("/status", h.Status)
	api.PUT("/controls/kill-switches", h.UpdateSwitches)
	api.PUT("/controls/allowlist", h.SetAllowlist)
	api.PUT("/gray", h.SaveGrayDraft)
	api.POST("/gray/approve", h.ApproveGray)
	api.POST("/gray/activate", h.ActivateGray)
	api.POST("/gray/pause", h.PauseGray)
	api.POST("/gray/stop", h.StopGray)
}
