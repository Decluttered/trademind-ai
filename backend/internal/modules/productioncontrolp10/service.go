package productioncontrolp10

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/config"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrBlocked          = errors.New("p10 capability blocked")
	ErrScopeExceeded    = errors.New("p10 scope exceeded")
	ErrRevisionConflict = errors.New("p10 revision conflict")
	ErrInvalidControl   = errors.New("p10 invalid control")
)

type Actor struct {
	TenantID  int64
	UserID    uuid.UUID
	RequestID string
}

type RuntimeStatus struct {
	CurrentAllowedLevel           string          `json:"currentAllowedLevel"`
	Environment                   string          `json:"environment"`
	DevelopmentStatus             string          `json:"developmentStatus"`
	VerificationStatus            string          `json:"verificationStatus"`
	ManualAcceptanceStatus        string          `json:"manualAcceptanceStatus"`
	ExternalActivationStatus      string          `json:"externalActivationStatus"`
	ProviderProtocolMappingStatus string          `json:"providerProtocolMappingStatus"`
	RealProviderEnabled           bool            `json:"realProviderEnabled"`
	RealPlatformNetworkEnabled    bool            `json:"realPlatformNetworkEnabled"`
	RealCredentialsEnabled        bool            `json:"realCredentialsEnabled"`
	RealInventoryReadEnabled      bool            `json:"realInventoryReadEnabled"`
	RealInventoryWriteEnabled     bool            `json:"realInventoryWriteEnabled"`
	InventoryMutationEnabled      bool            `json:"inventoryMutationEnabled"`
	BackgroundWorkerEnabled       bool            `json:"backgroundWorkerEnabled"`
	AutomaticRetryEnabled         bool            `json:"automaticRetryEnabled"`
	ReadOnlyCapability            bool            `json:"readOnlyCapability"`
	OfflineOAuthEnabled           bool            `json:"offlineOAuthEnabled"`
	OfflineCredentialAvailable    bool            `json:"offlineCredentialAvailable"`
	Control                       RuntimeControl  `json:"control"`
	Allowlist                     *ScopeAllowlist `json:"allowlist,omitempty"`
	Gray                          *GrayPolicy     `json:"gray,omitempty"`
	LastRead                      *LastReadStatus `json:"lastRead,omitempty"`
	InitialLimits                 map[string]int  `json:"initialLimits"`
	ProductionReady               bool            `json:"productionReady"`
	ProductionAcceptancePassed    bool            `json:"productionAcceptancePassed"`
}

type LastReadStatus struct {
	RunID             uuid.UUID  `json:"runId"`
	RequestID         string     `json:"requestId,omitempty"`
	ProviderRequestID string     `json:"providerRequestId,omitempty"`
	Status            string     `json:"status"`
	ProviderMode      string     `json:"providerMode"`
	Revision          int        `json:"revision"`
	LastErrorCode     string     `json:"lastErrorCode,omitempty"`
	RateLimited       bool       `json:"rateLimited"`
	RetryAfterSeconds int64      `json:"retryAfterSeconds,omitempty"`
	StartedAt         *time.Time `json:"startedAt,omitempty"`
	FinishedAt        *time.Time `json:"finishedAt,omitempty"`
}

type Service struct {
	DB     *gorm.DB
	Config *config.Config
	Now    func() time.Time
}

func (s *Service) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func defaultControl(tenantID int64) RuntimeControl {
	return RuntimeControl{TenantID: tenantID, ProviderKillActive: true, TenantKillActive: true, ShopKillActive: true, ReadKillActive: true, WriteKillActive: true, Revision: 1}
}

func (s *Service) Status(ctx context.Context, tenantID int64, allowedShopIDs []uuid.UUID) (*RuntimeStatus, error) {
	if s == nil || s.DB == nil || s.Config == nil || tenantID <= 0 {
		return nil, ErrInvalidControl
	}
	control, err := s.getOrDefaultControl(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	var allow ScopeAllowlist
	allowPtr := (*ScopeAllowlist)(nil)
	if allowedShopIDs == nil || len(allowedShopIDs) > 0 {
		query := s.DB.WithContext(ctx).Where("tenant_id = ?", tenantID)
		if allowedShopIDs != nil {
			query = query.Where("shop_id IN ?", allowedShopIDs)
		}
		if err := query.First(&allow).Error; err == nil {
			allowPtr = &allow
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	var gray GrayPolicy
	grayPtr := (*GrayPolicy)(nil)
	if allowedShopIDs == nil || len(allowedShopIDs) > 0 {
		query := s.DB.WithContext(ctx).Where("tenant_id = ?", tenantID)
		if allowedShopIDs != nil {
			query = query.Where("shop_id IN ?", allowedShopIDs)
		}
		if err := query.First(&gray).Error; err == nil {
			grayPtr = &gray
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	lastRead, err := s.lastReadStatus(ctx, tenantID, allowedShopIDs)
	if err != nil {
		return nil, err
	}
	protocolStatus := "pending_official_api_contract"
	if strings.TrimSpace(s.Config.P10.DouyinAPIBaseURL) != "" {
		protocolStatus = "official_product_detail_with_local_binding_pagination"
	}
	return &RuntimeStatus{
		CurrentAllowedLevel: s.Config.P10.CurrentAllowedLevel, Environment: s.Config.AppEnv,
		DevelopmentStatus: "completed", VerificationStatus: "repository_checks_passed_manual_acceptance_pending", ManualAcceptanceStatus: "pending", ExternalActivationStatus: "blocked_by_external_infrastructure_and_credentials",
		ProviderProtocolMappingStatus: protocolStatus,
		RealProviderEnabled:           s.Config.P10.RealProviderEnabled, RealPlatformNetworkEnabled: s.Config.P10.RealPlatformNetworkEnabled, RealCredentialsEnabled: s.Config.P10.RealCredentialsEnabled, RealInventoryReadEnabled: s.Config.P10.RealInventoryReadEnabled,
		RealInventoryWriteEnabled: false, InventoryMutationEnabled: false, BackgroundWorkerEnabled: false, AutomaticRetryEnabled: false, ReadOnlyCapability: true,
		OfflineOAuthEnabled: s.Config.P10.OfflineOAuthEnabled, OfflineCredentialAvailable: s.Config.P10.OfflineOAuthEnabled && strings.TrimSpace(s.Config.P10.LocalCredentialKey) != "",
		Control: control, Allowlist: allowPtr, Gray: grayPtr, LastRead: lastRead, InitialLimits: map[string]int{"maxTenant": 1, "maxShop": 1, "maxSku": 100}, ProductionReady: false, ProductionAcceptancePassed: false,
	}, nil
}

func (s *Service) lastReadStatus(ctx context.Context, tenantID int64, allowedShopIDs []uuid.UUID) (*LastReadStatus, error) {
	if allowedShopIDs != nil && len(allowedShopIDs) == 0 {
		return nil, nil
	}
	type readRun struct {
		ID                uuid.UUID
		Status            string
		ProviderMode      string
		Revision          int
		RequestID         string
		SafeErrorMetadata datatypes.JSON
		StartedAt         *time.Time
		FinishedAt        *time.Time
	}
	var row readRun
	query := s.DB.WithContext(ctx).Table("p9_inventory_sync_runs").
		Select("id", "status", "provider_mode", "revision", "request_id", "safe_error_metadata", "started_at", "finished_at").
		Where("tenant_id = ? AND provider_mode = ?", tenantID, "real_readonly")
	if allowedShopIDs != nil {
		query = query.Where("shop_connection_id IN ?", allowedShopIDs)
	}
	err := query.Order("created_at DESC").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	metadata := struct {
		ErrorCode         string `json:"errorCode"`
		ProviderRequestID string `json:"providerRequestId"`
		RetryAfterSeconds int64  `json:"retryAfterSeconds"`
	}{}
	if len(row.SafeErrorMetadata) > 0 {
		_ = json.Unmarshal(row.SafeErrorMetadata, &metadata)
	}
	return &LastReadStatus{
		RunID: row.ID, RequestID: row.RequestID, ProviderRequestID: metadata.ProviderRequestID, Status: row.Status, ProviderMode: row.ProviderMode, Revision: row.Revision, LastErrorCode: metadata.ErrorCode,
		RateLimited: metadata.ErrorCode == "ratelimited" || metadata.ErrorCode == "rate_limited", RetryAfterSeconds: metadata.RetryAfterSeconds,
		StartedAt: row.StartedAt, FinishedAt: row.FinishedAt,
	}, nil
}

func (s *Service) EvaluateRead(ctx context.Context, tenantID int64, shopID uuid.UUID, skuCount int) error {
	if s == nil || s.Config == nil || s.DB == nil || tenantID <= 0 || shopID == uuid.Nil || skuCount < 0 || skuCount > 100 {
		return ErrScopeExceeded
	}
	control, err := s.getOrDefaultControl(ctx, tenantID)
	if err != nil {
		return err
	}
	// Kill switches always take precedence over feature flags.
	if control.ProviderKillActive || control.TenantKillActive || control.ShopKillActive || control.ReadKillActive || control.WriteKillActive == false {
		return ErrBlocked
	}
	if s.Config.P10.CurrentAllowedLevel == "L0" || !s.Config.P10.RealProviderEnabled || !s.Config.P10.RealPlatformNetworkEnabled || !s.Config.P10.RealCredentialsEnabled || !s.Config.P10.RealInventoryReadEnabled {
		return ErrBlocked
	}
	var allow ScopeAllowlist
	if err := s.DB.WithContext(ctx).Where("tenant_id = ? AND shop_id = ? AND enabled = ?", tenantID, shopID, true).First(&allow).Error; err != nil {
		return ErrBlocked
	}
	var gray GrayPolicy
	if err := s.DB.WithContext(ctx).Where("tenant_id = ? AND shop_id = ?", tenantID, shopID).First(&gray).Error; err != nil {
		return ErrBlocked
	}
	if !gray.OwnerApproved || !gray.TechnicalLeadApproved || (gray.Status != GrayApproved && gray.Status != GrayActive) || skuCount > gray.MaxSKU {
		return ErrBlocked
	}
	return nil
}

type SwitchUpdate struct {
	ProviderKillActive bool `json:"providerKillActive"`
	TenantKillActive   bool `json:"tenantKillActive"`
	ShopKillActive     bool `json:"shopKillActive"`
	ReadKillActive     bool `json:"readKillActive"`
	ExpectedRevision   int  `json:"expectedRevision"`
}

func (s *Service) UpdateSwitches(ctx context.Context, actor Actor, input SwitchUpdate) (*RuntimeControl, error) {
	if input.ExpectedRevision < 1 {
		return nil, ErrInvalidControl
	}
	var out RuntimeControl
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ?", actor.TenantID).First(&out).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) || input.ExpectedRevision != 1 {
				return mapConflict(err)
			}
			out = defaultControl(actor.TenantID)
			if err := tx.Create(&out).Error; err != nil {
				return err
			}
		}
		if out.Revision != input.ExpectedRevision {
			return ErrRevisionConflict
		}
		now := s.now()
		updates := map[string]any{"provider_kill_active": input.ProviderKillActive, "tenant_kill_active": input.TenantKillActive, "shop_kill_active": input.ShopKillActive, "read_kill_active": input.ReadKillActive, "write_kill_active": true, "revision": out.Revision + 1, "updated_at": now}
		res := tx.Model(&RuntimeControl{}).Where("tenant_id = ? AND revision = ?", actor.TenantID, out.Revision).Updates(updates)
		if res.Error != nil || res.RowsAffected != 1 {
			return ErrRevisionConflict
		}
		out.ProviderKillActive, out.TenantKillActive, out.ShopKillActive, out.ReadKillActive, out.WriteKillActive = input.ProviderKillActive, input.TenantKillActive, input.ShopKillActive, input.ReadKillActive, true
		out.Revision++
		return audit(tx, actor, "kill_switch_change", map[string]any{"provider": out.ProviderKillActive, "tenant": out.TenantKillActive, "shop": out.ShopKillActive, "read": out.ReadKillActive, "write": true})
	})
	return &out, err
}

func (s *Service) SetAllowlist(ctx context.Context, actor Actor, shopID uuid.UUID, enabled bool, expectedRevision int) (*ScopeAllowlist, error) {
	if shopID == uuid.Nil || expectedRevision < 0 {
		return nil, ErrInvalidControl
	}
	var out ScopeAllowlist
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if enabled {
			var tenantCount int64
			if err := tx.Model(&ScopeAllowlist{}).Distinct("tenant_id").Where("enabled = ? AND tenant_id <> ?", true, actor.TenantID).Count(&tenantCount).Error; err != nil {
				return err
			}
			if tenantCount > 0 {
				return ErrScopeExceeded
			}
		}
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ?", actor.TenantID).First(&out).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if expectedRevision != 0 {
				return ErrRevisionConflict
			}
			out = ScopeAllowlist{TenantID: actor.TenantID, ShopID: shopID, Enabled: enabled, Revision: 1}
			if err := tx.Create(&out).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			if out.Revision != expectedRevision {
				return ErrRevisionConflict
			}
			res := tx.Model(&ScopeAllowlist{}).Where("tenant_id = ? AND revision = ?", actor.TenantID, expectedRevision).Updates(map[string]any{"shop_id": shopID, "enabled": enabled, "revision": expectedRevision + 1, "updated_at": s.now()})
			if res.Error != nil || res.RowsAffected != 1 {
				return ErrRevisionConflict
			}
			out.ShopID, out.Enabled, out.Revision = shopID, enabled, expectedRevision+1
		}
		return audit(tx, actor, "allowlist_change", map[string]any{"shopId": shopID.String(), "enabled": enabled})
	})
	return &out, err
}

func (s *Service) SaveGrayDraft(ctx context.Context, actor Actor, shopID uuid.UUID, maxSKU, expectedRevision int) (*GrayPolicy, error) {
	if shopID == uuid.Nil || maxSKU < 1 || maxSKU > 100 || expectedRevision < 0 {
		return nil, ErrScopeExceeded
	}
	var out GrayPolicy
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ?", actor.TenantID).First(&out).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if expectedRevision != 0 {
				return ErrRevisionConflict
			}
			out = GrayPolicy{TenantID: actor.TenantID, ShopID: shopID, MaxSKU: maxSKU, Status: GrayDraft, Revision: 1}
			if err := tx.Create(&out).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			if out.Revision != expectedRevision || out.Status == GrayActive {
				return ErrRevisionConflict
			}
			res := tx.Model(&GrayPolicy{}).Where("tenant_id = ? AND revision = ?", actor.TenantID, expectedRevision).Updates(map[string]any{"shop_id": shopID, "max_sku": maxSKU, "status": GrayDraft, "owner_approved": false, "technical_lead_approved": false, "revision": expectedRevision + 1, "updated_at": s.now()})
			if res.Error != nil || res.RowsAffected != 1 {
				return ErrRevisionConflict
			}
			out.ShopID, out.MaxSKU, out.Status, out.OwnerApproved, out.TechnicalLeadApproved, out.Revision = shopID, maxSKU, GrayDraft, false, false, expectedRevision+1
		}
		return audit(tx, actor, "gray_scope_change", map[string]any{"shopId": shopID.String(), "maxSku": maxSKU, "approvalReset": true})
	})
	return &out, err
}

func (s *Service) PauseOrStopGray(ctx context.Context, actor Actor, action string, expectedRevision int) (*GrayPolicy, error) {
	if action != "pause" && action != "stop" {
		return nil, ErrInvalidControl
	}
	var out GrayPolicy
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ?", actor.TenantID).First(&out).Error; err != nil {
			return mapConflict(err)
		}
		if out.Revision != expectedRevision {
			return ErrRevisionConflict
		}
		now := s.now()
		status := GrayPaused
		updates := map[string]any{"status": status, "paused_at": now, "revision": expectedRevision + 1, "updated_at": now}
		if action == "stop" {
			status = GrayStopped
			updates = map[string]any{"status": status, "stopped_at": now, "revision": expectedRevision + 1, "updated_at": now}
		}
		res := tx.Model(&GrayPolicy{}).Where("tenant_id = ? AND revision = ?", actor.TenantID, expectedRevision).Updates(updates)
		if res.Error != nil || res.RowsAffected != 1 {
			return ErrRevisionConflict
		}
		out.Status, out.Revision = status, expectedRevision+1
		return audit(tx, actor, "gray_"+action, map[string]any{"shopId": out.ShopID.String()})
	})
	return &out, err
}

func (s *Service) getOrDefaultControl(ctx context.Context, tenantID int64) (RuntimeControl, error) {
	var row RuntimeControl
	if err := s.DB.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&row).Error; err == nil {
		row.WriteKillActive = true
		return row, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return RuntimeControl{}, err
	}
	return defaultControl(tenantID), nil
}

func audit(tx *gorm.DB, actor Actor, action string, metadata map[string]any) error {
	raw, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	return tx.Create(&ControlAuditEvent{TenantID: actor.TenantID, ActorID: actor.UserID, Action: action, RequestID: strings.TrimSpace(actor.RequestID), Metadata: datatypes.JSON(raw)}).Error
}

func mapConflict(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrRevisionConflict
	}
	return err
}

func SafeErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrBlocked):
		return "P10_READ_BLOCKED"
	case errors.Is(err, ErrScopeExceeded):
		return "P10_SCOPE_EXCEEDED"
	case errors.Is(err, ErrRevisionConflict):
		return "P10_REVISION_CONFLICT"
	default:
		return "P10_CONTROL_ERROR"
	}
}
