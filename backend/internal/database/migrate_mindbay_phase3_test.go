package database

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/modules/monitoring"
	"github.com/trademind-ai/trademind/backend/internal/modules/publication"
	"gorm.io/gorm"
)

func TestMigrateMindBayPhase3CreatesMonitoringLedgerSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, migrateMindBayPhase3(db))
	for _, table := range []any{&publication.ListingSnapshot{}, &monitoring.MonitorRun{}, &monitoring.PriceRule{}, &monitoring.PriceDecision{}, &monitoring.ProfitLedgerEntry{}} {
		require.True(t, db.Migrator().HasTable(table))
	}
}
