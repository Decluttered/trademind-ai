package credentialp10

import (
	"time"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/pkg/model"
	"gorm.io/datatypes"
)

const (
	StatusPending = "pending"
	StatusActive  = "active"
	StatusRetired = "retired"
	StatusRevoked = "revoked"
	StatusExpired = "expired"
)

type PlatformCredential struct {
	model.HardDeleteBase
	TenantID      int64      `gorm:"not null;uniqueIndex:ux_p10_credentials_tenant_platform_shop,priority:1;index" json:"tenantId"`
	Platform      string     `gorm:"size:64;not null;uniqueIndex:ux_p10_credentials_tenant_platform_shop,priority:2;index" json:"platform"`
	ShopID        uuid.UUID  `gorm:"type:char(36);not null;uniqueIndex:ux_p10_credentials_tenant_platform_shop,priority:3;index" json:"shopId"`
	Status        string     `gorm:"size:32;not null;index" json:"status"`
	ActiveVersion int        `gorm:"not null;default:0" json:"activeVersion"`
	ExpiresAt     *time.Time `gorm:"index" json:"expiresAt,omitempty"`
	RotatedAt     *time.Time `json:"rotatedAt,omitempty"`
	RevokedAt     *time.Time `gorm:"index" json:"revokedAt,omitempty"`
	Revision      int        `gorm:"not null;default:1" json:"version"`
}

func (PlatformCredential) TableName() string { return "p10_platform_credentials" }

type OAuthCredential struct {
	model.HardDeleteBase
	TenantID             int64          `gorm:"not null;uniqueIndex:ux_p10_oauth_credentials_tenant_credential,priority:1;index" json:"tenantId"`
	PlatformCredentialID uuid.UUID      `gorm:"type:char(36);not null;uniqueIndex:ux_p10_oauth_credentials_tenant_credential,priority:2;index" json:"credentialId"`
	GrantType            string         `gorm:"size:64;not null" json:"grantType"`
	Scopes               datatypes.JSON `gorm:"type:jsonb" json:"scopes,omitempty"`
}

func (OAuthCredential) TableName() string { return "p10_oauth_credentials" }

type CredentialBinding struct {
	model.HardDeleteBase
	TenantID             int64     `gorm:"not null;uniqueIndex:ux_p10_credential_bindings_tenant_shop,priority:1;index" json:"tenantId"`
	PlatformCredentialID uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:ux_p10_credential_bindings_tenant_shop,priority:2;index" json:"credentialId"`
	ShopID               uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:ux_p10_credential_bindings_tenant_shop,priority:3;index" json:"shopId"`
	Platform             string    `gorm:"size:64;not null" json:"platform"`
}

func (CredentialBinding) TableName() string { return "p10_credential_bindings" }

type CredentialVersion struct {
	model.HardDeleteBase
	TenantID             int64      `gorm:"not null;uniqueIndex:ux_p10_credential_versions_tenant_credential_version,priority:1;index" json:"tenantId"`
	PlatformCredentialID uuid.UUID  `gorm:"type:char(36);not null;uniqueIndex:ux_p10_credential_versions_tenant_credential_version,priority:2;index" json:"credentialId"`
	Version              int        `gorm:"not null;uniqueIndex:ux_p10_credential_versions_tenant_credential_version,priority:3" json:"version"`
	Status               string     `gorm:"size:32;not null;index" json:"status"`
	Ciphertext           string     `gorm:"type:text;not null" json:"-"`
	Nonce                string     `gorm:"size:128;not null" json:"-"`
	Algorithm            string     `gorm:"size:64;not null" json:"algorithm"`
	KeyReference         string     `gorm:"size:255;not null" json:"keyReference"`
	EnvelopeVersion      int        `gorm:"not null" json:"envelopeVersion"`
	ExpiresAt            *time.Time `gorm:"index" json:"expiresAt,omitempty"`
	ActivatedAt          *time.Time `json:"activatedAt,omitempty"`
	RetiredAt            *time.Time `json:"retiredAt,omitempty"`
	RevokedAt            *time.Time `json:"revokedAt,omitempty"`
}

func (CredentialVersion) TableName() string { return "p10_credential_versions" }

type CredentialLifecycleEvent struct {
	model.HardDeleteBase
	TenantID     int64          `gorm:"not null;index" json:"tenantId"`
	CredentialID uuid.UUID      `gorm:"type:char(36);not null;index" json:"credentialId"`
	Version      int            `gorm:"not null" json:"version"`
	Action       string         `gorm:"size:64;not null;index" json:"action"`
	ActorID      uuid.UUID      `gorm:"type:char(36);not null;index" json:"actorId"`
	RequestID    string         `gorm:"size:128;index" json:"requestId,omitempty"`
	Metadata     datatypes.JSON `gorm:"type:jsonb;not null" json:"metadata"`
}

func (CredentialLifecycleEvent) TableName() string { return "p10_credential_lifecycle_events" }

type OAuthState struct {
	model.HardDeleteBase
	StateHash   string     `gorm:"size:64;uniqueIndex;not null" json:"-"`
	TenantID    int64      `gorm:"not null;index" json:"tenantId"`
	UserID      uuid.UUID  `gorm:"type:char(36);not null;index" json:"userId"`
	Platform    string     `gorm:"size:64;not null" json:"platform"`
	ShopID      uuid.UUID  `gorm:"type:char(36);not null;index" json:"shopId"`
	RedirectURI string     `gorm:"type:text;not null" json:"redirectUri"`
	ExpiresAt   time.Time  `gorm:"not null;index" json:"expiresAt"`
	ConsumedAt  *time.Time `gorm:"index" json:"consumedAt,omitempty"`
}

func (OAuthState) TableName() string { return "p10_oauth_states" }

func Migrate(db interface{ AutoMigrate(...any) error }) error {
	return db.AutoMigrate(&PlatformCredential{}, &OAuthCredential{}, &CredentialBinding{}, &CredentialVersion{}, &CredentialLifecycleEvent{}, &OAuthState{})
}
