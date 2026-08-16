package publication

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/modules/listingstudio"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type publisherFunc func(context.Context, int64, uuid.UUID, string, PublishPayload) (PublishOutcome, error)

func (f publisherFunc) Publish(ctx context.Context, workspaceID int64, shopID uuid.UUID, mode string, payload PublishPayload) (PublishOutcome, error) {
	return f(ctx, workspaceID, shopID, mode, payload)
}

func scheduledPublication(t *testing.T, withGPSR bool) (*gorm.DB, PublicationJob, listingstudio.ListingDraft) {
	t.Helper()
	db := publicationTestDB(t)
	require.NoError(t, db.AutoMigrate(&listingstudio.GPSRProfile{}, &listingstudio.ListingGPSR{}, &listingstudio.ImageAsset{}, &listingstudio.ImageSet{}, &MarketplaceListing{}, &ListingSnapshot{}))
	now := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	draft, version := readyDraft(t, db, 12, now.Add(-24*time.Hour))
	asset := listingstudio.ImageAsset{WorkspaceID: 12, OriginURL: "https://images.example/item.jpg", SHA256: "abc", StorageKey: "mindbay/item.jpg", ContentType: "image/jpeg", SizeBytes: 1024}
	require.NoError(t, db.Create(&asset).Error)
	assetIDs, err := json.Marshal([]uuid.UUID{asset.ID})
	require.NoError(t, err)
	imageSet := listingstudio.ImageSet{WorkspaceID: 12, Name: "publish", AssetIDs: datatypes.JSON(assetIDs)}
	require.NoError(t, db.Create(&imageSet).Error)
	price := int64(2999)
	require.NoError(t, db.Model(&draft).Updates(map[string]any{"price_cents": price, "image_set_id": imageSet.ID, "category": "123"}).Error)
	draft.PriceCents, draft.ImageSetID = &price, &imageSet.ID
	if withGPSR {
		profile := listingstudio.GPSRProfile{WorkspaceID: 12, Name: "Garden", ManufacturerName: "Maker", ManufacturerAddress: "Street 1, 10115 Berlin", ResponsiblePersonName: "EU Contact", ResponsiblePersonAddress: "Street 2, 10115 Berlin", SafetyInformation: "Keep dry", DocumentReferences: datatypes.JSON([]byte(`["doc://safety"]`))}
		require.NoError(t, db.Create(&profile).Error)
		require.NoError(t, db.Create(&listingstudio.ListingGPSR{WorkspaceID: 12, ListingDraftID: draft.ID, GPSRProfileID: &profile.ID}).Error)
	}
	svc := &Service{DB: db, Listings: &listingstudio.Service{DB: db}, Now: func() time.Time { return now }}
	result, err := svc.Apply(context.Background(), 12, "publish", ApplyInput{ShopID: uuid.New(), Marketplace: "EBAY_DE", Slots: []PreviewSlot{{ListingDraftID: draft.ID, ContentVersionID: version.ID, ScheduledFor: now.Add(time.Hour)}}, PublishConfig: []byte(`{"merchantLocationKey":"loc","paymentPolicyId":"pay","returnPolicyId":"return","fulfillmentPolicyId":"ship"}`)})
	require.NoError(t, err)
	require.Len(t, result.Jobs, 1)
	return db, result.Jobs[0], draft
}

func TestPublishPersistsMarketplaceListingAndTerminalStates(t *testing.T) {
	db, job, draft := scheduledPublication(t, true)
	publisher := publisherFunc(func(_ context.Context, workspaceID int64, shopID uuid.UUID, mode string, payload PublishPayload) (PublishOutcome, error) {
		require.Equal(t, int64(12), workspaceID)
		require.Equal(t, job.ShopID, shopID)
		require.Equal(t, "SIMULATED_CHECKOUT", mode)
		require.Equal(t, int64(2999), payload.PriceCents)
		return PublishOutcome{OfferID: "offer-1", ListingID: "listing-1", ListingURL: "https://www.sandbox.ebay.de/itm/listing-1", RequestArtifact: map[string]any{"sku": payload.SKU}, ResponseArtifact: map[string]any{"listingId": "listing-1"}}, nil
	})
	svc := &PublishService{DB: db, Listings: &listingstudio.Service{DB: db}, Publisher: publisher, AutomationMode: "SIMULATED_CHECKOUT"}
	listing, err := svc.Publish(context.Background(), 12, job.ID)
	require.NoError(t, err)
	require.Equal(t, "listing-1", listing.ExternalListingID)

	var persistedDraft listingstudio.ListingDraft
	require.NoError(t, db.First(&persistedDraft, "id=?", draft.ID).Error)
	require.Equal(t, listingstudio.StatePublished, persistedDraft.State)
	var persistedJob PublicationJob
	require.NoError(t, db.First(&persistedJob, "id=?", job.ID).Error)
	require.Equal(t, JobPublished, persistedJob.Status)
	var slot CalendarSlot
	require.NoError(t, db.First(&slot, "id=?", job.CalendarSlotID).Error)
	require.Equal(t, SlotPublished, slot.Status)
	var transitionCount int64
	require.NoError(t, db.Model(&PublicationTransitionEvent{}).Count(&transitionCount).Error)
	require.EqualValues(t, 3, transitionCount)
}

func TestPublishDryRunReturnsListingToReadyWithoutMarketplaceRow(t *testing.T) {
	db, job, draft := scheduledPublication(t, true)
	calls := 0
	publisher := publisherFunc(func(_ context.Context, _ int64, _ uuid.UUID, mode string, _ PublishPayload) (PublishOutcome, error) {
		calls++
		require.Equal(t, "DRY_RUN", mode)
		return PublishOutcome{DryRun: true, ResponseArtifact: map[string]any{"mutatingCalls": 0}}, nil
	})
	svc := &PublishService{DB: db, Listings: &listingstudio.Service{DB: db}, Publisher: publisher, AutomationMode: "DRY_RUN"}
	listing, err := svc.Publish(context.Background(), 12, job.ID)
	require.NoError(t, err)
	require.Nil(t, listing)
	require.Equal(t, 1, calls)
	var count int64
	require.NoError(t, db.Model(&MarketplaceListing{}).Count(&count).Error)
	require.Zero(t, count)
	var persistedDraft listingstudio.ListingDraft
	require.NoError(t, db.First(&persistedDraft, "id=?", draft.ID).Error)
	require.Equal(t, listingstudio.StateReady, persistedDraft.State)
	var persistedJob PublicationJob
	require.NoError(t, db.First(&persistedJob, "id=?", job.ID).Error)
	require.Equal(t, JobDryRun, persistedJob.Status)
	var transitionCount int64
	require.NoError(t, db.Model(&PublicationTransitionEvent{}).Count(&transitionCount).Error)
	require.EqualValues(t, 2, transitionCount)
}

func TestPublishBlocksMissingGPSRBeforeProvider(t *testing.T) {
	db, job, _ := scheduledPublication(t, false)
	calls := 0
	svc := &PublishService{DB: db, Listings: &listingstudio.Service{DB: db}, Publisher: publisherFunc(func(context.Context, int64, uuid.UUID, string, PublishPayload) (PublishOutcome, error) {
		calls++
		return PublishOutcome{}, nil
	}), AutomationMode: "SIMULATED_CHECKOUT"}
	_, err := svc.Publish(context.Background(), 12, job.ID)
	require.ErrorContains(t, err, "complete GPSR profile or audited override")
	require.Zero(t, calls)
}

func TestPublishAllowsPersistedAuditedGPSROverride(t *testing.T) {
	db, job, draft := scheduledPublication(t, false)
	require.NoError(t, db.Create(&listingstudio.ListingGPSR{WorkspaceID: 12, ListingDraftID: draft.ID, Override: true, OverrideReason: "Operator reviewed non-GPSR category"}).Error)
	calls := 0
	svc := &PublishService{DB: db, Listings: &listingstudio.Service{DB: db}, Publisher: publisherFunc(func(_ context.Context, _ int64, _ uuid.UUID, _ string, payload PublishPayload) (PublishOutcome, error) {
		calls++
		require.True(t, payload.GPSROverridden)
		require.Empty(t, payload.ManufacturerName)
		return PublishOutcome{OfferID: "offer-override", ListingID: "listing-override"}, nil
	}), AutomationMode: "SIMULATED_CHECKOUT"}

	listing, err := svc.Publish(context.Background(), 12, job.ID)
	require.NoError(t, err)
	require.Equal(t, "listing-override", listing.ExternalListingID)
	require.Equal(t, 1, calls)
}
