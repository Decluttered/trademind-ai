package customerchat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/modules/operationlog"
	"github.com/trademind-ai/trademind/backend/internal/modules/shop"
	"github.com/trademind-ai/trademind/backend/internal/pkg/adminperm"
	"gorm.io/gorm"
)

const (
	defaultAutoReplyMaxRunes     = 600
	defaultAutoRepliesPerHour    = 20
	maxAutoReplyPolicyRunes      = 4000
	maxAutoRepliesPerHourAllowed = 100
)

type AutoReplyPolicyDTO struct {
	ShopID              uuid.UUID `json:"shopId"`
	ShopName            string    `json:"shopName"`
	Platform            string    `json:"platform"`
	GlobalEnabled       bool      `json:"globalEnabled"`
	WorkerAvailable     bool      `json:"workerAvailable"`
	Enabled             bool      `json:"enabled"`
	EffectiveEnabled    bool      `json:"effectiveEnabled"`
	Tone                string    `json:"tone"`
	ShopPolicy          string    `json:"shopPolicy,omitempty"`
	MaxReplyRunes       int       `json:"maxReplyRunes"`
	MaxRepliesPerHour   int       `json:"maxRepliesPerHour"`
	RequireOrderContext bool      `json:"requireOrderContext"`
	LowRiskOnly         bool      `json:"lowRiskOnly"`
	UpdatedAt           time.Time `json:"updatedAt,omitempty"`
}

type UpdateAutoReplyPolicyBody struct {
	Enabled             bool   `json:"enabled"`
	Tone                string `json:"tone"`
	ShopPolicy          string `json:"shopPolicy"`
	MaxReplyRunes       int    `json:"maxReplyRunes"`
	MaxRepliesPerHour   int    `json:"maxRepliesPerHour"`
	RequireOrderContext bool   `json:"requireOrderContext"`
	LowRiskOnly         bool   `json:"lowRiskOnly"`
}

type AutoReplyRunDTO struct {
	ID             uuid.UUID  `json:"id"`
	ConversationID uuid.UUID  `json:"conversationId"`
	MessageID      uuid.UUID  `json:"messageId"`
	SuggestionID   *uuid.UUID `json:"suggestionId,omitempty"`
	SentMessageID  *uuid.UUID `json:"sentMessageId,omitempty"`
	Status         string     `json:"status"`
	RiskLevel      string     `json:"riskLevel,omitempty"`
	ReasonCode     string     `json:"reasonCode,omitempty"`
	ErrorMessage   string     `json:"errorMessage,omitempty"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	FinishedAt     *time.Time `json:"finishedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

func defaultAutoReplyPolicy(shopRow *shop.Shop) CustomerAutoReplyPolicy {
	p := CustomerAutoReplyPolicy{
		Tone:                "professional",
		MaxReplyRunes:       defaultAutoReplyMaxRunes,
		MaxRepliesPerHour:   defaultAutoRepliesPerHour,
		RequireOrderContext: true,
		LowRiskOnly:         true,
	}
	if shopRow != nil {
		p.ShopID = shopRow.ID
		p.TenantID = shopRow.TenantID
	}
	return p
}

func normalizeAutoReplyPolicy(body UpdateAutoReplyPolicyBody) (UpdateAutoReplyPolicyBody, error) {
	body.Tone = strings.ToLower(strings.TrimSpace(body.Tone))
	if body.Tone == "" {
		body.Tone = "professional"
	}
	switch body.Tone {
	case "professional", "friendly", "concise", "empathetic":
	default:
		return body, fmt.Errorf("unsupported tone")
	}
	body.ShopPolicy = strings.TrimSpace(body.ShopPolicy)
	if len([]rune(body.ShopPolicy)) > maxAutoReplyPolicyRunes {
		return body, fmt.Errorf("shopPolicy is too long")
	}
	if body.MaxReplyRunes == 0 {
		body.MaxReplyRunes = defaultAutoReplyMaxRunes
	}
	if body.MaxReplyRunes < 50 || body.MaxReplyRunes > 2000 {
		return body, fmt.Errorf("maxReplyRunes must be between 50 and 2000")
	}
	if body.MaxRepliesPerHour == 0 {
		body.MaxRepliesPerHour = defaultAutoRepliesPerHour
	}
	if body.MaxRepliesPerHour < 1 || body.MaxRepliesPerHour > maxAutoRepliesPerHourAllowed {
		return body, fmt.Errorf("maxRepliesPerHour must be between 1 and %d", maxAutoRepliesPerHourAllowed)
	}
	// Production auto send is intentionally low-risk-only. Do not allow the API to weaken it.
	if body.Enabled && !body.LowRiskOnly {
		return body, fmt.Errorf("lowRiskOnly must remain enabled for automatic replies")
	}
	return body, nil
}

func (s *Service) loadShopForAutoReply(ctx context.Context, shopID uuid.UUID) (*shop.Shop, error) {
	if s == nil || s.DB == nil || shopID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	var row shop.Shop
	if err := s.DB.WithContext(ctx).First(&row, "id = ?", shopID).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) loadAutoReplyPolicy(ctx context.Context, shopRow *shop.Shop) (CustomerAutoReplyPolicy, error) {
	p := defaultAutoReplyPolicy(shopRow)
	if shopRow == nil {
		return p, gorm.ErrRecordNotFound
	}
	err := s.DB.WithContext(ctx).Where("shop_id = ?", shopRow.ID).First(&p).Error
	if err == nil {
		if p.MaxReplyRunes <= 0 {
			p.MaxReplyRunes = defaultAutoReplyMaxRunes
		}
		if p.MaxRepliesPerHour <= 0 {
			p.MaxRepliesPerHour = defaultAutoRepliesPerHour
		}
		if strings.TrimSpace(p.Tone) == "" {
			p.Tone = "professional"
		}
		return p, nil
	}
	if err == gorm.ErrRecordNotFound {
		return p, nil
	}
	return p, err
}

func (s *Service) autoReplyPolicyDTO(ctx context.Context, shopRow *shop.Shop, p CustomerAutoReplyPolicy) (AutoReplyPolicyDTO, error) {
	setting, err := s.LoadAutoReplySetting(ctx, shopRow.TenantID)
	if err != nil {
		return AutoReplyPolicyDTO{}, err
	}
	workerAvailable := s.autoReplyRuntimeAvailable(ctx) && setting.MessageSyncEnabled
	return AutoReplyPolicyDTO{
		ShopID:              shopRow.ID,
		ShopName:            shopRow.ShopName,
		Platform:            shopRow.Platform,
		GlobalEnabled:       setting.AutoReplyEnabled,
		WorkerAvailable:     workerAvailable,
		Enabled:             p.Enabled,
		EffectiveEnabled:    setting.AutoReplyEnabled && workerAvailable && p.Enabled,
		Tone:                p.Tone,
		ShopPolicy:          p.ShopPolicy,
		MaxReplyRunes:       p.MaxReplyRunes,
		MaxRepliesPerHour:   p.MaxRepliesPerHour,
		RequireOrderContext: p.RequireOrderContext,
		LowRiskOnly:         p.LowRiskOnly,
		UpdatedAt:           p.UpdatedAt,
	}, nil
}

func (s *Service) GetAutoReplyPolicy(c *gin.Context, shopID uuid.UUID) (*AutoReplyPolicyDTO, error) {
	shopRow, err := s.loadShopForAutoReply(c.Request.Context(), shopID)
	if err != nil {
		return nil, err
	}
	if err := adminperm.EnsureStoreVisible(c, s.DB, &shopRow.ID); err != nil {
		return nil, err
	}
	p, err := s.loadAutoReplyPolicy(c.Request.Context(), shopRow)
	if err != nil {
		return nil, err
	}
	out, err := s.autoReplyPolicyDTO(c.Request.Context(), shopRow, p)
	return &out, err
}

func (s *Service) ListAutoReplyRuns(c *gin.Context, shopID uuid.UUID) ([]AutoReplyRunDTO, error) {
	shopRow, err := s.loadShopForAutoReply(c.Request.Context(), shopID)
	if err != nil {
		return nil, err
	}
	if err := adminperm.EnsureStoreVisible(c, s.DB, &shopRow.ID); err != nil {
		return nil, err
	}
	var rows []CustomerAutoReplyRun
	if err := s.DB.WithContext(c.Request.Context()).
		Where("shop_id = ?", shopRow.ID).
		Order("created_at DESC").
		Limit(50).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]AutoReplyRunDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, AutoReplyRunDTO{
			ID:             row.ID,
			ConversationID: row.ConversationID,
			MessageID:      row.MessageID,
			SuggestionID:   row.SuggestionID,
			SentMessageID:  row.SentMessageID,
			Status:         row.Status,
			RiskLevel:      row.RiskLevel,
			ReasonCode:     row.ReasonCode,
			ErrorMessage:   truncateRunes(row.ErrorMessage, 500),
			StartedAt:      row.StartedAt,
			FinishedAt:     row.FinishedAt,
			CreatedAt:      row.CreatedAt,
		})
	}
	return out, nil
}

func (s *Service) UpdateAutoReplyPolicy(c *gin.Context, shopID uuid.UUID, body UpdateAutoReplyPolicyBody, adminID *uuid.UUID) (*AutoReplyPolicyDTO, error) {
	if !adminperm.CanManageSettings(c, s.DB) || !adminperm.CanWriteCustomer(c, s.DB) {
		return nil, gorm.ErrRecordNotFound
	}
	shopRow, err := s.loadShopForAutoReply(c.Request.Context(), shopID)
	if err != nil {
		return nil, err
	}
	if !adminperm.RequireStoreOperate(c, s.DB, shopID) {
		return nil, gorm.ErrRecordNotFound
	}
	body, err = normalizeAutoReplyPolicy(body)
	if err != nil {
		return nil, err
	}
	p, err := s.loadAutoReplyPolicy(c.Request.Context(), shopRow)
	if err != nil {
		return nil, err
	}
	p.TenantID = shopRow.TenantID
	p.ShopID = shopID
	p.Enabled = body.Enabled
	p.Tone = body.Tone
	p.ShopPolicy = body.ShopPolicy
	p.MaxReplyRunes = body.MaxReplyRunes
	p.MaxRepliesPerHour = body.MaxRepliesPerHour
	p.RequireOrderContext = body.RequireOrderContext
	p.LowRiskOnly = body.LowRiskOnly
	p.NextPollAt = nil
	if body.Enabled {
		now := time.Now().UTC()
		p.LastEnabledAt = &now
		p.LastEnabledBy = adminID
	}
	if err := s.DB.WithContext(c.Request.Context()).Save(&p).Error; err != nil {
		return nil, err
	}
	out, err := s.autoReplyPolicyDTO(c.Request.Context(), shopRow, p)
	if err != nil {
		return nil, err
	}
	if s.OpLog != nil {
		_ = s.OpLog.Write(c, operationlog.WriteOpts{
			AdminUserID: adminID,
			TenantID:    shopRow.TenantID,
			ShopID:      &shopRow.ID,
			Platform:    shopRow.Platform,
			Action:      "customer.auto_reply.policy.update",
			Resource:    "customer_auto_reply_policy",
			ResourceID:  p.ID.String(),
			Status:      "success",
			Message:     fmt.Sprintf("shopId=%s enabled=%t effective=%t lowRiskOnly=%t maxPerHour=%d", shopRow.ID, p.Enabled, out.EffectiveEnabled, p.LowRiskOnly, p.MaxRepliesPerHour),
		})
	}
	return &out, nil
}
