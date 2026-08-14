package database

import (
	"fmt"

	"github.com/trademind-ai/trademind/backend/internal/modules/order"
	"gorm.io/gorm"
)

// migrateDouyinOrderRevision adds non-destructive order revision schema changes.
func migrateDouyinOrderRevision(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("migrate Douyin order revision: db is nil")
	}
	if err := db.AutoMigrate(&order.Order{}); err != nil {
		return fmt.Errorf("Douyin order revision AutoMigrate: %w", err)
	}
	if err := migrateDouyinOrderRevisionIndexes(db); err != nil {
		return err
	}
	return nil
}

func migrateDouyinOrderRevisionIndexes(db *gorm.DB) error {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return nil
	}
	stmts := []string{
		`CREATE INDEX IF NOT EXISTS ix_orders_platform_updated_at ON orders (platform_updated_at)`,
		`CREATE INDEX IF NOT EXISTS ix_orders_platform_revision ON orders (platform_revision)`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("Douyin order revision index: %w", err)
		}
	}
	return nil
}
