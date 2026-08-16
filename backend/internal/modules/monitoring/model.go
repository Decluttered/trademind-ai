package monitoring

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/pkg/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type DecisionOutcome string

const (
	OutcomeNoChange        DecisionOutcome = "NO_CHANGE"
	OutcomeProposed        DecisionOutcome = "PROPOSED"
	OutcomeAutoApplied     DecisionOutcome = "AUTO_APPLIED"
	OutcomeBlockedMargin   DecisionOutcome = "BLOCKED_MARGIN"
	OutcomeBlockedPolicy   DecisionOutcome = "BLOCKED_POLICY"
	OutcomeBlockedCooldown DecisionOutcome = "BLOCKED_COOLDOWN"
)

type MonitorRun struct {
	model.Base
	WorkspaceID          int64          `gorm:"not null;index;uniqueIndex:ux_monitor_run_idem,priority:1" json:"workspaceId"`
	MarketplaceListingID uuid.UUID      `gorm:"type:char(36);not null;index" json:"marketplaceListingId"`
	Trigger              string         `gorm:"size:24;not null;index" json:"trigger"`
	Status               string         `gorm:"size:24;not null;index" json:"status"`
	IdempotencyKey       string         `gorm:"size:160;not null;uniqueIndex:ux_monitor_run_idem,priority:2" json:"idempotencyKey"`
	CorrelationID        string         `gorm:"size:160;not null;index" json:"correlationId"`
	Input                datatypes.JSON `gorm:"type:json" json:"input,omitempty"`
	ErrorMessage         string         `gorm:"type:text" json:"errorMessage,omitempty"`
	CompletedAt          *time.Time     `json:"completedAt,omitempty"`
}

func (MonitorRun) TableName() string { return "monitor_run" }

type PriceRule struct {
	model.HardDeleteBase
	WorkspaceID             int64  `gorm:"not null;index;uniqueIndex:ux_price_rule_version,priority:1" json:"workspaceId"`
	Name                    string `gorm:"size:160;not null;uniqueIndex:ux_price_rule_version,priority:2" json:"name"`
	Version                 int    `gorm:"not null;uniqueIndex:ux_price_rule_version,priority:3" json:"version"`
	MinMarginBasisPoints    int64  `gorm:"not null" json:"minMarginBasisPoints"`
	TargetMarginBasisPoints int64  `gorm:"not null" json:"targetMarginBasisPoints"`
	MaxPriceCents           *int64 `json:"maxPriceCents,omitempty"`
	MaxDeltaCents           int64  `gorm:"not null" json:"maxDeltaCents"`
	MaxDeltaBasisPoints     int64  `gorm:"not null" json:"maxDeltaBasisPoints"`
	CooldownSeconds         int64  `gorm:"not null" json:"cooldownSeconds"`
	PlatformFeeBasisPoints  int64  `gorm:"not null" json:"platformFeeBasisPoints"`
	ShippingCents           int64  `gorm:"not null" json:"shippingCents"`
	ReserveCents            int64  `gorm:"not null" json:"reserveCents"`
	AutoApply               bool   `gorm:"not null" json:"autoApply"`
}

func (PriceRule) TableName() string { return "price_rule" }
func (*PriceRule) BeforeUpdate(*gorm.DB) error {
	return errors.New("price rules are immutable; create a new version")
}
func (*PriceRule) BeforeDelete(*gorm.DB) error { return errors.New("price rules are immutable") }

type PriceDecision struct {
	model.HardDeleteBase
	WorkspaceID          int64           `gorm:"not null;index;uniqueIndex:ux_price_decision_input,priority:1" json:"workspaceId"`
	MarketplaceListingID uuid.UUID       `gorm:"type:char(36);not null;index;uniqueIndex:ux_price_decision_input,priority:2" json:"marketplaceListingId"`
	MonitorRunID         uuid.UUID       `gorm:"type:char(36);not null;index" json:"monitorRunId"`
	PriceRuleID          uuid.UUID       `gorm:"type:char(36);not null;index" json:"priceRuleId"`
	RuleVersion          int             `gorm:"not null;uniqueIndex:ux_price_decision_input,priority:3" json:"ruleVersion"`
	SourceSnapshotID     uuid.UUID       `gorm:"type:char(36);not null;index;uniqueIndex:ux_price_decision_input,priority:4" json:"sourceSnapshotId"`
	ListingSnapshotID    uuid.UUID       `gorm:"type:char(36);not null;index" json:"listingSnapshotId"`
	OldPriceCents        int64           `gorm:"not null" json:"oldPriceCents"`
	TargetPriceCents     int64           `gorm:"not null" json:"targetPriceCents"`
	LandedCostCents      int64           `gorm:"not null" json:"landedCostCents"`
	ExpectedFeeCents     int64           `gorm:"not null" json:"expectedFeeCents"`
	ExpectedMarginCents  int64           `gorm:"not null" json:"expectedMarginCents"`
	ExpectedMarginBPS    int64           `gorm:"not null" json:"expectedMarginBasisPoints"`
	Outcome              DecisionOutcome `gorm:"size:32;not null;index" json:"outcome"`
	AutoEligible         bool            `gorm:"not null" json:"autoEligible"`
	Reason               string          `gorm:"type:text;not null" json:"reason"`
	InputHash            string          `gorm:"size:64;not null;index" json:"inputHash"`
	ApplyArtifact        datatypes.JSON  `gorm:"type:json" json:"applyArtifact,omitempty"`
	AppliedAt            *time.Time      `json:"appliedAt,omitempty"`
}

func (PriceDecision) TableName() string { return "price_decision" }
func (*PriceDecision) BeforeDelete(*gorm.DB) error {
	return errors.New("price decisions are immutable")
}

type ProfitLedgerEntry struct {
	model.HardDeleteBase
	WorkspaceID          int64     `gorm:"not null;index;uniqueIndex:ux_profit_evidence,priority:1" json:"workspaceId"`
	MarketplaceListingID uuid.UUID `gorm:"type:char(36);not null;index" json:"marketplaceListingId"`
	EntryType            string    `gorm:"size:40;not null;index;uniqueIndex:ux_profit_evidence,priority:2" json:"entryType"`
	Phase                string    `gorm:"size:16;not null;index" json:"phase"`
	AmountCents          int64     `gorm:"not null" json:"amountCents"`
	Currency             string    `gorm:"size:3;not null" json:"currency"`
	CalculationRule      string    `gorm:"size:160;not null" json:"calculationRule"`
	EvidenceType         string    `gorm:"size:40;not null;uniqueIndex:ux_profit_evidence,priority:3" json:"evidenceType"`
	EvidenceID           uuid.UUID `gorm:"type:char(36);not null;index;uniqueIndex:ux_profit_evidence,priority:4" json:"evidenceId"`
	OccurredAt           time.Time `gorm:"not null;index" json:"occurredAt"`
}

func (ProfitLedgerEntry) TableName() string { return "profit_ledger_entry" }
func (*ProfitLedgerEntry) BeforeUpdate(*gorm.DB) error {
	return errors.New("profit ledger entries are append-only")
}
func (*ProfitLedgerEntry) BeforeDelete(*gorm.DB) error {
	return errors.New("profit ledger entries are append-only")
}
