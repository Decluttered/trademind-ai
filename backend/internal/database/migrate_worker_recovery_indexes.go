package database

import (
	"fmt"

	"gorm.io/gorm"
)

// migrateWorkerRecoveryIndexes adds lease indexes for collect/image/customer sync and webhook payload hashes.
func migrateWorkerRecoveryIndexes(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("migrate worker recovery indexes: db is nil")
	}
	return createWorkerRecoveryIndexes(db)
}

func createWorkerRecoveryIndexes(db *gorm.DB) error {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return nil
	}
	stmts := []string{
		`CREATE INDEX IF NOT EXISTS ix_collect_execution_id ON collect_tasks (execution_id)`,
		`CREATE INDEX IF NOT EXISTS ix_collect_heartbeat_at ON collect_tasks (heartbeat_at)`,
		`CREATE INDEX IF NOT EXISTS ix_image_execution_id ON image_tasks (execution_id)`,
		`CREATE INDEX IF NOT EXISTS ix_image_heartbeat_at ON image_tasks (heartbeat_at)`,
		`CREATE INDEX IF NOT EXISTS ix_customer_sync_execution_id ON customer_message_sync_tasks (execution_id)`,
		`CREATE INDEX IF NOT EXISTS ix_customer_sync_heartbeat_at ON customer_message_sync_tasks (heartbeat_at)`,
		`CREATE INDEX IF NOT EXISTS ix_webhook_payload_hash ON webhook_events (platform, payload_hash)`,
	}
	for _, sql := range stmts {
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("worker recovery index: %w", err)
		}
	}
	return nil
}
