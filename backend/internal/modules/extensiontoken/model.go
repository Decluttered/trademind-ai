package extensiontoken

import (
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/pkg/model"
	"time"
)

type Grant struct {
	model.HardDeleteBase
	WorkspaceID int64      `gorm:"not null;index" json:"workspaceId"`
	AdminUserID uuid.UUID  `gorm:"type:char(36);not null;index" json:"adminUserId"`
	JTI         string     `gorm:"size:64;not null;uniqueIndex" json:"jti"`
	Scope       string     `gorm:"size:80;not null" json:"scope"`
	ExpiresAt   time.Time  `gorm:"not null;index" json:"expiresAt"`
	RevokedAt   *time.Time `gorm:"index" json:"revokedAt,omitempty"`
}

func (Grant) TableName() string { return "extension_token_grant" }
