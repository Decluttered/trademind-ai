package productioncontrolp10

import (
	"time"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/pkg/model"
	"gorm.io/datatypes"
)

const (
	GrayDraft           = "draft"
	GrayPendingApproval = "pending_approval"
	GrayApproved        = "approved"
	GrayActive          = "active"
	GrayPaused          = "paused"
	GrayStopped         = "stopped"
)

type RuntimeControl struct {
	model.HardDeleteBase
	TenantID           int64 `gorm:"not null;uniqueIndex" json:"tenantId"`
	ProviderKillActive bool  `gorm:"not null;default:true" json:"providerKillActive"`
	TenantKillActive   bool  `gorm:"not null;default:true" json:"tenantKillActive"`
	ShopKillActive     bool  `gorm:"not null;default:true" json:"shopKillActive"`
	ReadKillActive     bool  `gorm:"not null;default:true" json:"readKillActive"`
	WriteKillActive    bool  `gorm:"not null;default:true" json:"writeKillActive"`
	Revision           int   `gorm:"not null;default:1" json:"revision"`
}

func (RuntimeControl) TableName() string { return "production_runtime_controls" }

type ScopeAllowlist struct {
	model.HardDeleteBase
	TenantID int64     `gorm:"not null;uniqueIndex:ux_production_scope_allowlist_tenant_shop,priority:1;index" json:"tenantId"`
	ShopID   uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:ux_production_scope_allowlist_tenant_shop,priority:2;index" json:"shopId"`
	Enabled  bool      `gorm:"not null;default:false;index" json:"enabled"`
	Revision int       `gorm:"not null;default:1" json:"revision"`
}

func (ScopeAllowlist) TableName() string { return "production_scope_allowlists" }

type GrayPolicy struct {
	model.HardDeleteBase
	TenantID              int64      `gorm:"not null;uniqueIndex" json:"tenantId"`
	ShopID                uuid.UUID  `gorm:"type:char(36);not null;index" json:"shopId"`
	MaxSKU                int        `gorm:"not null;default:100" json:"maxSku"`
	Status                string     `gorm:"size:32;not null;index" json:"status"`
	OwnerApproved         bool       `gorm:"not null;default:false" json:"ownerApproved"`
	TechnicalLeadApproved bool       `gorm:"not null;default:false" json:"technicalLeadApproved"`
	ApprovedAt            *time.Time `json:"approvedAt,omitempty"`
	ActivatedAt           *time.Time `json:"activatedAt,omitempty"`
	PausedAt              *time.Time `json:"pausedAt,omitempty"`
	StoppedAt             *time.Time `json:"stoppedAt,omitempty"`
	Revision              int        `gorm:"not null;default:1" json:"revision"`
}

func (GrayPolicy) TableName() string { return "production_rollout_policies" }

type ControlAuditEvent struct {
	model.HardDeleteBase
	TenantID  int64          `gorm:"not null;index" json:"tenantId"`
	ActorID   uuid.UUID      `gorm:"type:char(36);not null;index" json:"actorId"`
	Action    string         `gorm:"size:64;not null;index" json:"action"`
	RequestID string         `gorm:"size:128;index" json:"requestId,omitempty"`
	Metadata  datatypes.JSON `gorm:"type:jsonb;not null" json:"metadata"`
}

func (ControlAuditEvent) TableName() string { return "production_control_audit_events" }

func Migrate(db interface{ AutoMigrate(...any) error }) error {
	return db.AutoMigrate(&RuntimeControl{}, &ScopeAllowlist{}, &GrayPolicy{}, &ControlAuditEvent{})
}
