package extensiontoken

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/config"
	"github.com/trademind-ai/trademind/backend/internal/pkg/adminperm"
	"github.com/trademind-ai/trademind/backend/internal/pkg/ctxkey"
	"github.com/trademind-ai/trademind/backend/internal/pkg/response"
	"gorm.io/gorm"
	"strings"
	"time"
)

type Handler struct {
	DB     *gorm.DB
	Config *config.Config
}

func (h *Handler) Mint(c *gin.Context) {
	w, err := adminperm.TenantIDFromGin(c)
	if err != nil {
		response.Fail(c, 401, response.CodeUnauthorized, "workspace unavailable")
		return
	}
	raw, ok := c.Get(ctxkey.AdminID)
	if !ok {
		response.Fail(c, 401, response.CodeUnauthorized, "admin unavailable")
		return
	}
	adminRaw, ok := raw.(string)
	if !ok {
		response.Fail(c, 401, response.CodeUnauthorized, "admin unavailable")
		return
	}
	adminID, err := uuid.Parse(strings.TrimSpace(adminRaw))
	if err != nil {
		response.Fail(c, 401, response.CodeUnauthorized, "admin unavailable")
		return
	}
	grant := Grant{WorkspaceID: w, AdminUserID: adminID, JTI: uuid.NewString(), Scope: "capture", ExpiresAt: time.Now().UTC().Add(15 * time.Minute)}
	grant.ID = uuid.New()
	token, err := Mint(h.Config, grant)
	if err != nil {
		response.Fail(c, 500, response.CodeInternalError, "extension token could not be signed")
		return
	}
	if err := h.DB.WithContext(c).Create(&grant).Error; err != nil {
		response.Fail(c, 500, response.CodeInternalError, "extension token could not be created")
		return
	}
	response.OK(c, gin.H{"grantId": grant.ID, "token": token, "audience": Audience, "scope": grant.Scope, "expiresAt": grant.ExpiresAt})
}
func (h *Handler) Revoke(c *gin.Context) {
	w, err := adminperm.TenantIDFromGin(c)
	if err != nil {
		response.Fail(c, 401, response.CodeUnauthorized, "workspace unavailable")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, 400, response.CodeBadRequest, "invalid grant id")
		return
	}
	now := time.Now().UTC()
	result := h.DB.WithContext(c).Model(&Grant{}).Where("id=? AND workspace_id=?", id, w).Update("revoked_at", now)
	if result.Error != nil {
		response.Fail(c, 500, response.CodeInternalError, "extension token could not be revoked")
		return
	}
	response.OK(c, gin.H{"revoked": result.RowsAffected > 0})
}
func RegisterAdmin(r gin.IRouter, h *Handler) {
	r.POST("/extension-tokens", h.Mint)
	r.DELETE("/extension-tokens/:id", h.Revoke)
}
