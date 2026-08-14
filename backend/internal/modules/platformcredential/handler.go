package platformcredential

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

type createOfflineCredentialRequest struct {
	Platform string    `json:"platform"`
	ShopID   uuid.UUID `json:"shopId"`
}

type expectedRevisionRequest struct {
	ExpectedRevision int `json:"expectedRevision"`
}

type oauthStartRequest struct {
	Platform    string    `json:"platform"`
	ShopID      uuid.UUID `json:"shopId"`
	RedirectURI string    `json:"redirectUri"`
}

type oauthCompleteRequest struct {
	State string `json:"state"`
}

func (h *Handler) actor(c *gin.Context, permission string, shopID uuid.UUID) (Actor, *adminperm.Principal, bool) {
	if h == nil || h.Service == nil || h.Service.DB == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeInternalError, "生产凭据服务不可用")
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
		response.Fail(c, http.StatusForbidden, response.CodeForbidden, "无权操作该店铺")
		return Actor{}, nil, false
	}
	requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
	if requestID == "" {
		requestID = uuid.NewString()
	}
	return Actor{TenantID: tenantID, UserID: principal.UserID, RequestID: requestID}, principal, true
}

func bindStrictJSON(c *gin.Context, target any) error {
	if c.Request.Body == nil {
		return io.EOF
	}
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

func (h *Handler) List(c *gin.Context) {
	actor, principal, ok := h.actor(c, adminperm.PermInventorySyncRead, uuid.Nil)
	if !ok {
		return
	}
	out, err := h.Service.List(c.Request.Context(), actor.TenantID, principal.AllowedStoreIDs())
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, gin.H{"items": out})
}

func (h *Handler) CreateOffline(c *gin.Context) {
	var req createOfflineCredentialRequest
	if err := bindStrictJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "请求参数无效")
		return
	}
	actor, _, ok := h.actor(c, adminperm.PermConfigManage, req.ShopID)
	if !ok {
		return
	}
	out, err := h.Service.CreateOfflineCredential(c.Request.Context(), actor, req.Platform, req.ShopID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, response.CodeOK, "ok", out)
}

func (h *Handler) Rotate(c *gin.Context) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param("credentialId")))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "凭证编号无效")
		return
	}
	var req expectedRevisionRequest
	if err := bindStrictJSON(c, &req); err != nil || req.ExpectedRevision < 1 {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "请求参数无效")
		return
	}
	actor, _, ok := h.actor(c, adminperm.PermConfigManage, uuid.Nil)
	if !ok {
		return
	}
	out, err := h.Service.RotateOfflineCredential(c.Request.Context(), actor, id, req.ExpectedRevision)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) Revoke(c *gin.Context) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param("credentialId")))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "凭证编号无效")
		return
	}
	var req expectedRevisionRequest
	if err := bindStrictJSON(c, &req); err != nil || req.ExpectedRevision < 1 {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "请求参数无效")
		return
	}
	actor, _, ok := h.actor(c, adminperm.PermConfigManage, uuid.Nil)
	if !ok {
		return
	}
	out, err := h.Service.Revoke(c.Request.Context(), actor, id, req.ExpectedRevision)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) StartOfflineOAuth(c *gin.Context) {
	var req oauthStartRequest
	if err := bindStrictJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "请求参数无效")
		return
	}
	actor, _, ok := h.actor(c, adminperm.PermConfigManage, req.ShopID)
	if !ok {
		return
	}
	value, err := h.Service.BuildOfflineAuthorizationURL(c.Request.Context(), actor, req.Platform, req.ShopID, req.RedirectURI)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, gin.H{"authorizationUrl": value, "mode": "offline_fixture", "networkRequestExecuted": false})
}

func (h *Handler) CompleteOfflineOAuth(c *gin.Context) {
	var req oauthCompleteRequest
	if err := bindStrictJSON(c, &req); err != nil || strings.TrimSpace(req.State) == "" {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "请求参数无效")
		return
	}
	actor, _, ok := h.actor(c, adminperm.PermConfigManage, uuid.Nil)
	if !ok {
		return
	}
	out, err := h.Service.CompleteOfflineAuthorization(c.Request.Context(), actor, req.State)
	if err != nil {
		respondError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, response.CodeOK, "ok", out)
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrCredentialNotFound):
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "凭证不存在")
	case errors.Is(err, ErrCredentialConflict):
		response.Fail(c, http.StatusConflict, response.CodeBadRequest, "凭证版本冲突，请刷新后重试")
	case errors.Is(err, ErrOAuthStateInvalid):
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "授权状态无效、已过期或已使用")
	case errors.Is(err, ErrRedirectNotAllowed):
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "回调地址不在允许列表")
	case errors.Is(err, ErrCredentialUseDenied), errors.Is(err, ErrKeyUnavailable):
		response.Fail(c, http.StatusForbidden, response.CodeForbidden, "当前环境不允许使用凭证")
	default:
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalError, "凭证操作失败")
	}
}

func Register(group *gin.RouterGroup, h *Handler) {
	api := group.Group("/p10/credentials")
	api.GET("", h.List)
	api.POST("/offline", h.CreateOffline)
	api.POST("/:credentialId/rotate", h.Rotate)
	api.POST("/:credentialId/revoke", h.Revoke)
	api.POST("/oauth/offline/start", h.StartOfflineOAuth)
	api.POST("/oauth/offline/complete", h.CompleteOfflineOAuth)
}
