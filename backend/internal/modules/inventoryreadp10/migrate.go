package inventoryreadp10

import "gorm.io/gorm"

// Migrate extends the existing P9 run contract with the P10 read-only provider mode.
// P9 source remains frozen; P10 owns this additive database compatibility change.
func Migrate(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" || !db.Migrator().HasTable("inventory_sync_runs") {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`ALTER TABLE inventory_sync_runs DROP CONSTRAINT IF EXISTS chk_inventory_sync_runs_provider_mode`).Error; err != nil {
			return err
		}
		return tx.Exec(`ALTER TABLE inventory_sync_runs ADD CONSTRAINT chk_inventory_sync_runs_provider_mode CHECK (provider_mode IN ('mock','sandbox','local_draft_only','real_readonly'))`).Error
	})
}
