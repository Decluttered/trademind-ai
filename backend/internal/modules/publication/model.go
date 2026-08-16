package publication

import (
	"time"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/modules/listingstudio"
	"github.com/trademind-ai/trademind/backend/internal/pkg/model"
	"gorm.io/datatypes"
)

type SlotStatus string

const (
	SlotDraft      SlotStatus = "DRAFT"
	SlotScheduled  SlotStatus = "SCHEDULED"
	SlotHeld       SlotStatus = "HELD"
	SlotPublishing SlotStatus = "PUBLISHING"
	SlotPublished  SlotStatus = "PUBLISHED"
	SlotFailed     SlotStatus = "FAILED"
	SlotCancelled  SlotStatus = "CANCELLED"
)

type JobStatus string

const (
	JobScheduled  JobStatus = "SCHEDULED"
	JobPublishing JobStatus = "PUBLISHING"
	JobPublished  JobStatus = "PUBLISHED"
	JobFailed     JobStatus = "FAILED"
	JobDryRun     JobStatus = "DRY_RUN"
)

type CalendarSlot struct {
	model.Base
	WorkspaceID      int64      `gorm:"not null;index;uniqueIndex:ux_calendar_idem,priority:1" json:"workspaceId"`
	ListingDraftID   uuid.UUID  `gorm:"type:char(36);not null;index" json:"listingDraftId"`
	ContentVersionID uuid.UUID  `gorm:"type:char(36);not null;index" json:"contentVersionId"`
	ScheduledFor     time.Time  `gorm:"not null;index" json:"scheduledFor"`
	SlotType         string     `gorm:"size:32;not null;default:EBAY_PUBLISH;index" json:"slotType"`
	Status           SlotStatus `gorm:"size:24;not null;index" json:"status"`
	IdempotencyKey   string     `gorm:"size:160;not null;uniqueIndex:ux_calendar_idem,priority:2" json:"idempotencyKey"`
	PlannerScore     int        `gorm:"not null;default:0" json:"plannerScore"`
	PlannerReason    string     `gorm:"type:text" json:"plannerReason,omitempty"`
	PublicationJobID *uuid.UUID `gorm:"type:char(36);index" json:"publicationJobId,omitempty"`
}

func (CalendarSlot) TableName() string { return "calendar_slot" }

type PublicationJob struct {
	model.Base
	WorkspaceID      int64          `gorm:"not null;index;uniqueIndex:ux_publication_job_idem,priority:1" json:"workspaceId"`
	CalendarSlotID   uuid.UUID      `gorm:"type:char(36);not null;index" json:"calendarSlotId"`
	ListingDraftID   uuid.UUID      `gorm:"type:char(36);not null;index" json:"listingDraftId"`
	ContentVersionID uuid.UUID      `gorm:"type:char(36);not null;index" json:"contentVersionId"`
	ShopID           uuid.UUID      `gorm:"type:char(36);not null;index" json:"shopId"`
	Marketplace      string         `gorm:"size:32;not null;index" json:"marketplace"`
	Status           JobStatus      `gorm:"size:24;not null;index" json:"status"`
	IdempotencyKey   string         `gorm:"size:160;not null;uniqueIndex:ux_publication_job_idem,priority:2" json:"idempotencyKey"`
	WorkflowID       string         `gorm:"size:255;index" json:"workflowId,omitempty"`
	ScheduledFor     time.Time      `gorm:"not null;index" json:"scheduledFor"`
	Attempt          int            `gorm:"not null;default:0" json:"attempt"`
	PublishConfig    datatypes.JSON `gorm:"type:json;not null" json:"publishConfig"`
	RequestArtifact  datatypes.JSON `gorm:"type:json" json:"requestArtifact,omitempty"`
	ResponseArtifact datatypes.JSON `gorm:"type:json" json:"responseArtifact,omitempty"`
	ErrorClass       string         `gorm:"size:64;index" json:"errorClass,omitempty"`
	ErrorMessage     string         `gorm:"type:text" json:"errorMessage,omitempty"`
	PublishedAt      *time.Time     `json:"publishedAt,omitempty"`
	Actor            string         `gorm:"size:160;not null;default:system" json:"actor"`
	CorrelationID    string         `gorm:"size:160;index;not null;default:unknown" json:"correlationId"`
	PayloadHash      string         `gorm:"size:64;index;not null;default:0000000000000000000000000000000000000000000000000000000000000000" json:"payloadHash"`
}

func (PublicationJob) TableName() string { return "publication_job" }

type MarketplaceListing struct {
	model.Base
	WorkspaceID       int64     `gorm:"not null;index;uniqueIndex:ux_marketplace_external,priority:1" json:"workspaceId"`
	ListingDraftID    uuid.UUID `gorm:"type:char(36);not null;index" json:"listingDraftId"`
	PublicationJobID  uuid.UUID `gorm:"type:char(36);not null;index" json:"publicationJobId"`
	Marketplace       string    `gorm:"size:32;not null;uniqueIndex:ux_marketplace_external,priority:2" json:"marketplace"`
	ExternalListingID string    `gorm:"size:255;not null;uniqueIndex:ux_marketplace_external,priority:3" json:"externalListingId"`
	ExternalOfferID   string    `gorm:"size:255;index" json:"externalOfferId,omitempty"`
	ExternalURL       string    `gorm:"type:text" json:"externalUrl,omitempty"`
	SKU               string    `gorm:"size:80;not null;index" json:"sku"`
	Status            string    `gorm:"size:32;not null;index" json:"status"`
	PriceCents        int64     `gorm:"not null" json:"priceCents"`
	Currency          string    `gorm:"size:3;not null" json:"currency"`
}

func (MarketplaceListing) TableName() string { return "marketplace_listing" }

type ListingSnapshot struct {
	model.HardDeleteBase
	WorkspaceID          int64          `gorm:"not null;index" json:"workspaceId"`
	MarketplaceListingID *uuid.UUID     `gorm:"type:char(36);index" json:"marketplaceListingId,omitempty"`
	ListingDraftID       uuid.UUID      `gorm:"type:char(36);not null;index" json:"listingDraftId"`
	PublicationJobID     uuid.UUID      `gorm:"type:char(36);not null;index" json:"publicationJobId"`
	Kind                 string         `gorm:"size:40;not null;index" json:"kind"`
	PriceCents           *int64         `json:"priceCents,omitempty"`
	Quantity             *int           `json:"quantity,omitempty"`
	Status               string         `gorm:"size:32;index" json:"status,omitempty"`
	CapturedAt           *time.Time     `gorm:"index" json:"capturedAt,omitempty"`
	PayloadHash          string         `gorm:"size:64;index" json:"payloadHash,omitempty"`
	Payload              datatypes.JSON `gorm:"type:json;not null" json:"payload"`
}

func (ListingSnapshot) TableName() string { return "listing_snapshot" }

type PublicationTransitionEvent struct {
	model.HardDeleteBase
	WorkspaceID      int64                    `gorm:"not null;index" json:"workspaceId"`
	PublicationJobID uuid.UUID                `gorm:"type:char(36);not null;index" json:"publicationJobId"`
	ListingDraftID   uuid.UUID                `gorm:"type:char(36);not null;index" json:"listingDraftId"`
	FromState        listingstudio.DraftState `gorm:"size:24;not null" json:"fromState"`
	ToState          listingstudio.DraftState `gorm:"size:24;not null" json:"toState"`
	Actor            string                   `gorm:"size:160;not null" json:"actor"`
	CorrelationID    string                   `gorm:"size:160;index;not null" json:"correlationId"`
	PayloadHash      string                   `gorm:"size:64;index;not null" json:"payloadHash"`
}

func (PublicationTransitionEvent) TableName() string { return "publication_transition_event" }
