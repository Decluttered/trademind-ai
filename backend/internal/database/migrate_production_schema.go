package database

import (
	"github.com/trademind-ai/trademind/backend/internal/modules/inventoryread"
	"github.com/trademind-ai/trademind/backend/internal/modules/platformcredential"
	"github.com/trademind-ai/trademind/backend/internal/modules/productioncontrol"
	"gorm.io/gorm"
)

// AutoMigrateProductionSchema applies core migrations, then production capability schema changes.
func AutoMigrateProductionSchema(db *gorm.DB) error {
	if err := AutoMigrate(db); err != nil {
		return err
	}
	if err := migrateLegacyTableNames(db, productionLegacyTableRenames); err != nil {
		return err
	}
	if err := platformcredential.Migrate(db); err != nil {
		return err
	}
	if err := productioncontrol.Migrate(db); err != nil {
		return err
	}
	return inventoryread.Migrate(db)
}
