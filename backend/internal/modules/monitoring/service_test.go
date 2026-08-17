package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/modules/catalog"
	"github.com/trademind-ai/trademind/backend/internal/modules/listingstudio"
	"github.com/trademind-ai/trademind/backend/internal/modules/publication"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type fakeOfferGateway struct {
	reads   int
	updates int
}

func (f *fakeOfferGateway) ReadOffer(context.Context, int64, publication.PublicationJob, publication.MarketplaceListing) (OfferSnapshot, error) {
	f.reads++
	return OfferSnapshot{PriceCents: 3_500, Quantity: 2, Status: "PUBLISHED", Raw: map[string]any{"offerId": "offer-1", "price": "35.00"}}, nil
}
func (f *fakeOfferGateway) UpdateOffer(_ context.Context, _ int64, _ publication.PublicationJob, _ publication.MarketplaceListing, price int64) (OfferSnapshot, map[string]any, bool, error) {
	f.updates++
	return OfferSnapshot{PriceCents: price, Quantity: 2, Status: "DRY_RUN", Raw: map[string]any{"status": "DRY_RUN"}}, map[string]any{"priceCents": price}, true, nil
}

func monitoringFixture(t *testing.T) (*gorm.DB, publication.MarketplaceListing, PriceRule) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&catalog.SourceProduct{}, &catalog.ProductSnapshot{}, &listingstudio.ListingDraft{}, &publication.PublicationJob{}, &publication.MarketplaceListing{}, &publication.ListingSnapshot{}, &MonitorRun{}, &PriceRule{}, &PriceDecision{}, &ProfitLedgerEntry{}))
	source := catalog.SourceProduct{WorkspaceID: 7, Source: "amazon.de", ExternalID: "B000TEST01", CanonicalURL: "https://www.amazon.de/dp/B000TEST01"}
	require.NoError(t, db.Create(&source).Error)
	price := int64(2_000)
	snapshot := catalog.ProductSnapshot{WorkspaceID: 7, SourceProductID: source.ID, CapturedAt: time.Now(), Title: "Fixture", Currency: "EUR", PriceCents: &price, Availability: "Auf Lager", Images: datatypes.JSON([]byte("[]")), Attributes: datatypes.JSON([]byte("{}")), Variants: datatypes.JSON([]byte("[]")), Raw: datatypes.JSON([]byte("{}"))}
	require.NoError(t, db.Create(&snapshot).Error)
	require.NoError(t, db.Model(&source).Update("current_snapshot_id", snapshot.ID).Error)
	draft := listingstudio.ListingDraft{WorkspaceID: 7, SourceProductID: source.ID, State: listingstudio.StatePublished, Currency: "EUR"}
	require.NoError(t, db.Create(&draft).Error)
	job := publication.PublicationJob{WorkspaceID: 7, CalendarSlotID: uuid.New(), ListingDraftID: draft.ID, ContentVersionID: uuid.New(), ShopID: uuid.New(), Marketplace: "EBAY_DE", Status: publication.JobPublished, IdempotencyKey: "publish-fixture", ScheduledFor: time.Now(), PublishConfig: datatypes.JSON([]byte("{}")), Actor: "test", CorrelationID: "test", PayloadHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	require.NoError(t, db.Create(&job).Error)
	listing := publication.MarketplaceListing{WorkspaceID: 7, ListingDraftID: draft.ID, PublicationJobID: job.ID, Marketplace: "ebay", ExternalListingID: "listing-1", ExternalOfferID: "offer-1", SKU: "MB-1", Status: "ACTIVE", PriceCents: 3_500, Currency: "EUR"}
	require.NoError(t, db.Create(&listing).Error)
	rule := PriceRule{WorkspaceID: 7, Name: "Default", Version: 1, IdempotencyKey: "rule-fixture", MinMarginBasisPoints: 1_500, TargetMarginBasisPoints: 2_000, MaxDeltaCents: 1_000, MaxDeltaBasisPoints: 3_000, PlatformFeeBasisPoints: 1_200, ReserveCents: 100, AutoApply: true}
	require.NoError(t, db.Create(&rule).Error)
	return db, listing, rule
}

func TestRunIsIdempotentAndWritesExpectedLedger(t *testing.T) {
	db, listing, rule := monitoringFixture(t)
	gateway := &fakeOfferGateway{}
	service := &Service{DB: db, Offers: gateway, Now: func() time.Time { return time.Unix(1_700_000_000, 0) }}
	input := RunInput{MarketplaceListingID: listing.ID, PriceRuleID: rule.ID, Trigger: "manual"}
	first, err := service.Run(context.Background(), 7, "monitor-key", "trace-1", input)
	require.NoError(t, err)
	second, err := service.Run(context.Background(), 7, "monitor-key", "trace-1", input)
	require.NoError(t, err)
	require.Equal(t, first.Decision.ID, second.Decision.ID)
	require.Equal(t, 1, gateway.reads)
	var decisions, ledger int64
	require.NoError(t, db.Model(&PriceDecision{}).Count(&decisions).Error)
	require.NoError(t, db.Model(&ProfitLedgerEntry{}).Count(&ledger).Error)
	require.Equal(t, int64(1), decisions)
	require.Equal(t, int64(4), ledger)
	require.Equal(t, 0, gateway.updates, "an unset automation mode must never auto-apply")

	applied, err := service.Apply(context.Background(), 7, first.Decision.ID)
	require.NoError(t, err)
	require.Equal(t, OutcomeProposed, applied.Outcome)
	require.Equal(t, 1, gateway.updates)
}

func TestCreateRuleReturnsSameVersionForSameCommandKey(t *testing.T) {
	db, _, _ := monitoringFixture(t)
	service := &Service{DB: db}
	input := CreateRuleInput{Name: "Conservative", MinMarginBasisPoints: 1_500, TargetMarginBasisPoints: 2_000, MaxDeltaCents: 500, MaxDeltaBasisPoints: 1_000, PlatformFeeBasisPoints: 1_200}
	first, err := service.CreateRule(context.Background(), 7, "rule-command", input)
	require.NoError(t, err)
	second, err := service.CreateRule(context.Background(), 7, "rule-command", input)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
}
