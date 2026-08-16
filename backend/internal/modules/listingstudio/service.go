package listingstudio

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/pkg/pagination"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"regexp"
	"sort"
	"strings"
	"time"
)

type SourceFacts struct {
	Title      string
	Brand      string
	Attributes map[string]any
	Images     []string
}
type SourceReader interface {
	ReadSource(context.Context, int64, uuid.UUID) (SourceFacts, error)
}
type GeneratedContent struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Specifics   map[string]string `json:"specifics"`
}
type Generator interface {
	Generate(context.Context, int64, SourceFacts) (GeneratedContent, string, string, error)
}
type SettingsReader interface {
	PlainByGroup(context.Context, int64, string) (map[string]string, error)
}
type Service struct {
	DB        *gorm.DB
	Sources   SourceReader
	Generator Generator
	Settings  SettingsReader
}
type CreateDraftInput struct {
	SourceProductID   uuid.UUID         `json:"sourceProductId"`
	Category          string            `json:"category"`
	PriceCents        *int64            `json:"priceCents"`
	RequiredSpecifics []string          `json:"requiredSpecifics"`
	Specifics         map[string]string `json:"specifics"`
	ImageAssetIDs     []uuid.UUID       `json:"imageAssetIds"`
	GPSRProfileID     *uuid.UUID        `json:"gpsrProfileId"`
}

type CreateGPSRProfileInput struct {
	Name                     string   `json:"name"`
	ManufacturerName         string   `json:"manufacturerName"`
	ManufacturerAddress      string   `json:"manufacturerAddress"`
	ResponsiblePersonName    string   `json:"responsiblePersonName"`
	ResponsiblePersonAddress string   `json:"responsiblePersonAddress"`
	SafetyInformation        string   `json:"safetyInformation"`
	DocumentReferences       []string `json:"documentReferences"`
}

func (s *Service) CreateGPSRProfile(ctx context.Context, w int64, in CreateGPSRProfileInput) (*GPSRProfile, error) {
	values := []string{in.Name, in.ManufacturerName, in.ManufacturerAddress, in.ResponsiblePersonName, in.ResponsiblePersonAddress, in.SafetyInformation}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("all GPSR identity, address and safety fields are required")
		}
	}
	if len(in.DocumentReferences) == 0 {
		return nil, fmt.Errorf("at least one GPSR document reference is required")
	}
	row := GPSRProfile{WorkspaceID: w, Name: strings.TrimSpace(in.Name), ManufacturerName: strings.TrimSpace(in.ManufacturerName), ManufacturerAddress: strings.TrimSpace(in.ManufacturerAddress), ResponsiblePersonName: strings.TrimSpace(in.ResponsiblePersonName), ResponsiblePersonAddress: strings.TrimSpace(in.ResponsiblePersonAddress), SafetyInformation: strings.TrimSpace(in.SafetyInformation), DocumentReferences: j(in.DocumentReferences)}
	return &row, s.DB.WithContext(ctx).Create(&row).Error
}

func (s *Service) ListGPSRProfiles(ctx context.Context, w int64) ([]GPSRProfile, error) {
	var rows []GPSRProfile
	err := s.DB.WithContext(ctx).Where("workspace_id=?", w).Order("name ASC, id ASC").Limit(200).Find(&rows).Error
	return rows, err
}

func j(v any) datatypes.JSON { b, _ := json.Marshal(v); return datatypes.JSON(b) }

func (s *Service) Create(ctx context.Context, w int64, in CreateDraftInput) (*ListingDraft, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("listing studio unavailable")
	}
	if in.SourceProductID == uuid.Nil {
		return nil, fmt.Errorf("sourceProductId is required")
	}
	if s.Sources != nil {
		if _, err := s.Sources.ReadSource(ctx, w, in.SourceProductID); err != nil {
			return nil, err
		}
	}
	if in.PriceCents != nil && *in.PriceCents < 0 {
		return nil, fmt.Errorf("priceCents must be non-negative")
	}
	row := ListingDraft{WorkspaceID: w, SourceProductID: in.SourceProductID, State: StateDrafting, Category: strings.TrimSpace(in.Category), PriceCents: in.PriceCents, Currency: "EUR", RequiredSpecifics: j(in.RequiredSpecifics), Specifics: j(in.Specifics), ValidationErrors: j([]string{})}
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(in.ImageAssetIDs) > 0 {
			var count int64
			if err := tx.Model(&ImageAsset{}).Where("workspace_id=? AND id IN ?", w, in.ImageAssetIDs).Count(&count).Error; err != nil {
				return err
			}
			if count != int64(len(in.ImageAssetIDs)) {
				return fmt.Errorf("one or more image assets not found")
			}
			set := ImageSet{WorkspaceID: w, Name: "Listing images", AssetIDs: j(in.ImageAssetIDs)}
			if err := tx.Create(&set).Error; err != nil {
				return err
			}
			row.ImageSetID = &set.ID
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		gpsr := ListingGPSR{WorkspaceID: w, ListingDraftID: row.ID, GPSRProfileID: in.GPSRProfileID}
		return tx.Create(&gpsr).Error
	})
	return &row, err
}

var bannedClaims = []*regexp.Regexp{regexp.MustCompile(`(?i)\bheil(?:t|en|ung)\b`), regexp.MustCompile(`(?i)\bgarantiert(?:e|er|es)?\b`), regexp.MustCompile(`(?i)\b100\s*%\s*sicher\b`)}

func ValidateContent(content ListingContentVersion) []string {
	errs := []string{}
	title := strings.TrimSpace(content.Title)
	if title == "" {
		errs = append(errs, "title is required")
	}
	if len([]rune(title)) > 80 {
		errs = append(errs, "title exceeds 80 characters")
	}
	for _, r := range bannedClaims {
		if r.MatchString(title + " " + content.Description) {
			errs = append(errs, "content contains a prohibited claim")
			break
		}
	}
	words := strings.Fields(strings.ToLower(title))
	seen := map[string]bool{}
	for _, word := range words {
		if len(word) > 3 && seen[word] {
			errs = append(errs, "title contains duplicate terms")
			break
		}
		seen[word] = true
	}
	return errs
}

// ValidateFactfulness flags generated specifics that are not backed by the
// immutable source snapshot. Free-form copy remains review-gated by the prompt
// and operator because semantic entailment is intentionally not guessed here.
func ValidateFactfulness(content ListingContentVersion, facts SourceFacts) []string {
	var specifics map[string]string
	if err := json.Unmarshal(content.Specifics, &specifics); err != nil {
		return []string{"generated specifics are invalid"}
	}
	errs := []string{}
	for key, value := range specifics {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		backed := strings.EqualFold(strings.TrimSpace(key), "marke") && strings.EqualFold(value, strings.TrimSpace(facts.Brand))
		if !backed {
			if sourceValue, ok := facts.Attributes[key]; ok {
				backed = strings.EqualFold(value, strings.TrimSpace(fmt.Sprint(sourceValue)))
			}
		}
		if !backed {
			errs = append(errs, "generated specific is not backed by source facts: "+key)
		}
	}
	sort.Strings(errs)
	return errs
}

type ValidationInput struct {
	GPSROverride   bool       `json:"gpsrOverride"`
	OverrideReason string     `json:"overrideReason"`
	AdminID        *uuid.UUID `json:"-"`
}
type ValidationResult struct {
	State          DraftState `json:"state"`
	Errors         []string   `json:"errors"`
	GPSROverridden bool       `json:"gpsrOverridden"`
}

func (s *Service) Validate(ctx context.Context, w int64, id uuid.UUID, in ValidationInput) (*ValidationResult, error) {
	var out ValidationResult
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var d ListingDraft
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND workspace_id=?", id, w).First(&d).Error; err != nil {
			return err
		}
		errs := []string{}
		if strings.TrimSpace(d.Category) == "" {
			errs = append(errs, "category is required")
		}
		if d.PriceCents == nil || *d.PriceCents <= 0 {
			errs = append(errs, "positive priceCents is required")
		}
		if d.ImageSetID == nil {
			errs = append(errs, "image set is required")
		}
		var required []string
		var specifics map[string]string
		_ = json.Unmarshal(d.RequiredSpecifics, &required)
		_ = json.Unmarshal(d.Specifics, &specifics)
		for _, key := range required {
			if strings.TrimSpace(specifics[key]) == "" {
				errs = append(errs, "required specific missing: "+key)
			}
		}
		if d.CurrentContentVersionID == nil {
			errs = append(errs, "generated content is required")
		} else {
			var v ListingContentVersion
			if err := tx.Where("id=? AND workspace_id=?", *d.CurrentContentVersionID, w).First(&v).Error; err != nil {
				return err
			}
			errs = append(errs, ValidateContent(v)...)
			if s.Sources != nil {
				facts, err := s.Sources.ReadSource(ctx, w, d.SourceProductID)
				if err != nil {
					return err
				}
				errs = append(errs, ValidateFactfulness(v, facts)...)
			}
		}
		var gpsr ListingGPSR
		if err := tx.Where("listing_draft_id=? AND workspace_id=?", d.ID, w).First(&gpsr).Error; err != nil {
			return err
		}
		validGPSR := gpsr.Override
		if gpsr.GPSRProfileID != nil {
			var p GPSRProfile
			if err := tx.Where("id=? AND workspace_id=?", *gpsr.GPSRProfileID, w).First(&p).Error; err == nil {
				var documents []string
				_ = json.Unmarshal(p.DocumentReferences, &documents)
				validGPSR = strings.TrimSpace(p.ManufacturerName) != "" && strings.TrimSpace(p.ManufacturerAddress) != "" && strings.TrimSpace(p.ResponsiblePersonName) != "" && strings.TrimSpace(p.ResponsiblePersonAddress) != "" && strings.TrimSpace(p.SafetyInformation) != "" && len(documents) > 0
			}
		}
		if !validGPSR && in.GPSROverride {
			enabled := false
			if s.Settings != nil {
				plain, _ := s.Settings.PlainByGroup(ctx, w, "mindbay_listing")
				enabled = strings.EqualFold(strings.TrimSpace(plain["gpsr_override_enabled"]), "true")
			}
			if !enabled {
				return fmt.Errorf("GPSR override is disabled for this workspace")
			}
			if strings.TrimSpace(in.OverrideReason) == "" {
				return fmt.Errorf("GPSR override reason is required")
			}
			now := time.Now().UTC()
			gpsr.Override = true
			gpsr.OverrideReason = strings.TrimSpace(in.OverrideReason)
			gpsr.OverriddenAt = &now
			gpsr.OverriddenBy = in.AdminID
			if err := tx.Save(&gpsr).Error; err != nil {
				return err
			}
			validGPSR = true
			out.GPSROverridden = true
		}
		if !validGPSR {
			errs = append(errs, "valid GPSR profile or audited override is required")
		}
		sort.Strings(errs)
		next := StateBlocked
		if len(errs) == 0 {
			switch d.State {
			case StateBlocked:
				next = StateDrafting
			case StateDrafting:
				next = StateNeedsReview
			case StateNeedsReview, StateReady:
				next = StateReady
			default:
				return fmt.Errorf("draft state %s cannot be validated", d.State)
			}
		}
		if next != d.State && !CanTransition(d.State, next) {
			return fmt.Errorf("invalid listing transition %s -> %s", d.State, next)
		}
		d.State = next
		d.ValidationErrors = j(errs)
		if err := tx.Save(&d).Error; err != nil {
			return err
		}
		out.State = d.State
		out.Errors = errs
		return nil
	})
	return &out, err
}

func (s *Service) Generate(ctx context.Context, w int64, id uuid.UUID) (*ListingContentVersion, error) {
	if s.Generator == nil || s.Sources == nil {
		return nil, fmt.Errorf("listing generator unavailable")
	}
	var d ListingDraft
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", id, w).First(&d).Error; err != nil {
		return nil, err
	}
	if d.State == StateReady || d.State == StateScheduled || d.State == StatePublishing || d.State == StatePublished || d.State == StatePaused || d.State == StateEnded {
		return nil, fmt.Errorf("listing content is immutable in state %s", d.State)
	}
	facts, err := s.Sources.ReadSource(ctx, w, d.SourceProductID)
	if err != nil {
		return nil, err
	}
	generated, generator, prompt, err := s.Generator.Generate(ctx, w, facts)
	if err != nil {
		return nil, err
	}
	row := ListingContentVersion{WorkspaceID: w, ListingDraftID: d.ID, Title: strings.TrimSpace(generated.Title), Description: strings.TrimSpace(generated.Description), Specifics: j(generated.Specifics), Generator: generator, PromptCode: prompt}
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&ListingContentVersion{}).Where("listing_draft_id=? AND workspace_id=?", d.ID, w).Select("COALESCE(MAX(version),0)").Scan(&row.Version).Error; err != nil {
			return err
		}
		row.Version++
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		next := StateNeedsReview
		if d.State == StateBlocked {
			next = StateDrafting
		}
		if next != d.State && !CanTransition(d.State, next) {
			return fmt.Errorf("invalid listing transition %s -> %s", d.State, next)
		}
		updates := map[string]any{"current_content_version_id": row.ID, "state": next, "specifics": row.Specifics}
		return tx.Model(&ListingDraft{}).Where("id=? AND workspace_id=?", d.ID, w).Updates(updates).Error
	})
	return &row, err
}

// TransitionPublication is the only Phase 2 entry point allowed to move a
// listing through publication states. Callers may pass a transaction so slot,
// job and listing state changes remain atomic.
func (s *Service) TransitionPublication(ctx context.Context, tx *gorm.DB, workspaceID int64, id uuid.UUID, from, to DraftState, contentVersionID uuid.UUID) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("listing studio unavailable")
	}
	if tx == nil {
		tx = s.DB
	}
	var draft ListingDraft
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND workspace_id=?", id, workspaceID).First(&draft).Error; err != nil {
		return err
	}
	if draft.State != from {
		return fmt.Errorf("listing state changed: expected %s, got %s", from, draft.State)
	}
	if !CanTransition(from, to) {
		return fmt.Errorf("invalid listing transition %s -> %s", from, to)
	}
	if contentVersionID == uuid.Nil || draft.CurrentContentVersionID == nil || *draft.CurrentContentVersionID != contentVersionID {
		return fmt.Errorf("listing content version changed")
	}
	if from == StateReady && len(draft.ValidationErrors) > 0 && string(draft.ValidationErrors) != "[]" && string(draft.ValidationErrors) != "null" {
		return fmt.Errorf("listing has validation blockers")
	}
	return tx.WithContext(ctx).Model(&ListingDraft{}).Where("id=? AND workspace_id=? AND state=?", id, workspaceID, from).Update("state", to).Error
}

func (s *Service) Get(ctx context.Context, w int64, id uuid.UUID) (*ListingDraft, []ListingContentVersion, error) {
	var d ListingDraft
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", id, w).First(&d).Error; err != nil {
		return nil, nil, err
	}
	var v []ListingContentVersion
	if err := s.DB.WithContext(ctx).Where("listing_draft_id=? AND workspace_id=?", id, w).Order("version DESC").Find(&v).Error; err != nil {
		return nil, nil, err
	}
	return &d, v, nil
}

type DraftPage struct {
	Items      []ListingDraft `json:"items"`
	NextCursor string         `json:"nextCursor,omitempty"`
	HasMore    bool           `json:"hasMore"`
	Limit      int            `json:"limit"`
}

func (s *Service) List(ctx context.Context, w int64, state DraftState, limit int, cursorRaw string) (*DraftPage, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	scope := pagination.Fingerprint(map[string]any{"resource": "phase1-listing-drafts", "state": state})
	cur, err := pagination.DecodeCursor(cursorRaw, w, "", scope)
	if err != nil {
		return nil, err
	}
	tx := s.DB.WithContext(ctx).Where("workspace_id=?", w)
	if state != "" {
		tx = tx.Where("state=?", state)
	}
	tx, err = pagination.ApplyDescKeyset(tx, "created_at", "id", cur)
	if err != nil {
		return nil, err
	}
	var rows []ListingDraft
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
		next, err = pagination.BuildNextCursor(true, w, "", scope, "created_at", last.CreatedAt, last.ID.String())
	}
	return &DraftPage{Items: rows, NextCursor: next, HasMore: more, Limit: limit}, err
}
