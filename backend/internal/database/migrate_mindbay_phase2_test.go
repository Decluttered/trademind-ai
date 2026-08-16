package database

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/modules/publication"
	"gorm.io/gorm"
)

func TestMigrateMindBayPhase2IsRepeatableAndEnforcesActiveSlotUniqueness(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, migrateMindBayPhase2(db))
	require.NoError(t, migrateMindBayPhase2(db))
	for _, table := range []any{&publication.CalendarSlot{}, &publication.PublicationJob{}, &publication.MarketplaceListing{}, &publication.ListingSnapshot{}, &publication.PublicationTransitionEvent{}} {
		require.True(t, db.Migrator().HasTable(table))
	}

	when := time.Date(2026, 8, 16, 10, 0, 0, 0, time.UTC)
	first := publication.CalendarSlot{WorkspaceID: 7, ListingDraftID: uuid.New(), ContentVersionID: uuid.New(), ScheduledFor: when, SlotType: "EBAY_PUBLISH", Status: publication.SlotScheduled, IdempotencyKey: "first"}
	second := publication.CalendarSlot{WorkspaceID: 7, ListingDraftID: uuid.New(), ContentVersionID: uuid.New(), ScheduledFor: when, SlotType: "EBAY_PUBLISH", Status: publication.SlotHeld, IdempotencyKey: "second"}
	require.NoError(t, db.Create(&first).Error)
	require.Error(t, db.Create(&second).Error)
}
