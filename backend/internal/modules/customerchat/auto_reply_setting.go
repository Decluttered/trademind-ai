package customerchat

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/modules/operationlog"
	"github.com/trademind-ai/trademind/backend/internal/pkg/adminperm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultAutoReplyPollIntervalSeconds = 60
	minimumAutoReplyPollIntervalSeconds = 15
	maximumAutoReplyPollIntervalSeconds = 3600
)

type AutoReplySettingDTO struct {
	MessageSyncEnabled  bool      `json:"messageSyncEnabled"`
	AutoReplyEnabled    bool      `json:"autoReplyEnabled"`
	PollIntervalSeconds int       `json:"pollIntervalSeconds"`
	WorkerAvailable     bool      `json:"workerAvailable"`
	EffectiveEnabled    bool      `json:"effectiveEnabled"`
	UpdatedAt           time.Time `json:"updatedAt,omitempty"`
}

type UpdateAutoReplySettingBody struct {
	MessageSyncEnabled  bool `json:"messageSyncEnabled"`
	AutoReplyEnabled    bool `json:"autoReplyEnabled"`
	PollIntervalSeconds int  `json:"pollIntervalSeconds"`
}

func defaultAutoReplySetting(tenantID int64) CustomerAutoReplySetting {
	return CustomerAutoReplySetting{TenantID: tenantID, PollIntervalSeconds: defaultAutoReplyPollIntervalSeconds}
}

func normalizeAutoReplySetting(body UpdateAutoReplySettingBody) (UpdateAutoReplySettingBody, error) {
	if body.PollIntervalSeconds == 0 {
		body.PollIntervalSeconds = defaultAutoReplyPollIntervalSeconds
	}
	if body.PollIntervalSeconds < minimumAutoReplyPollIntervalSeconds || body.PollIntervalSeconds > maximumAutoReplyPollIntervalSeconds {
		return body, fmt.Errorf("pollIntervalSeconds must be between %d and %d", minimumAutoReplyPollIntervalSeconds, maximumAutoReplyPollIntervalSeconds)
	}
	if body.AutoReplyEnabled && !body.MessageSyncEnabled {
		return body, fmt.Errorf("messageSyncEnabled must be enabled before automatic replies")
	}
	return body, nil
}

// LoadAutoReplySetting returns fail-closed tenant runtime settings for request and worker paths.
func (s *Service) LoadAutoReplySetting(ctx context.Context, tenantID int64) (CustomerAutoReplySetting, error) {
	setting := defaultAutoReplySetting(tenantID)
	if s == nil || s.DB == nil || tenantID < 0 {
		return setting, fmt.Errorf("customer auto reply settings unavailable")
	}
	err := s.DB.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&setting).Error
	if err == gorm.ErrRecordNotFound {
		return setting, nil
	}
	if err != nil {
		return setting, err
	}
	if setting.PollIntervalSeconds < minimumAutoReplyPollIntervalSeconds || setting.PollIntervalSeconds > maximumAutoReplyPollIntervalSeconds {
		setting.PollIntervalSeconds = defaultAutoReplyPollIntervalSeconds
	}
	return setting, nil
}

func (s *Service) autoReplySettingDTO(ctx context.Context, setting CustomerAutoReplySetting) AutoReplySettingDTO {
	workerAvailable := s.autoReplyRuntimeAvailable(ctx)
	return AutoReplySettingDTO{
		MessageSyncEnabled: setting.MessageSyncEnabled, AutoReplyEnabled: setting.AutoReplyEnabled,
		PollIntervalSeconds: setting.PollIntervalSeconds, WorkerAvailable: workerAvailable,
		EffectiveEnabled: workerAvailable && setting.MessageSyncEnabled && setting.AutoReplyEnabled,
		UpdatedAt:        setting.UpdatedAt,
	}
}

func (s *Service) GetAutoReplySetting(c *gin.Context) (*AutoReplySettingDTO, error) {
	tenantID, err := adminperm.TenantIDFromGin(c)
	if err != nil {
		return nil, err
	}
	setting, err := s.LoadAutoReplySetting(c.Request.Context(), tenantID)
	if err != nil {
		return nil, err
	}
	out := s.autoReplySettingDTO(c.Request.Context(), setting)
	return &out, nil
}

func (s *Service) UpdateAutoReplySetting(c *gin.Context, body UpdateAutoReplySettingBody, adminID *uuid.UUID) (*AutoReplySettingDTO, error) {
	if !adminperm.CanManageSettings(c, s.DB) || !adminperm.CanWriteCustomer(c, s.DB) {
		return nil, gorm.ErrRecordNotFound
	}
	tenantID, err := adminperm.TenantIDFromGin(c)
	if err != nil {
		return nil, err
	}
	body, err = normalizeAutoReplySetting(body)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	settingRow := defaultAutoReplySetting(tenantID)
	settingRow.MessageSyncEnabled = body.MessageSyncEnabled
	settingRow.AutoReplyEnabled = body.AutoReplyEnabled
	settingRow.PollIntervalSeconds = body.PollIntervalSeconds
	settingRow.UpdatedBy = adminID
	err = s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "tenant_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"message_sync_enabled": body.MessageSyncEnabled, "auto_reply_enabled": body.AutoReplyEnabled,
				"poll_interval_seconds": body.PollIntervalSeconds, "updated_by": adminID, "updated_at": now,
			}),
		}).Create(&settingRow).Error; err != nil {
			return err
		}
		return tx.Model(&CustomerAutoReplyPolicy{}).Where("tenant_id = ?", tenantID).Update("next_poll_at", nil).Error
	})
	if err != nil {
		return nil, err
	}
	setting, err := s.LoadAutoReplySetting(c.Request.Context(), tenantID)
	if err != nil {
		return nil, err
	}
	out := s.autoReplySettingDTO(c.Request.Context(), setting)
	if s.OpLog != nil {
		_ = s.OpLog.Write(c, operationlog.WriteOpts{AdminUserID: adminID, TenantID: tenantID,
			Action: "customer.auto_reply.setting.update", Resource: "customer_auto_reply_setting", ResourceID: setting.ID.String(), Status: "success",
			Message: fmt.Sprintf("messageSyncEnabled=%t autoReplyEnabled=%t pollIntervalSeconds=%d effective=%t", setting.MessageSyncEnabled, setting.AutoReplyEnabled, setting.PollIntervalSeconds, out.EffectiveEnabled)})
	}
	return &out, nil
}
