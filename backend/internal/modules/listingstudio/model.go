package listingstudio

import (
	"errors"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/pkg/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

type DraftState string

const (
	StateDiscovered  DraftState = "DISCOVERED"
	StateAssessed    DraftState = "ASSESSED"
	StateDrafting    DraftState = "DRAFTING"
	StateNeedsReview DraftState = "NEEDS_REVIEW"
	StateReady       DraftState = "READY"
	StateScheduled   DraftState = "SCHEDULED"
	StatePublishing  DraftState = "PUBLISHING"
	StatePublished   DraftState = "PUBLISHED"
	StatePaused      DraftState = "PAUSED"
	StateEnded       DraftState = "ENDED"
	StatePublishFail DraftState = "PUBLISH_FAILED"
	StateBlocked     DraftState = "BLOCKED"
)

var transitions = map[DraftState]map[DraftState]bool{
	StateDiscovered:  {StateAssessed: true},
	StateAssessed:    {StateDrafting: true},
	StateDrafting:    {StateNeedsReview: true, StateBlocked: true},
	StateNeedsReview: {StateReady: true, StateBlocked: true, StateDrafting: true},
	StateReady:       {StateScheduled: true},
	StateScheduled:   {StatePublishing: true, StateReady: true},
	StatePublishing:  {StatePublished: true, StatePublishFail: true},
	StatePublished:   {StatePaused: true, StateEnded: true},
	StatePaused:      {StatePublished: true, StateEnded: true},
	StatePublishFail: {StateReady: true},
	StateBlocked:     {StateDrafting: true},
}

func CanTransition(from, to DraftState) bool { return from == to || transitions[from][to] }

type ListingDraft struct {
	model.Base
	WorkspaceID             int64          `gorm:"not null;index" json:"workspaceId"`
	SourceProductID         uuid.UUID      `gorm:"type:char(36);not null;index" json:"sourceProductId"`
	State                   DraftState     `gorm:"size:24;not null;index" json:"state"`
	Category                string         `gorm:"type:text" json:"category,omitempty"`
	PriceCents              *int64         `json:"priceCents,omitempty"`
	Currency                string         `gorm:"size:3;not null" json:"currency"`
	RequiredSpecifics       datatypes.JSON `gorm:"type:json" json:"requiredSpecifics"`
	Specifics               datatypes.JSON `gorm:"type:json" json:"specifics"`
	CurrentContentVersionID *uuid.UUID     `gorm:"type:char(36);index" json:"currentContentVersionId,omitempty"`
	ImageSetID              *uuid.UUID     `gorm:"type:char(36);index" json:"imageSetId,omitempty"`
	ValidationErrors        datatypes.JSON `gorm:"type:json" json:"validationErrors"`
}

func (ListingDraft) TableName() string { return "listing_draft" }

type ListingContentVersion struct {
	model.HardDeleteBase
	WorkspaceID    int64          `gorm:"not null;index;uniqueIndex:ux_listing_content_version,priority:1" json:"workspaceId"`
	ListingDraftID uuid.UUID      `gorm:"type:char(36);not null;index;uniqueIndex:ux_listing_content_version,priority:2" json:"listingDraftId"`
	Version        int            `gorm:"not null;uniqueIndex:ux_listing_content_version,priority:3" json:"version"`
	Title          string         `gorm:"type:text" json:"title"`
	Description    string         `gorm:"type:text" json:"description"`
	Specifics      datatypes.JSON `gorm:"type:json" json:"specifics"`
	Generator      string         `gorm:"size:80" json:"generator"`
	PromptCode     string         `gorm:"size:120" json:"promptCode"`
}

func (ListingContentVersion) TableName() string { return "listing_content_version" }
func (v *ListingContentVersion) BeforeUpdate(*gorm.DB) error {
	return errors.New("listing content versions are immutable")
}
func (v *ListingContentVersion) BeforeDelete(*gorm.DB) error {
	return errors.New("listing content versions are immutable")
}

type GPSRProfile struct {
	model.Base
	WorkspaceID              int64          `gorm:"not null;index" json:"workspaceId"`
	Name                     string         `gorm:"size:160;not null" json:"name"`
	ManufacturerName         string         `gorm:"type:text" json:"manufacturerName"`
	ManufacturerAddress      string         `gorm:"type:text" json:"manufacturerAddress"`
	ResponsiblePersonName    string         `gorm:"type:text" json:"responsiblePersonName"`
	ResponsiblePersonAddress string         `gorm:"type:text" json:"responsiblePersonAddress"`
	SafetyInformation        string         `gorm:"type:text" json:"safetyInformation"`
	DocumentReferences       datatypes.JSON `gorm:"type:json" json:"documentReferences"`
}

func (GPSRProfile) TableName() string { return "gpsr_profile" }

type ListingGPSR struct {
	model.HardDeleteBase
	WorkspaceID    int64      `gorm:"not null;uniqueIndex:ux_listing_gpsr,priority:1" json:"workspaceId"`
	ListingDraftID uuid.UUID  `gorm:"type:char(36);not null;uniqueIndex:ux_listing_gpsr,priority:2" json:"listingDraftId"`
	GPSRProfileID  *uuid.UUID `gorm:"type:char(36);index" json:"gpsrProfileId,omitempty"`
	Override       bool       `gorm:"not null" json:"override"`
	OverrideReason string     `gorm:"type:text" json:"overrideReason,omitempty"`
	OverriddenBy   *uuid.UUID `gorm:"type:char(36)" json:"overriddenBy,omitempty"`
	OverriddenAt   *time.Time `json:"overriddenAt,omitempty"`
}

func (ListingGPSR) TableName() string { return "listing_gpsr" }

type ImageAsset struct {
	model.HardDeleteBase
	WorkspaceID int64          `gorm:"not null;index" json:"workspaceId"`
	OriginURL   string         `gorm:"type:text;not null" json:"originUrl"`
	SHA256      string         `gorm:"size:64;not null;index" json:"sha256"`
	StorageKey  string         `gorm:"type:text;not null" json:"storageKey"`
	ContentType string         `gorm:"size:80;not null" json:"contentType"`
	SizeBytes   int64          `gorm:"not null" json:"sizeBytes"`
	Width       int            `json:"width"`
	Height      int            `json:"height"`
	Flags       datatypes.JSON `gorm:"type:json" json:"flags"`
}

func (ImageAsset) TableName() string { return "image_asset" }

type ImageDerivative struct {
	model.HardDeleteBase
	WorkspaceID  int64     `gorm:"not null;index" json:"workspaceId"`
	ImageAssetID uuid.UUID `gorm:"type:char(36);not null;index" json:"imageAssetId"`
	Kind         string    `gorm:"size:40;not null" json:"kind"`
	StorageKey   string    `gorm:"type:text;not null" json:"storageKey"`
	SHA256       string    `gorm:"size:64;not null" json:"sha256"`
	ContentType  string    `gorm:"size:80;not null" json:"contentType"`
}

func (ImageDerivative) TableName() string { return "image_derivative" }

type ImageSet struct {
	model.Base
	WorkspaceID int64          `gorm:"not null;index" json:"workspaceId"`
	Name        string         `gorm:"size:160;not null" json:"name"`
	AssetIDs    datatypes.JSON `gorm:"type:json;not null" json:"assetIds"`
}

func (ImageSet) TableName() string { return "image_set" }
