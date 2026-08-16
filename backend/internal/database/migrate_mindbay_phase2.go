package database

import (
	"fmt"

	"github.com/trademind-ai/trademind/backend/internal/modules/publication"
	"gorm.io/gorm"
)

func migrateMindBayPhase2(db *gorm.DB) error {
	if err := db.AutoMigrate(&publication.CalendarSlot{}, &publication.PublicationJob{}, &publication.MarketplaceListing{}, &publication.ListingSnapshot{}, &publication.PublicationTransitionEvent{}); err != nil {
		return fmt.Errorf("mindbay phase 2 AutoMigrate: %w", err)
	}
	// GORM cannot express a cross-dialect partial unique index. Both PostgreSQL
	// and SQLite accept this predicate; MySQL relies on the transaction guard.
	if db.Dialector.Name() == "postgres" || db.Dialector.Name() == "sqlite" {
		if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS ux_calendar_active_slot ON calendar_slot (workspace_id, scheduled_for, slot_type) WHERE deleted_at IS NULL AND status IN ('SCHEDULED','HELD','PUBLISHING')").Error; err != nil {
			return fmt.Errorf("mindbay phase 2 active slot index: %w", err)
		}
	}
	return nil
}
