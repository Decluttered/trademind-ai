package database

import (
	"fmt"

	"github.com/trademind-ai/trademind/backend/internal/modules/monitoring"
	"github.com/trademind-ai/trademind/backend/internal/modules/publication"
	"gorm.io/gorm"
)

func migrateMindBayPhase3(db *gorm.DB) error {
	if err := db.AutoMigrate(&publication.ListingSnapshot{}, &monitoring.MonitorRun{}, &monitoring.PriceRule{}, &monitoring.PriceDecision{}, &monitoring.ProfitLedgerEntry{}); err != nil {
		return fmt.Errorf("mindbay phase 3 AutoMigrate: %w", err)
	}
	return nil
}
