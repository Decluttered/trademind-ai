package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/pkg/pagination"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Collector interface {
	Collect(context.Context, string, string, map[string]any) (json.RawMessage, error)
}
type Service struct {
	DB        *gorm.DB
	Collector Collector
}

type NormalizedProduct struct {
	Source          string              `json:"source"`
	SourceURL       string              `json:"sourceUrl"`
	SourceProductID string              `json:"sourceProductId"`
	Title           string              `json:"title"`
	Currency        string              `json:"currency"`
	PriceCents      *int64              `json:"priceCents"`
	Availability    string              `json:"availability"`
	Brand           string              `json:"brand"`
	GTIN            string              `json:"gtin"`
	MainImages      []string            `json:"mainImages"`
	Attributes      map[string]any      `json:"attributes"`
	Variants        []map[string]string `json:"variants"`
	Raw             map[string]any      `json:"raw"`
}
type CaptureResult struct {
	Product    SourceProduct         `json:"product"`
	Snapshot   ProductSnapshot       `json:"snapshot"`
	Assessment OpportunityAssessment `json:"assessment"`
}

var amazonASIN = regexp.MustCompile(`^[A-Z0-9]{10}$`)

func jsonData(v any) datatypes.JSON { b, _ := json.Marshal(v); return datatypes.JSON(b) }

func (s *Service) Capture(ctx context.Context, workspaceID int64, p NormalizedProduct) (*CaptureResult, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("catalog unavailable")
	}
	p.Source = strings.ToLower(strings.TrimSpace(p.Source))
	p.SourceProductID = strings.ToUpper(strings.TrimSpace(p.SourceProductID))
	p.Currency = strings.ToUpper(strings.TrimSpace(p.Currency))
	if p.Source != "amazon.de" || !amazonASIN.MatchString(p.SourceProductID) || strings.TrimSpace(p.Title) == "" || p.Currency != "EUR" {
		return nil, fmt.Errorf("invalid Amazon.de normalized product")
	}
	var result CaptureResult
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		product := SourceProduct{WorkspaceID: workspaceID, Source: p.Source, ExternalID: p.SourceProductID, CanonicalURL: p.SourceURL}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "workspace_id"}, {Name: "source"}, {Name: "external_id"}}, DoUpdates: clause.Assignments(map[string]any{"canonical_url": p.SourceURL, "updated_at": time.Now().UTC()})}).Create(&product).Error; err != nil {
			return err
		}
		var persisted SourceProduct
		if err := tx.Where("workspace_id = ? AND source = ? AND external_id = ?", workspaceID, p.Source, p.SourceProductID).First(&persisted).Error; err != nil {
			return err
		}
		product = persisted
		snapshot := ProductSnapshot{WorkspaceID: workspaceID, SourceProductID: product.ID, CapturedAt: time.Now().UTC(), Title: p.Title, Currency: p.Currency, PriceCents: p.PriceCents, Availability: p.Availability, Brand: p.Brand, GTIN: p.GTIN, Images: jsonData(p.MainImages), Attributes: jsonData(p.Attributes), Variants: jsonData(p.Variants), Raw: jsonData(p.Raw)}
		if err := tx.Create(&snapshot).Error; err != nil {
			return err
		}
		if err := tx.Model(&SourceProduct{}).Where("id = ? AND workspace_id = ?", product.ID, workspaceID).Update("current_snapshot_id", snapshot.ID).Error; err != nil {
			return err
		}
		availability := 0.0
		if strings.Contains(strings.ToLower(p.Availability), "lager") {
			availability = 100
		}
		quality := float64(min(100, 25+len(p.MainImages)*10+len(p.Attributes)*5))
		scored := CalculateOpportunityScore(ScoreFactors{Availability: &availability, ListingQuality: &quality})
		assessment := OpportunityAssessment{WorkspaceID: workspaceID, SourceProductID: product.ID, SnapshotID: snapshot.ID, Score: scored.Score, Confidence: scored.Confidence, RuleVersion: scored.RuleVersion, Factors: jsonData(scored.Factors)}
		if err := tx.Create(&assessment).Error; err != nil {
			return err
		}
		product.CurrentSnapshotID = &snapshot.ID
		result = CaptureResult{Product: product, Snapshot: snapshot, Assessment: assessment}
		return nil
	})
	return &result, err
}

func (s *Service) RunDiscovery(ctx context.Context, workspaceID int64, rawURL string) (*CaptureResult, error) {
	if s.Collector == nil {
		return nil, fmt.Errorf("collector unavailable")
	}
	raw, err := s.Collector.Collect(ctx, "amazon.de", strings.TrimSpace(rawURL), nil)
	if err != nil {
		return nil, err
	}
	var product NormalizedProduct
	if err := json.Unmarshal(raw, &product); err != nil {
		return nil, fmt.Errorf("decode collector product: %w", err)
	}
	return s.Capture(ctx, workspaceID, product)
}

type ProductListItem struct {
	SourceProduct
	CurrentSnapshot ProductSnapshot        `json:"currentSnapshot"`
	Assessment      *OpportunityAssessment `json:"assessment,omitempty"`
}
type Page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
	HasMore    bool   `json:"hasMore"`
	Limit      int    `json:"limit"`
}

type ProductFilter struct {
	Query        string
	CollectionID *uuid.UUID
	MinScore     *int
}

func (s *Service) ListProducts(ctx context.Context, workspaceID int64, limit int, cursorRaw string, filter ProductFilter) (*Page[ProductListItem], error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	query := strings.TrimSpace(filter.Query)
	scope := pagination.Fingerprint(map[string]any{"q": query, "collectionId": filter.CollectionID, "minScore": filter.MinScore, "resource": "phase1-products"})
	cur, err := pagination.DecodeCursor(cursorRaw, workspaceID, "", scope)
	if err != nil {
		return nil, err
	}
	tx := s.DB.WithContext(ctx).Where("workspace_id = ?", workspaceID)
	if query != "" {
		tx = tx.Where("external_id LIKE ?", "%"+query+"%")
	}
	if filter.CollectionID != nil {
		tx = tx.Where("EXISTS (SELECT 1 FROM collection_product cp WHERE cp.workspace_id = source_product.workspace_id AND cp.source_product_id = source_product.id AND cp.collection_id = ?)", *filter.CollectionID)
	}
	if filter.MinScore != nil {
		tx = tx.Where("EXISTS (SELECT 1 FROM opportunity_assessment oa WHERE oa.workspace_id = source_product.workspace_id AND oa.source_product_id = source_product.id AND oa.score >= ? AND oa.created_at = (SELECT MAX(latest.created_at) FROM opportunity_assessment latest WHERE latest.workspace_id = oa.workspace_id AND latest.source_product_id = oa.source_product_id))", *filter.MinScore)
	}
	tx, err = pagination.ApplyDescKeyset(tx, "created_at", "id", cur)
	if err != nil {
		return nil, err
	}
	var products []SourceProduct
	if err = tx.Order("created_at DESC, id DESC").Limit(limit + 1).Find(&products).Error; err != nil {
		return nil, err
	}
	hasMore := len(products) > limit
	if hasMore {
		products = products[:limit]
	}
	items := make([]ProductListItem, 0, len(products))
	for _, p := range products {
		var snap ProductSnapshot
		if p.CurrentSnapshotID != nil {
			if err := s.DB.WithContext(ctx).Where("id = ? AND workspace_id = ?", *p.CurrentSnapshotID, workspaceID).First(&snap).Error; err != nil {
				return nil, err
			}
		}
		var a OpportunityAssessment
		var ap *OpportunityAssessment
		if s.DB.WithContext(ctx).Where("source_product_id = ? AND workspace_id = ?", p.ID, workspaceID).Order("created_at DESC").First(&a).Error == nil {
			ap = &a
		}
		items = append(items, ProductListItem{SourceProduct: p, CurrentSnapshot: snap, Assessment: ap})
	}
	next := ""
	if hasMore && len(products) > 0 {
		last := products[len(products)-1]
		next, err = pagination.BuildNextCursor(true, workspaceID, "", scope, "created_at", last.CreatedAt, last.ID.String())
		if err != nil {
			return nil, err
		}
	}
	return &Page[ProductListItem]{Items: items, NextCursor: next, HasMore: hasMore, Limit: limit}, nil
}

type CreateCollectionInput struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Kind        string         `json:"kind"`
	Rules       map[string]any `json:"rules"`
}

var collectionKinds = map[string]bool{"manual": true, "search": true, "seller": true, "rule": true, "import": true}

func (s *Service) CreateCollection(ctx context.Context, workspaceID int64, in CreateCollectionInput) (*Collection, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Kind = strings.ToLower(strings.TrimSpace(in.Kind))
	if in.Name == "" || !collectionKinds[in.Kind] {
		return nil, fmt.Errorf("collection name and valid kind are required")
	}
	row := Collection{WorkspaceID: workspaceID, Name: in.Name, Description: strings.TrimSpace(in.Description), Kind: in.Kind, Rules: jsonData(in.Rules)}
	return &row, s.DB.WithContext(ctx).Create(&row).Error
}

func (s *Service) AddProductToCollection(ctx context.Context, workspaceID int64, collectionID, productID uuid.UUID, reason string) (*CollectionProduct, error) {
	var collectionCount, productCount int64
	if err := s.DB.WithContext(ctx).Model(&Collection{}).Where("id=? AND workspace_id=?", collectionID, workspaceID).Count(&collectionCount).Error; err != nil {
		return nil, err
	}
	if err := s.DB.WithContext(ctx).Model(&SourceProduct{}).Where("id=? AND workspace_id=?", productID, workspaceID).Count(&productCount).Error; err != nil {
		return nil, err
	}
	if collectionCount != 1 || productCount != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	row := CollectionProduct{WorkspaceID: workspaceID, CollectionID: collectionID, SourceProductID: productID, Reason: strings.TrimSpace(reason)}
	created := s.DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	if created.Error != nil {
		return nil, created.Error
	}
	if created.RowsAffected == 0 {
		row.ID = uuid.Nil
		if err := s.DB.WithContext(ctx).Where("workspace_id=? AND collection_id=? AND source_product_id=?", workspaceID, collectionID, productID).First(&row).Error; err != nil {
			return nil, err
		}
	}
	return &row, nil
}

func (s *Service) ListCollections(ctx context.Context, workspaceID int64, limit int, cursorRaw string) (*Page[Collection], error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	scope := pagination.Fingerprint(map[string]any{"resource": "phase1-collections"})
	cur, err := pagination.DecodeCursor(cursorRaw, workspaceID, "", scope)
	if err != nil {
		return nil, err
	}
	tx := s.DB.WithContext(ctx).Where("workspace_id = ?", workspaceID)
	tx, err = pagination.ApplyDescKeyset(tx, "created_at", "id", cur)
	if err != nil {
		return nil, err
	}
	var rows []Collection
	if err = tx.Order("created_at DESC, id DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, err
	}
	more := len(rows) > limit
	if more {
		rows = rows[:limit]
	}
	next := ""
	if more && len(rows) > 0 {
		last := rows[len(rows)-1]
		next, err = pagination.BuildNextCursor(true, workspaceID, "", scope, "created_at", last.CreatedAt, last.ID.String())
	}
	return &Page[Collection]{Items: rows, NextCursor: next, HasMore: more, Limit: limit}, err
}

func (s *Service) GetProductSnapshot(ctx context.Context, workspaceID int64, sourceProductID uuid.UUID) (*SourceProduct, *ProductSnapshot, error) {
	var p SourceProduct
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", sourceProductID, workspaceID).First(&p).Error; err != nil {
		return nil, nil, err
	}
	if p.CurrentSnapshotID == nil {
		return nil, nil, gorm.ErrRecordNotFound
	}
	var snap ProductSnapshot
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", *p.CurrentSnapshotID, workspaceID).First(&snap).Error; err != nil {
		return nil, nil, err
	}
	return &p, &snap, nil
}
