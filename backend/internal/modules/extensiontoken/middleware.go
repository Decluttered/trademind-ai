package extensiontoken

import (
	"github.com/gin-gonic/gin"
	"github.com/trademind-ai/trademind/backend/internal/config"
	"github.com/trademind-ai/trademind/backend/internal/pkg/ctxkey"
	"github.com/trademind-ai/trademind/backend/internal/pkg/response"
	"gorm.io/gorm"
	"strings"
	"time"
)

func Middleware(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Fail(c, 401, response.CodeUnauthorized, "extension token required")
			c.Abort()
			return
		}
		claims, err := Parse(cfg, parts[1])
		if err != nil {
			response.Fail(c, 401, response.CodeUnauthorized, "invalid extension token")
			c.Abort()
			return
		}
		var count int64
		if err := db.WithContext(c).Model(&Grant{}).Where("jti=? AND workspace_id=? AND revoked_at IS NULL AND expires_at>?", claims.ID, claims.WorkspaceID, time.Now().UTC()).Count(&count).Error; err != nil || count != 1 {
			response.Fail(c, 401, response.CodeUnauthorized, "extension token revoked or expired")
			c.Abort()
			return
		}
		c.Set(ctxkey.AdminID, claims.Subject)
		c.Set(ctxkey.TenantID, claims.WorkspaceID)
		c.Next()
	}
}
