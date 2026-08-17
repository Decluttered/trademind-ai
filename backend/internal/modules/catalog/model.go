package catalog

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/pkg/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SourceProduct struct {
	model.HardDeleteBase
	WorkspaceID       int64      `gorm:"not null;uniqueIndex:ux_source_product_workspace_source_external,priority:1;index" json:"workspaceId"`
	Source            string     `gorm:"size:40;not null;uniqueIndex:ux_source_product_workspace_source_external,priority:2" json:"source"`
	ExternalID        string     `gorm:"size:128;not null;uniqueIndex:ux_source_product_workspace_source_external,priority:3" json:"externalId"`
	CanonicalURL      string     `gorm:"type:text;not null" json:"canonicalUrl"`
	CurrentSnapshotID *uuid.UUID `gorm:"type:char(36);index" json:"currentSnapshotId,omitempty"`
}

func (SourceProduct) TableName() string { return "source_product" }

type ProductSnapshot struct {
	model.HardDeleteBase
	WorkspaceID     int64          `gorm:"not null;index" json:"workspaceId"`
	SourceProductID uuid.UUID      `gorm:"type:char(36);not null;index" json:"sourceProductId"`
	CapturedAt      time.Time      `gorm:"not null;index" json:"capturedAt"`
	Title           string         `gorm:"type:text;not null" json:"title"`
	Currency        string         `gorm:"size:3;not null" json:"currency"`
	PriceCents      *int64         `json:"priceCents,omitempty"`
	Availability    string         `gorm:"type:text" json:"availability,omitempty"`
	Brand           string         `gorm:"type:text" json:"brand,omitempty"`
	GTIN            string         `gorm:"size:32" json:"gtin,omitempty"`
	Images          datatypes.JSON `gorm:"type:json" json:"images"`
	Attributes      datatypes.JSON `gorm:"type:json" json:"attributes"`
	Variants        datatypes.JSON `gorm:"type:json" json:"variants"`
	Raw             datatypes.JSON `gorm:"type:json;not null" json:"raw"`
}

func (ProductSnapshot) TableName() string { return "product_snapshot" }
func (p *ProductSnapshot) BeforeUpdate(*gorm.DB) error {
	return errors.New("product snapshots are immutable")
}
func (p *ProductSnapshot) BeforeDelete(*gorm.DB) error {
	return errors.New("product snapshots are immutable")
}

type Collection struct {
	model.Base
	WorkspaceID int64          `gorm:"not null;index" json:"workspaceId"`
	Name        string         `gorm:"size:160;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	Kind        string         `gorm:"size:24;not null;index" json:"kind"`
	Rules       datatypes.JSON `gorm:"type:json" json:"rules,omitempty"`
}

func (Collection) TableName() string { return "collection" }

type CollectionProduct struct {
	model.HardDeleteBase
	WorkspaceID     int64     `gorm:"not null;uniqueIndex:ux_collection_product,priority:1" json:"workspaceId"`
	CollectionID    uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:ux_collection_product,priority:2" json:"collectionId"`
	SourceProductID uuid.UUID `gorm:"type:char(36);not null;uniqueIndex:ux_collection_product,priority:3" json:"sourceProductId"`
	Reason          string    `gorm:"type:text" json:"reason,omitempty"`
}

func (CollectionProduct) TableName() string { return "collection_product" }

type OpportunityAssessment struct {
	model.HardDeleteBase
	WorkspaceID     int64          `gorm:"not null;index" json:"workspaceId"`
	SourceProductID uuid.UUID      `gorm:"type:char(36);not null;index" json:"sourceProductId"`
	SnapshotID      uuid.UUID      `gorm:"type:char(36);not null;index" json:"snapshotId"`
	Score           int            `gorm:"not null" json:"score"`
	Confidence      int            `gorm:"not null" json:"confidence"`
	RuleVersion     string         `gorm:"size:40;not null" json:"ruleVersion"`
	Factors         datatypes.JSON `gorm:"type:json;not null" json:"factors"`
}

func (OpportunityAssessment) TableName() string { return "opportunity_assessment" }
