package database

import (
	"github.com/trademind-ai/trademind/backend/internal/modules/credentialp10"
	"github.com/trademind-ai/trademind/backend/internal/modules/inventoryreadp10"
	"github.com/trademind-ai/trademind/backend/internal/modules/productioncontrolp10"
	"gorm.io/gorm"
)

// AutoMigrateWithP10 preserves the frozen P9 migration contract and then applies P10-owned schema changes.
func AutoMigrateWithP10(db *gorm.DB) error {
	if err := AutoMigrate(db); err != nil {
		return err
	}
	if err := migrateLegacyPhaseTableNames(db, productionPhaseTableRenames); err != nil {
		return err
	}
	if err := credentialp10.Migrate(db); err != nil {
		return err
	}
	if err := productioncontrolp10.Migrate(db); err != nil {
		return err
	}
	return inventoryreadp10.Migrate(db)
}
