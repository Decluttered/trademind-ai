package publication

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/modules/listingstudio"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WorkflowStarter interface {
	StartPublishListing(context.Context, PublicationJob) error
}

type ShopGate interface {
	ValidateEbayShop(context.Context, int64, uuid.UUID) error
}

type Service struct {
	DB       *gorm.DB
	Listings *listingstudio.Service
	Starter  WorkflowStarter
	Shops    ShopGate
	Now      func() time.Time
}

type ApplyInput struct {
	ShopID        uuid.UUID       `json:"shopId"`
	Marketplace   string          `json:"marketplace"`
	Slots         []PreviewSlot   `json:"slots"`
	PublishConfig json.RawMessage `json:"publishConfig"`
	Actor         string          `json:"-"`
	CorrelationID string          `json:"-"`
}

type ApplyResult struct {
	Slots []CalendarSlot   `json:"slots"`
	Jobs  []PublicationJob `json:"jobs"`
}

type CalendarSlotView struct {
	CalendarSlot
	PriceCents        *int64    `json:"priceCents,omitempty"`
	Currency          string    `json:"currency,omitempty"`
	JobStatus         JobStatus `json:"jobStatus,omitempty"`
	ErrorMessage      string    `json:"errorMessage,omitempty"`
	ExternalListingID string    `json:"externalListingId,omitempty"`
	ExternalURL       string    `json:"externalUrl,omitempty"`
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) Preview(ctx context.Context, workspaceID int64, in PreviewInput) (PreviewResult, error) {
	if s == nil || s.DB == nil {
		return PreviewResult{}, fmt.Errorf("publication planner unavailable")
	}
	type row struct {
		ListingDraftID   uuid.UUID
		ContentVersionID uuid.UUID
		Title            string
		Category         string
		CreatedAt        time.Time
	}
	var rows []row
	err := s.DB.WithContext(ctx).Table("listing_draft AS d").
		Select("d.id AS listing_draft_id, d.current_content_version_id AS content_version_id, v.title, d.category, d.created_at").
		Joins("JOIN listing_content_version v ON v.id = d.current_content_version_id AND v.workspace_id = d.workspace_id").
		Where("d.workspace_id=? AND d.state=? AND d.deleted_at IS NULL", workspaceID, listingstudio.StateReady).
		Order("d.created_at ASC, d.id ASC").Scan(&rows).Error
	if err != nil {
		return PreviewResult{}, err
	}
	candidates := make([]Candidate, 0, len(rows))
	for _, row := range rows {
		candidates = append(candidates, Candidate(row))
	}
	return BuildPreview(in, candidates, s.now())
}

func (s *Service) Apply(ctx context.Context, workspaceID int64, commandKey string, in ApplyInput) (*ApplyResult, error) {
	if s == nil || s.DB == nil || s.Listings == nil {
		return nil, fmt.Errorf("publication planner unavailable")
	}
	commandKey = strings.TrimSpace(commandKey)
	if commandKey == "" {
		return nil, fmt.Errorf("idempotency key is required")
	}
	if in.ShopID == uuid.Nil || len(in.Slots) == 0 {
		return nil, fmt.Errorf("shopId and at least one slot are required")
	}
	if s.Shops != nil {
		if err := s.Shops.ValidateEbayShop(ctx, workspaceID, in.ShopID); err != nil {
			return nil, err
		}
	}
	marketplace := strings.TrimSpace(in.Marketplace)
	if marketplace == "" {
		marketplace = "EBAY_DE"
	}
	if marketplace != "EBAY_DE" {
		return nil, fmt.Errorf("Phase 2 supports EBAY_DE only")
	}
	actor := strings.TrimSpace(in.Actor)
	if actor == "" {
		actor = "system:calendar"
	}
	correlationID := strings.TrimSpace(in.CorrelationID)
	if correlationID == "" {
		correlationID = commandKey
	}
	config := datatypes.JSON(in.PublishConfig)
	if len(config) == 0 {
		config = datatypes.JSON([]byte("{}"))
	}
	result := &ApplyResult{}
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for index, proposed := range in.Slots {
			itemKey := fmt.Sprintf("%s:%d", commandKey, index)
			var existing CalendarSlot
			err := tx.Where("workspace_id=? AND idempotency_key=?", workspaceID, itemKey).First(&existing).Error
			if err == nil {
				var job PublicationJob
				if existing.PublicationJobID != nil {
					if findErr := tx.First(&job, "id=? AND workspace_id=?", *existing.PublicationJobID, workspaceID).Error; findErr != nil {
						return findErr
					}
				}
				result.Slots = append(result.Slots, existing)
				result.Jobs = append(result.Jobs, job)
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if proposed.ListingDraftID == uuid.Nil || proposed.ContentVersionID == uuid.Nil || proposed.ScheduledFor.Before(s.now().Add(-time.Minute)) {
				return fmt.Errorf("invalid proposed slot at index %d", index)
			}
			if err := s.ensureSlotFree(tx, workspaceID, proposed.ScheduledFor); err != nil {
				return err
			}
			if err := s.Listings.TransitionPublication(ctx, tx, workspaceID, proposed.ListingDraftID, listingstudio.StateReady, listingstudio.StateScheduled, proposed.ContentVersionID); err != nil {
				return err
			}
			slot := CalendarSlot{WorkspaceID: workspaceID, ListingDraftID: proposed.ListingDraftID, ContentVersionID: proposed.ContentVersionID, ScheduledFor: proposed.ScheduledFor.UTC(), SlotType: "EBAY_PUBLISH", Status: SlotScheduled, IdempotencyKey: itemKey, PlannerScore: proposed.Score, PlannerReason: proposed.Reason}
			if err := tx.Create(&slot).Error; err != nil {
				return err
			}
			payloadBytes, _ := json.Marshal(struct {
				WorkspaceID int64
				Slot        PreviewSlot
				Marketplace string
				Config      json.RawMessage
			}{workspaceID, proposed, marketplace, json.RawMessage(config)})
			payloadHash := fmt.Sprintf("%x", sha256.Sum256(payloadBytes))
			job := PublicationJob{WorkspaceID: workspaceID, CalendarSlotID: slot.ID, ListingDraftID: slot.ListingDraftID, ContentVersionID: slot.ContentVersionID, ShopID: in.ShopID, Marketplace: marketplace, Status: JobScheduled, IdempotencyKey: itemKey, WorkflowID: "publish-listing-" + slot.ID.String(), ScheduledFor: slot.ScheduledFor, PublishConfig: config, Actor: actor, CorrelationID: correlationID, PayloadHash: payloadHash}
			if err := tx.Create(&job).Error; err != nil {
				return err
			}
			slot.PublicationJobID = &job.ID
			if err := tx.Model(&slot).Update("publication_job_id", job.ID).Error; err != nil {
				return err
			}
			if err := recordTransition(tx, job, listingstudio.StateReady, listingstudio.StateScheduled, actor, correlationID); err != nil {
				return err
			}
			result.Slots = append(result.Slots, slot)
			result.Jobs = append(result.Jobs, job)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if s.Starter != nil {
		for _, job := range result.Jobs {
			if err := s.Starter.StartPublishListing(ctx, job); err != nil {
				return result, fmt.Errorf("slots persisted but workflow %s could not start: %w", job.WorkflowID, err)
			}
		}
	}
	return result, nil
}

func recordTransition(tx *gorm.DB, job PublicationJob, from, to listingstudio.DraftState, actor, correlationID string) error {
	event := PublicationTransitionEvent{WorkspaceID: job.WorkspaceID, PublicationJobID: job.ID, ListingDraftID: job.ListingDraftID, FromState: from, ToState: to, Actor: strings.TrimSpace(actor), CorrelationID: strings.TrimSpace(correlationID), PayloadHash: job.PayloadHash}
	return tx.Create(&event).Error
}

func (s *Service) ensureSlotFree(tx *gorm.DB, workspaceID int64, when time.Time) error {
	var count int64
	if err := tx.Model(&CalendarSlot{}).Where("workspace_id=? AND scheduled_for=? AND slot_type=? AND status IN ?", workspaceID, when.UTC(), "EBAY_PUBLISH", []SlotStatus{SlotScheduled, SlotHeld, SlotPublishing}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("an active publication slot already exists at %s", when.UTC().Format(time.RFC3339))
	}
	return nil
}

func (s *Service) ListSlots(ctx context.Context, workspaceID int64, from, to time.Time) ([]CalendarSlotView, error) {
	tx := s.DB.WithContext(ctx).Table("calendar_slot AS s").
		Select("s.*, d.price_cents, d.currency, j.status AS job_status, j.error_message, ml.external_listing_id, ml.external_url").
		Joins("JOIN listing_draft d ON d.id=s.listing_draft_id AND d.workspace_id=s.workspace_id").
		Joins("LEFT JOIN publication_job j ON j.id=s.publication_job_id AND j.workspace_id=s.workspace_id").
		Joins("LEFT JOIN marketplace_listing ml ON ml.publication_job_id=j.id AND ml.workspace_id=s.workspace_id").
		Where("s.workspace_id=? AND s.deleted_at IS NULL", workspaceID)
	if !from.IsZero() {
		tx = tx.Where("s.scheduled_for>=?", from.UTC())
	}
	if !to.IsZero() {
		tx = tx.Where("s.scheduled_for<?", to.UTC())
	}
	var rows []CalendarSlotView
	err := tx.Order("s.scheduled_for ASC, s.id ASC").Limit(500).Scan(&rows).Error
	return rows, err
}

func lockJob(tx *gorm.DB, workspaceID int64, id uuid.UUID) (*PublicationJob, error) {
	var job PublicationJob
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND workspace_id=?", id, workspaceID).First(&job).Error
	return &job, err
}
