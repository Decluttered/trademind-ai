package publication

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/modules/listingstudio"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PublishPayload struct {
	SKU                   string
	Title                 string
	Description           string
	CategoryID            string
	Condition             string
	Currency              string
	PriceCents            int64
	Quantity              int
	ImageURLs             []string
	Aspects               map[string][]string
	Marketplace           string
	MerchantLocationKey   string
	PaymentPolicyID       string
	ReturnPolicyID        string
	FulfillmentPolicyID   string
	ManufacturerName      string
	ManufacturerAddress   string
	ResponsiblePersonName string
	ResponsibleAddress    string
	SafetyInformation     string
	SafetyStatementIDs    []string
	GPSROverridden        bool
}

var safetyStatementPattern = regexp.MustCompile(`^EBPSS[0-9]+$`)

type PublishOutcome struct {
	OfferID          string
	ListingID        string
	ListingURL       string
	DryRun           bool
	RequestArtifact  map[string]any
	ResponseArtifact map[string]any
}

type ListingPublisher interface {
	Publish(context.Context, int64, uuid.UUID, string, PublishPayload) (PublishOutcome, error)
}

type PublishService struct {
	DB             *gorm.DB
	Listings       *listingstudio.Service
	Publisher      ListingPublisher
	AutomationMode string
	Now            func() time.Time
}

func (s *PublishService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *PublishService) Revalidate(ctx context.Context, workspaceID int64, jobID uuid.UUID) error {
	if s == nil || s.DB == nil || s.Listings == nil {
		return fmt.Errorf("publication service unavailable")
	}
	var job PublicationJob
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", jobID, workspaceID).First(&job).Error; err != nil {
		return err
	}
	_, err := s.buildPayload(ctx, workspaceID, job)
	return err
}

func (s *PublishService) Reconcile(ctx context.Context, workspaceID int64, jobID uuid.UUID) (*MarketplaceListing, error) {
	var listing MarketplaceListing
	err := s.DB.WithContext(ctx).Where("workspace_id=? AND publication_job_id=?", workspaceID, jobID).First(&listing).Error
	if err == nil {
		return &listing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	var job PublicationJob
	if findErr := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", jobID, workspaceID).First(&job).Error; findErr != nil {
		return nil, findErr
	}
	if job.Status == JobDryRun {
		return nil, nil
	}
	return nil, fmt.Errorf("publication result is not reconciled (job state %s)", job.Status)
}

func (s *PublishService) Publish(ctx context.Context, workspaceID int64, jobID uuid.UUID) (*MarketplaceListing, error) {
	if s == nil || s.DB == nil || s.Listings == nil || s.Publisher == nil {
		return nil, fmt.Errorf("publication service unavailable")
	}
	var existing MarketplaceListing
	if err := s.DB.WithContext(ctx).Where("workspace_id=? AND publication_job_id=?", workspaceID, jobID).First(&existing).Error; err == nil {
		return &existing, nil
	}
	var job PublicationJob
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", jobID, workspaceID).First(&job).Error; err != nil {
		return nil, err
	}
	payload, err := s.buildPayload(ctx, workspaceID, job)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(strings.TrimSpace(s.AutomationMode), "DRY_RUN") {
		outcome, publishErr := s.Publisher.Publish(ctx, workspaceID, job.ShopID, s.AutomationMode, payload)
		if publishErr != nil {
			return nil, publishErr
		}
		return nil, s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			requestJSON, _ := json.Marshal(outcome.RequestArtifact)
			responseJSON, _ := json.Marshal(outcome.ResponseArtifact)
			if err := tx.Model(&PublicationJob{}).Where("id=? AND workspace_id=?", job.ID, workspaceID).Updates(map[string]any{"status": JobDryRun, "request_artifact": datatypes.JSON(requestJSON), "response_artifact": datatypes.JSON(responseJSON)}).Error; err != nil {
				return err
			}
			if err := tx.Model(&CalendarSlot{}).Where("id=? AND workspace_id=?", job.CalendarSlotID, workspaceID).Update("status", SlotHeld).Error; err != nil {
				return err
			}
			if err := s.Listings.TransitionPublication(ctx, tx, workspaceID, job.ListingDraftID, listingstudio.StateScheduled, listingstudio.StateReady, job.ContentVersionID); err != nil {
				return err
			}
			return recordWorkflowTransition(tx, job, listingstudio.StateScheduled, listingstudio.StateReady)
		})
	}
	if err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locked, lockErr := lockJob(tx, workspaceID, jobID)
		if lockErr != nil {
			return lockErr
		}
		if locked.Status == JobPublished {
			return nil
		}
		if locked.Status != JobScheduled && locked.Status != JobFailed {
			return fmt.Errorf("publication job is not runnable in state %s", locked.Status)
		}
		if locked.Status == JobFailed {
			if err := s.Listings.TransitionPublication(ctx, tx, workspaceID, job.ListingDraftID, listingstudio.StateReady, listingstudio.StateScheduled, job.ContentVersionID); err != nil {
				return err
			}
			if err := recordWorkflowTransition(tx, job, listingstudio.StateReady, listingstudio.StateScheduled); err != nil {
				return err
			}
		}
		if err := s.Listings.TransitionPublication(ctx, tx, workspaceID, job.ListingDraftID, listingstudio.StateScheduled, listingstudio.StatePublishing, job.ContentVersionID); err != nil {
			return err
		}
		if err := recordWorkflowTransition(tx, job, listingstudio.StateScheduled, listingstudio.StatePublishing); err != nil {
			return err
		}
		if err := tx.Model(&PublicationJob{}).Where("id=?", job.ID).Updates(map[string]any{"status": JobPublishing, "attempt": gorm.Expr("attempt + 1"), "error_class": "", "error_message": ""}).Error; err != nil {
			return err
		}
		return tx.Model(&CalendarSlot{}).Where("id=?", job.CalendarSlotID).Update("status", SlotPublishing).Error
	}); err != nil {
		return nil, err
	}
	outcome, err := s.Publisher.Publish(ctx, workspaceID, job.ShopID, s.AutomationMode, payload)
	if err != nil {
		_ = s.fail(ctx, workspaceID, job, err)
		return nil, err
	}
	var row MarketplaceListing
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		requestJSON, _ := json.Marshal(outcome.RequestArtifact)
		responseJSON, _ := json.Marshal(outcome.ResponseArtifact)
		now := s.now()
		row = MarketplaceListing{WorkspaceID: workspaceID, ListingDraftID: job.ListingDraftID, PublicationJobID: job.ID, Marketplace: "ebay", ExternalListingID: outcome.ListingID, ExternalOfferID: outcome.OfferID, ExternalURL: outcome.ListingURL, SKU: payload.SKU, Status: "ACTIVE", PriceCents: payload.PriceCents, Currency: payload.Currency}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		snapshot := ListingSnapshot{WorkspaceID: workspaceID, ListingDraftID: job.ListingDraftID, PublicationJobID: job.ID, Kind: "PUBLISHED", Payload: datatypes.JSON(responseJSON)}
		if err := tx.Create(&snapshot).Error; err != nil {
			return err
		}
		if err := tx.Model(&PublicationJob{}).Where("id=?", job.ID).Updates(map[string]any{"status": JobPublished, "published_at": now, "request_artifact": datatypes.JSON(requestJSON), "response_artifact": datatypes.JSON(responseJSON)}).Error; err != nil {
			return err
		}
		if err := tx.Model(&CalendarSlot{}).Where("id=?", job.CalendarSlotID).Update("status", SlotPublished).Error; err != nil {
			return err
		}
		if err := s.Listings.TransitionPublication(ctx, tx, workspaceID, job.ListingDraftID, listingstudio.StatePublishing, listingstudio.StatePublished, job.ContentVersionID); err != nil {
			return err
		}
		return recordWorkflowTransition(tx, job, listingstudio.StatePublishing, listingstudio.StatePublished)
	})
	return &row, err
}

func (s *PublishService) fail(ctx context.Context, workspaceID int64, job PublicationJob, cause error) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		message := cause.Error()
		if len(message) > 1000 {
			message = message[:1000]
		}
		errorClass := "PROVIDER"
		if classified, ok := cause.(interface{ ErrorClass() string }); ok && strings.TrimSpace(classified.ErrorClass()) != "" {
			errorClass = classified.ErrorClass()
		}
		if err := tx.Model(&PublicationJob{}).Where("id=? AND workspace_id=?", job.ID, workspaceID).Updates(map[string]any{"status": JobFailed, "error_class": errorClass, "error_message": message}).Error; err != nil {
			return err
		}
		if err := tx.Model(&CalendarSlot{}).Where("id=? AND workspace_id=?", job.CalendarSlotID, workspaceID).Update("status", SlotFailed).Error; err != nil {
			return err
		}
		if err := s.Listings.TransitionPublication(ctx, tx, workspaceID, job.ListingDraftID, listingstudio.StatePublishing, listingstudio.StatePublishFail, job.ContentVersionID); err != nil {
			return err
		}
		if err := recordWorkflowTransition(tx, job, listingstudio.StatePublishing, listingstudio.StatePublishFail); err != nil {
			return err
		}
		if err := s.Listings.TransitionPublication(ctx, tx, workspaceID, job.ListingDraftID, listingstudio.StatePublishFail, listingstudio.StateReady, job.ContentVersionID); err != nil {
			return err
		}
		return recordWorkflowTransition(tx, job, listingstudio.StatePublishFail, listingstudio.StateReady)
	})
}

func recordWorkflowTransition(tx *gorm.DB, job PublicationJob, from, to listingstudio.DraftState) error {
	correlationID := strings.TrimSpace(job.WorkflowID)
	if correlationID == "" {
		correlationID = job.CorrelationID
	}
	return recordTransition(tx, job, from, to, "temporal-worker", correlationID)
}

func (s *PublishService) buildPayload(ctx context.Context, workspaceID int64, job PublicationJob) (PublishPayload, error) {
	var draft listingstudio.ListingDraft
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", job.ListingDraftID, workspaceID).First(&draft).Error; err != nil {
		return PublishPayload{}, err
	}
	retryReady := job.Status == JobFailed && draft.State == listingstudio.StateReady
	if draft.CurrentContentVersionID == nil || *draft.CurrentContentVersionID != job.ContentVersionID || (draft.State != listingstudio.StateScheduled && draft.State != listingstudio.StatePublishing && !retryReady) {
		return PublishPayload{}, fmt.Errorf("listing is no longer the scheduled immutable version")
	}
	if draft.PriceCents == nil || *draft.PriceCents <= 0 || draft.ImageSetID == nil {
		return PublishPayload{}, fmt.Errorf("listing price and image set are required")
	}
	var version listingstudio.ListingContentVersion
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=? AND listing_draft_id=?", job.ContentVersionID, workspaceID, draft.ID).First(&version).Error; err != nil {
		return PublishPayload{}, err
	}
	var imageSet listingstudio.ImageSet
	if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", *draft.ImageSetID, workspaceID).First(&imageSet).Error; err != nil {
		return PublishPayload{}, err
	}
	var imageIDs []uuid.UUID
	_ = json.Unmarshal(imageSet.AssetIDs, &imageIDs)
	var assets []listingstudio.ImageAsset
	if len(imageIDs) == 0 || s.DB.WithContext(ctx).Where("workspace_id=? AND id IN ?", workspaceID, imageIDs).Find(&assets).Error != nil || len(assets) == 0 {
		return PublishPayload{}, fmt.Errorf("at least one listing image is required")
	}
	var link listingstudio.ListingGPSR
	if err := s.DB.WithContext(ctx).Where("workspace_id=? AND listing_draft_id=?", workspaceID, draft.ID).First(&link).Error; err != nil {
		return PublishPayload{}, fmt.Errorf("complete GPSR profile or audited override is required")
	}
	var gpsr listingstudio.GPSRProfile
	if link.GPSRProfileID != nil {
		if err := s.DB.WithContext(ctx).Where("id=? AND workspace_id=?", *link.GPSRProfileID, workspaceID).First(&gpsr).Error; err != nil {
			return PublishPayload{}, fmt.Errorf("complete GPSR profile is required")
		}
	}
	completeGPSR := strings.TrimSpace(gpsr.ManufacturerName) != "" && strings.TrimSpace(gpsr.ManufacturerAddress) != "" && strings.TrimSpace(gpsr.ResponsiblePersonName) != "" && strings.TrimSpace(gpsr.ResponsiblePersonAddress) != "" && strings.TrimSpace(gpsr.SafetyInformation) != ""
	if !completeGPSR && !link.Override {
		return PublishPayload{}, fmt.Errorf("complete GPSR profile or audited override is required")
	}
	if !completeGPSR {
		gpsr = listingstudio.GPSRProfile{}
	}
	config := map[string]any{}
	_ = json.Unmarshal(job.PublishConfig, &config)
	get := func(key, fallback string) string {
		if value := strings.TrimSpace(fmt.Sprint(config[key])); value != "" && value != "<nil>" {
			return value
		}
		return fallback
	}
	quantity := 1
	if _, err := fmt.Sscan(get("quantity", "1"), &quantity); err != nil || quantity < 1 {
		return PublishPayload{}, fmt.Errorf("positive publish quantity is required")
	}
	var specifics map[string]string
	_ = json.Unmarshal(version.Specifics, &specifics)
	aspects := map[string][]string{}
	for key, value := range specifics {
		if strings.TrimSpace(value) != "" {
			aspects[key] = []string{strings.TrimSpace(value)}
		}
	}
	images := make([]string, 0, len(assets))
	for _, asset := range assets {
		if strings.HasPrefix(asset.OriginURL, "https://") {
			images = append(images, asset.OriginURL)
		}
	}
	if len(images) == 0 {
		return PublishPayload{}, fmt.Errorf("eBay requires at least one public HTTPS image")
	}
	statementIDs := []string{}
	for _, value := range strings.FieldsFunc(get("productSafetyStatementIds", get("product_safety_statement_ids", "")), func(r rune) bool { return r == ',' || r == ';' || r == ' ' || r == '\n' }) {
		value = strings.ToUpper(strings.TrimSpace(value))
		if !safetyStatementPattern.MatchString(value) {
			return PublishPayload{}, fmt.Errorf("invalid eBay product safety statement id %q", value)
		}
		statementIDs = append(statementIDs, value)
	}
	sku := "MB-" + strings.ToUpper(strings.ReplaceAll(draft.ID.String(), "-", ""))[:20]
	return PublishPayload{SKU: sku, Title: version.Title, Description: version.Description, CategoryID: draft.Category, Condition: get("condition", "NEW"), Currency: draft.Currency, PriceCents: *draft.PriceCents, Quantity: quantity, ImageURLs: images, Aspects: aspects, Marketplace: job.Marketplace, MerchantLocationKey: get("merchantLocationKey", ""), PaymentPolicyID: get("paymentPolicyId", ""), ReturnPolicyID: get("returnPolicyId", ""), FulfillmentPolicyID: get("fulfillmentPolicyId", ""), ManufacturerName: gpsr.ManufacturerName, ManufacturerAddress: gpsr.ManufacturerAddress, ResponsiblePersonName: gpsr.ResponsiblePersonName, ResponsibleAddress: gpsr.ResponsiblePersonAddress, SafetyInformation: gpsr.SafetyInformation, SafetyStatementIDs: statementIDs, GPSROverridden: link.Override}, nil
}
