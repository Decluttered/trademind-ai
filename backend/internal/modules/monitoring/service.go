package monitoring

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/modules/catalog"
	"github.com/trademind-ai/trademind/backend/internal/modules/listingstudio"
	"github.com/trademind-ai/trademind/backend/internal/modules/publication"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OfferSnapshot struct {
	PriceCents int64
	Quantity   int
	Status     string
	Raw        map[string]any
}

type OfferGateway interface {
	ReadOffer(context.Context, int64, publication.PublicationJob, publication.MarketplaceListing) (OfferSnapshot, error)
	UpdateOffer(context.Context, int64, publication.PublicationJob, publication.MarketplaceListing, int64) (OfferSnapshot, map[string]any, bool, error)
}

type Service struct {
	DB             *gorm.DB
	Offers         OfferGateway
	AutomationMode string
	Now            func() time.Time
}

type CreateRuleInput struct {
	Name                    string `json:"name"`
	MinMarginBasisPoints    int64  `json:"minMarginBasisPoints"`
	TargetMarginBasisPoints int64  `json:"targetMarginBasisPoints"`
	MaxPriceCents           *int64 `json:"maxPriceCents"`
	MaxDeltaCents           int64  `json:"maxDeltaCents"`
	MaxDeltaBasisPoints     int64  `json:"maxDeltaBasisPoints"`
	CooldownSeconds         int64  `json:"cooldownSeconds"`
	PlatformFeeBasisPoints  int64  `json:"platformFeeBasisPoints"`
	ShippingCents           int64  `json:"shippingCents"`
	ReserveCents            int64  `json:"reserveCents"`
	AutoApply               bool   `json:"autoApply"`
}

type RunInput struct {
	MarketplaceListingID uuid.UUID `json:"marketplaceListingId"`
	PriceRuleID          uuid.UUID `json:"priceRuleId"`
	Trigger              string    `json:"trigger"`
}

type RunResult struct {
	Run      MonitorRun                  `json:"run"`
	Decision PriceDecision               `json:"decision"`
	Snapshot publication.ListingSnapshot `json:"listingSnapshot"`
}

type ProfitReport struct {
	Currency string              `json:"currency"`
	Totals   map[string]int64    `json:"totals"`
	Items    []ProfitLedgerEntry `json:"items"`
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func validateRule(in CreateRuleInput) error {
	if strings.TrimSpace(in.Name) == "" || in.MinMarginBasisPoints < 0 || in.TargetMarginBasisPoints < in.MinMarginBasisPoints || in.TargetMarginBasisPoints >= 10_000 || in.PlatformFeeBasisPoints < 0 || in.PlatformFeeBasisPoints >= 10_000 || in.MaxDeltaCents < 0 || in.MaxDeltaBasisPoints < 0 || in.CooldownSeconds < 0 || in.ShippingCents < 0 || in.ReserveCents < 0 || (in.MaxPriceCents != nil && *in.MaxPriceCents <= 0) {
		return fmt.Errorf("invalid price rule guardrails")
	}
	if in.TargetMarginBasisPoints+in.PlatformFeeBasisPoints >= 10_000 {
		return fmt.Errorf("target margin plus fee must be below 100 percent")
	}
	return nil
}

func (s *Service) CreateRule(ctx context.Context, workspaceID int64, in CreateRuleInput) (*PriceRule, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("monitoring service unavailable")
	}
	if err := validateRule(in); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	var out PriceRule
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var latest struct{ Version int }
		if err := tx.Model(&PriceRule{}).Select("COALESCE(MAX(version),0) AS version").Where("workspace_id=? AND name=?", workspaceID, name).Scan(&latest).Error; err != nil {
			return err
		}
		out = PriceRule{WorkspaceID: workspaceID, Name: name, Version: latest.Version + 1, MinMarginBasisPoints: in.MinMarginBasisPoints, TargetMarginBasisPoints: in.TargetMarginBasisPoints, MaxPriceCents: in.MaxPriceCents, MaxDeltaCents: in.MaxDeltaCents, MaxDeltaBasisPoints: in.MaxDeltaBasisPoints, CooldownSeconds: in.CooldownSeconds, PlatformFeeBasisPoints: in.PlatformFeeBasisPoints, ShippingCents: in.ShippingCents, ReserveCents: in.ReserveCents, AutoApply: in.AutoApply}
		return tx.Create(&out).Error
	})
	return &out, err
}

func (s *Service) ListRules(ctx context.Context, workspaceID int64) ([]PriceRule, error) {
	var rows []PriceRule
	err := s.DB.WithContext(ctx).Where("workspace_id=?", workspaceID).Order("name ASC, version DESC").Limit(200).Find(&rows).Error
	return rows, err
}

func (s *Service) ListMonitorable(ctx context.Context, workspaceID int64) ([]publication.MarketplaceListing, error) {
	var rows []publication.MarketplaceListing
	err := s.DB.WithContext(ctx).Where("workspace_id=? AND marketplace=?", workspaceID, "EBAY_DE").Order("updated_at DESC").Limit(200).Find(&rows).Error
	return rows, err
}

func (s *Service) Run(ctx context.Context, workspaceID int64, idempotencyKey, correlationID string, in RunInput) (*RunResult, error) {
	if s == nil || s.DB == nil || s.Offers == nil {
		return nil, fmt.Errorf("monitoring service unavailable")
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" || in.MarketplaceListingID == uuid.Nil || in.PriceRuleID == uuid.Nil {
		return nil, fmt.Errorf("idempotency key, marketplaceListingId and priceRuleId are required")
	}
	var existing MonitorRun
	if err := s.DB.WithContext(ctx).Where("workspace_id=? AND idempotency_key=?", workspaceID, idempotencyKey).First(&existing).Error; err == nil {
		return s.resultForRun(ctx, workspaceID, existing)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	var listing publication.MarketplaceListing
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", in.MarketplaceListingID, workspaceID).First(&listing).Error; err != nil {
		return nil, err
	}
	var job publication.PublicationJob
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", listing.PublicationJobID, workspaceID).First(&job).Error; err != nil {
		return nil, err
	}
	var draft listingstudio.ListingDraft
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", listing.ListingDraftID, workspaceID).First(&draft).Error; err != nil {
		return nil, err
	}
	var source catalog.SourceProduct
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", draft.SourceProductID, workspaceID).First(&source).Error; err != nil {
		return nil, err
	}
	if source.CurrentSnapshotID == nil {
		return nil, fmt.Errorf("source product has no current snapshot")
	}
	var sourceSnapshot catalog.ProductSnapshot
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", *source.CurrentSnapshotID, workspaceID).First(&sourceSnapshot).Error; err != nil {
		return nil, err
	}
	if sourceSnapshot.PriceCents == nil {
		return nil, fmt.Errorf("source snapshot has no price")
	}
	var rule PriceRule
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", in.PriceRuleID, workspaceID).First(&rule).Error; err != nil {
		return nil, err
	}
	trigger := strings.ToLower(strings.TrimSpace(in.Trigger))
	if trigger == "" {
		trigger = "manual"
	}
	inputJSON, _ := json.Marshal(in)
	run := MonitorRun{WorkspaceID: workspaceID, MarketplaceListingID: listing.ID, Trigger: trigger, Status: "RUNNING", IdempotencyKey: idempotencyKey, CorrelationID: strings.TrimSpace(correlationID), Input: datatypes.JSON(inputJSON)}
	if run.CorrelationID == "" {
		run.CorrelationID = idempotencyKey
	}
	if err := s.DB.WithContext(ctx).Create(&run).Error; err != nil {
		return nil, err
	}
	offer, err := s.Offers.ReadOffer(ctx, workspaceID, job, listing)
	if err != nil {
		_ = s.DB.WithContext(ctx).Model(&run).Updates(map[string]any{"status": "FAILED", "error_message": err.Error()}).Error
		return nil, err
	}
	now := s.now()
	payload, _ := json.Marshal(offer.Raw)
	payloadHash := fmt.Sprintf("%x", sha256.Sum256(payload))
	snapshot := publication.ListingSnapshot{WorkspaceID: workspaceID, MarketplaceListingID: &listing.ID, ListingDraftID: listing.ListingDraftID, PublicationJobID: listing.PublicationJobID, Kind: "MONITOR", PriceCents: &offer.PriceCents, Quantity: &offer.Quantity, Status: offer.Status, CapturedAt: &now, PayloadHash: payloadHash, Payload: datatypes.JSON(payload)}
	if len(snapshot.Payload) == 0 {
		snapshot.Payload = datatypes.JSON([]byte("{}"))
	}
	var lastApplied PriceDecision
	lastErr := s.DB.WithContext(ctx).Where("workspace_id=? AND marketplace_listing_id=? AND outcome=? AND applied_at IS NOT NULL", workspaceID, listing.ID, OutcomeAutoApplied).Order("applied_at DESC").First(&lastApplied).Error
	var lastAppliedAt *time.Time
	if lastErr == nil {
		lastAppliedAt = lastApplied.AppliedAt
	} else if !errors.Is(lastErr, gorm.ErrRecordNotFound) {
		return nil, lastErr
	}
	plan, err := Plan(PlanInput{SourcePriceCents: *sourceSnapshot.PriceCents, OldPriceCents: offer.PriceCents, Availability: sourceSnapshot.Availability, LastAppliedAt: lastAppliedAt, Now: now, Rule: rule})
	if err != nil {
		return nil, err
	}
	hashInput, _ := json.Marshal(struct {
		SourceSnapshotID   uuid.UUID
		ListingPayloadHash string
		RuleID             uuid.UUID
		RuleVersion        int
	}{sourceSnapshot.ID, payloadHash, rule.ID, rule.Version})
	decision := PriceDecision{WorkspaceID: workspaceID, MarketplaceListingID: listing.ID, MonitorRunID: run.ID, PriceRuleID: rule.ID, RuleVersion: rule.Version, SourceSnapshotID: sourceSnapshot.ID, OldPriceCents: offer.PriceCents, TargetPriceCents: plan.TargetPriceCents, LandedCostCents: plan.LandedCostCents, ExpectedFeeCents: plan.ExpectedFeeCents, ExpectedMarginCents: plan.ExpectedMarginCents, ExpectedMarginBPS: plan.ExpectedMarginBPS, Outcome: plan.Outcome, AutoEligible: plan.AutoEligible, Reason: plan.Reason, InputHash: fmt.Sprintf("%x", sha256.Sum256(hashInput))}
	completed := now
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&snapshot).Error; err != nil {
			return err
		}
		decision.ListingSnapshotID = snapshot.ID
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "workspace_id"}, {Name: "marketplace_listing_id"}, {Name: "rule_version"}, {Name: "source_snapshot_id"}}, DoNothing: true}).Create(&decision).Error; err != nil {
			return err
		}
		if decision.ID == uuid.Nil {
			if err := tx.Where("workspace_id=? AND marketplace_listing_id=? AND rule_version=? AND source_snapshot_id=?", workspaceID, listing.ID, rule.Version, sourceSnapshot.ID).First(&decision).Error; err != nil {
				return err
			}
		}
		if err := s.recordExpectedLedger(tx, listing, decision, rule, now); err != nil {
			return err
		}
		return tx.Model(&MonitorRun{}).Where("id=? AND workspace_id=?", run.ID, workspaceID).Updates(map[string]any{"status": "SUCCEEDED", "completed_at": completed}).Error
	})
	if err != nil {
		return nil, err
	}
	run.Status, run.CompletedAt = "SUCCEEDED", &completed
	if decision.AutoEligible && !strings.EqualFold(strings.TrimSpace(s.AutomationMode), "DRY_RUN") {
		if _, applyErr := s.Apply(ctx, workspaceID, decision.ID); applyErr != nil {
			return nil, applyErr
		}
		_ = s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", decision.ID, workspaceID).First(&decision).Error
	}
	return &RunResult{Run: run, Decision: decision, Snapshot: snapshot}, nil
}

func (s *Service) resultForRun(ctx context.Context, workspaceID int64, run MonitorRun) (*RunResult, error) {
	var decision PriceDecision
	if err := s.DB.WithContext(ctx).Where("workspace_id=? AND monitor_run_id=?", workspaceID, run.ID).First(&decision).Error; err != nil {
		return nil, err
	}
	var snapshot publication.ListingSnapshot
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", decision.ListingSnapshotID, workspaceID).First(&snapshot).Error; err != nil {
		return nil, err
	}
	return &RunResult{Run: run, Decision: decision, Snapshot: snapshot}, nil
}

func (s *Service) recordExpectedLedger(tx *gorm.DB, listing publication.MarketplaceListing, decision PriceDecision, rule PriceRule, occurred time.Time) error {
	entries := []struct {
		kind   string
		amount int64
	}{{"expected_revenue", decision.TargetPriceCents}, {"expected_cost", decision.LandedCostCents}, {"expected_fees", decision.ExpectedFeeCents}, {"expected_margin", decision.ExpectedMarginCents}}
	for _, item := range entries {
		entry := ProfitLedgerEntry{WorkspaceID: listing.WorkspaceID, MarketplaceListingID: listing.ID, EntryType: item.kind, Phase: "expected", AmountCents: item.amount, Currency: listing.Currency, CalculationRule: fmt.Sprintf("price-rule:%s:v%d", rule.Name, rule.Version), EvidenceType: "price_decision", EvidenceID: decision.ID, OccurredAt: occurred}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&entry).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Apply(ctx context.Context, workspaceID int64, id uuid.UUID) (*PriceDecision, error) {
	if s == nil || s.DB == nil || s.Offers == nil {
		return nil, fmt.Errorf("monitoring service unavailable")
	}
	var decision PriceDecision
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", id, workspaceID).First(&decision).Error; err != nil {
		return nil, err
	}
	if decision.Outcome == OutcomeAutoApplied {
		return &decision, nil
	}
	if decision.Outcome != OutcomeProposed || decision.TargetPriceCents <= 0 {
		return nil, fmt.Errorf("decision outcome %s cannot be applied", decision.Outcome)
	}
	var listing publication.MarketplaceListing
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", decision.MarketplaceListingID, workspaceID).First(&listing).Error; err != nil {
		return nil, err
	}
	var job publication.PublicationJob
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", listing.PublicationJobID, workspaceID).First(&job).Error; err != nil {
		return nil, err
	}
	verified, artifact, dryRun, err := s.Offers.UpdateOffer(ctx, workspaceID, job, listing, decision.TargetPriceCents)
	if err != nil {
		return nil, err
	}
	artifactJSON, _ := json.Marshal(artifact)
	if dryRun {
		if err := s.DB.WithContext(ctx).Model(&PriceDecision{}).Where("id=? AND workspace_id=? AND outcome=?", decision.ID, workspaceID, OutcomeProposed).Update("apply_artifact", datatypes.JSON(artifactJSON)).Error; err != nil {
			return nil, err
		}
		decision.ApplyArtifact = datatypes.JSON(artifactJSON)
		return &decision, nil
	}
	now := s.now()
	verifyPayload, _ := json.Marshal(verified.Raw)
	verifyHash := fmt.Sprintf("%x", sha256.Sum256(verifyPayload))
	verifySnapshot := publication.ListingSnapshot{WorkspaceID: workspaceID, MarketplaceListingID: &listing.ID, ListingDraftID: listing.ListingDraftID, PublicationJobID: listing.PublicationJobID, Kind: "REPRICE_VERIFY", PriceCents: &verified.PriceCents, Quantity: &verified.Quantity, Status: verified.Status, CapturedAt: &now, PayloadHash: verifyHash, Payload: datatypes.JSON(verifyPayload)}
	if len(verifySnapshot.Payload) == 0 {
		verifySnapshot.Payload = datatypes.JSON([]byte("{}"))
	}
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked PriceDecision
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND workspace_id=?", decision.ID, workspaceID).First(&locked).Error; err != nil {
			return err
		}
		if locked.Outcome == OutcomeAutoApplied {
			decision = locked
			return nil
		}
		if verified.PriceCents != decision.TargetPriceCents {
			return fmt.Errorf("eBay verification price does not match target")
		}
		if err := tx.Create(&verifySnapshot).Error; err != nil {
			return err
		}
		if err := tx.Model(&publication.MarketplaceListing{}).Where("id=? AND workspace_id=?", listing.ID, workspaceID).Update("price_cents", verified.PriceCents).Error; err != nil {
			return err
		}
		if err := tx.Model(&PriceDecision{}).Where("id=? AND workspace_id=?", decision.ID, workspaceID).Updates(map[string]any{"outcome": OutcomeAutoApplied, "applied_at": now, "apply_artifact": datatypes.JSON(artifactJSON)}).Error; err != nil {
			return err
		}
		decision.Outcome, decision.AppliedAt, decision.ApplyArtifact = OutcomeAutoApplied, &now, datatypes.JSON(artifactJSON)
		return nil
	})
	return &decision, err
}

func (s *Service) ListDecisions(ctx context.Context, workspaceID int64, outcome string) ([]PriceDecision, error) {
	tx := s.DB.WithContext(ctx).Where("workspace_id=?", workspaceID)
	if strings.TrimSpace(outcome) != "" {
		tx = tx.Where("outcome=?", strings.TrimSpace(outcome))
	}
	var rows []PriceDecision
	err := tx.Order("created_at DESC, id DESC").Limit(200).Find(&rows).Error
	return rows, err
}

func (s *Service) Profit(ctx context.Context, workspaceID int64, from, to time.Time) (*ProfitReport, error) {
	tx := s.DB.WithContext(ctx).Where("workspace_id=?", workspaceID)
	if !from.IsZero() {
		tx = tx.Where("occurred_at>=?", from.UTC())
	}
	if !to.IsZero() {
		tx = tx.Where("occurred_at<?", to.UTC())
	}
	var rows []ProfitLedgerEntry
	if err := tx.Order("occurred_at DESC, id DESC").Limit(500).Find(&rows).Error; err != nil {
		return nil, err
	}
	report := &ProfitReport{Currency: "EUR", Totals: map[string]int64{}, Items: rows}
	for _, row := range rows {
		report.Totals[row.EntryType] += row.AmountCents
	}
	return report, nil
}
