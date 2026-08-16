package publication

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/modules/listingstudio"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func publicationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&listingstudio.ListingDraft{}, &listingstudio.ListingContentVersion{}, &CalendarSlot{}, &PublicationJob{}, &MarketplaceListing{}, &PublicationTransitionEvent{}))
	return db
}

func readyDraft(t *testing.T, db *gorm.DB, workspaceID int64, createdAt time.Time) (listingstudio.ListingDraft, listingstudio.ListingContentVersion) {
	t.Helper()
	draft := listingstudio.ListingDraft{WorkspaceID: workspaceID, SourceProductID: uuid.New(), State: listingstudio.StateReady, Category: "123", Currency: "EUR", ValidationErrors: datatypes.JSON([]byte("[]"))}
	require.NoError(t, db.Create(&draft).Error)
	version := listingstudio.ListingContentVersion{WorkspaceID: workspaceID, ListingDraftID: draft.ID, Version: 1, Title: "Fixture", Description: "Description", Specifics: datatypes.JSON([]byte("{}"))}
	require.NoError(t, db.Create(&version).Error)
	require.NoError(t, db.Model(&draft).Updates(map[string]any{"current_content_version_id": version.ID, "created_at": createdAt}).Error)
	draft.CurrentContentVersionID = &version.ID
	draft.CreatedAt = createdAt
	return draft, version
}

func TestBuildPreviewIsDeterministicAndBounded(t *testing.T) {
	now := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	rows := []Candidate{{ListingDraftID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), ContentVersionID: uuid.New(), Category: "A", CreatedAt: now.Add(-48 * time.Hour)}, {ListingDraftID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), ContentVersionID: uuid.New(), Category: "B", CreatedAt: now.Add(-48 * time.Hour)}, {ListingDraftID: uuid.New(), ContentVersionID: uuid.New(), Category: "A", CreatedAt: now.Add(-24 * time.Hour)}}
	out, err := BuildPreview(PreviewInput{StartAt: now.Add(time.Hour), Days: 1, MaxPerDay: 2, MinSpacingMinutes: 90}, rows, now)
	require.NoError(t, err)
	require.Len(t, out.Slots, 2)
	require.Equal(t, 1, out.Unplaced)
	require.Equal(t, "00000000-0000-0000-0000-000000000001", out.Slots[0].ListingDraftID.String())
	require.Equal(t, 90*time.Minute, out.Slots[1].ScheduledFor.Sub(out.Slots[0].ScheduledFor))
}

func TestPreviewDoesNotMutateAndApplyIsIdempotent(t *testing.T) {
	db := publicationTestDB(t)
	now := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	draft, version := readyDraft(t, db, 7, now.Add(-24*time.Hour))
	listingSvc := &listingstudio.Service{DB: db}
	svc := &Service{DB: db, Listings: listingSvc, Now: func() time.Time { return now }}
	preview, err := svc.Preview(context.Background(), 7, PreviewInput{StartAt: now.Add(time.Hour), Days: 1, MaxPerDay: 1, MinSpacingMinutes: 60})
	require.NoError(t, err)
	require.Len(t, preview.Slots, 1)
	var slotsBefore, jobsBefore int64
	require.NoError(t, db.Model(&CalendarSlot{}).Count(&slotsBefore).Error)
	require.NoError(t, db.Model(&PublicationJob{}).Count(&jobsBefore).Error)
	require.Zero(t, slotsBefore)
	require.Zero(t, jobsBefore)
	in := ApplyInput{ShopID: uuid.New(), Marketplace: "EBAY_DE", Slots: preview.Slots, PublishConfig: []byte(`{"merchantLocationKey":"loc"}`)}
	first, err := svc.Apply(context.Background(), 7, "same-command", in)
	require.NoError(t, err)
	second, err := svc.Apply(context.Background(), 7, "same-command", in)
	require.NoError(t, err)
	require.Equal(t, first.Slots[0].ID, second.Slots[0].ID)
	require.Equal(t, first.Jobs[0].ID, second.Jobs[0].ID)
	var slotCount, jobCount int64
	require.NoError(t, db.Model(&CalendarSlot{}).Count(&slotCount).Error)
	require.NoError(t, db.Model(&PublicationJob{}).Count(&jobCount).Error)
	require.EqualValues(t, 1, slotCount)
	require.EqualValues(t, 1, jobCount)
	var transitionCount int64
	require.NoError(t, db.Model(&PublicationTransitionEvent{}).Count(&transitionCount).Error)
	require.EqualValues(t, 1, transitionCount)
	require.Len(t, first.Jobs[0].PayloadHash, 64)
	var persisted listingstudio.ListingDraft
	require.NoError(t, db.First(&persisted, "id=?", draft.ID).Error)
	require.Equal(t, listingstudio.StateScheduled, persisted.State)
	require.Equal(t, version.ID, *persisted.CurrentContentVersionID)
}

func TestApplyRejectsSecondActiveSlotAtSameTime(t *testing.T) {
	db := publicationTestDB(t)
	now := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	firstDraft, firstVersion := readyDraft(t, db, 7, now.Add(-48*time.Hour))
	secondDraft, secondVersion := readyDraft(t, db, 7, now.Add(-24*time.Hour))
	svc := &Service{DB: db, Listings: &listingstudio.Service{DB: db}, Now: func() time.Time { return now }}
	scheduledFor := now.Add(2 * time.Hour)
	shopID := uuid.New()

	_, err := svc.Apply(context.Background(), 7, "first", ApplyInput{ShopID: shopID, Marketplace: "EBAY_DE", Slots: []PreviewSlot{{ListingDraftID: firstDraft.ID, ContentVersionID: firstVersion.ID, ScheduledFor: scheduledFor}}})
	require.NoError(t, err)
	_, err = svc.Apply(context.Background(), 7, "second", ApplyInput{ShopID: shopID, Marketplace: "EBAY_DE", Slots: []PreviewSlot{{ListingDraftID: secondDraft.ID, ContentVersionID: secondVersion.ID, ScheduledFor: scheduledFor}}})
	require.ErrorContains(t, err, "active publication slot")

	var persisted listingstudio.ListingDraft
	require.NoError(t, db.First(&persisted, "id=?", secondDraft.ID).Error)
	require.Equal(t, listingstudio.StateReady, persisted.State)
}

func TestListSlotsIncludesOperationalPublicationDetails(t *testing.T) {
	db := publicationTestDB(t)
	now := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	draft, version := readyDraft(t, db, 7, now.Add(-24*time.Hour))
	price := int64(1299)
	require.NoError(t, db.Model(&draft).Updates(map[string]any{"price_cents": price, "currency": "EUR"}).Error)
	svc := &Service{DB: db, Listings: &listingstudio.Service{DB: db}, Now: func() time.Time { return now }}
	result, err := svc.Apply(context.Background(), 7, "calendar-details", ApplyInput{ShopID: uuid.New(), Marketplace: "EBAY_DE", Slots: []PreviewSlot{{ListingDraftID: draft.ID, ContentVersionID: version.ID, ScheduledFor: now.Add(time.Hour)}}})
	require.NoError(t, err)
	job := result.Jobs[0]
	require.NoError(t, db.Model(&job).Updates(map[string]any{"status": JobFailed, "error_message": "sandbox validation failed"}).Error)
	require.NoError(t, db.Create(&MarketplaceListing{WorkspaceID: 7, ListingDraftID: draft.ID, PublicationJobID: job.ID, Marketplace: "ebay", ExternalListingID: "110011", ExternalURL: "https://sandbox.ebay.example/110011", SKU: "TM-110011", Status: "ACTIVE", PriceCents: price, Currency: "EUR"}).Error)

	rows, err := svc.ListSlots(context.Background(), 7, now, now.Add(2*time.Hour))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0].PriceCents)
	require.Equal(t, price, *rows[0].PriceCents)
	require.Equal(t, "EUR", rows[0].Currency)
	require.Equal(t, JobFailed, rows[0].JobStatus)
	require.Equal(t, "sandbox validation failed", rows[0].ErrorMessage)
	require.Equal(t, "110011", rows[0].ExternalListingID)
}
