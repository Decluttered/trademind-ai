package database

import (
	"fmt"

	"gorm.io/gorm"
)

func migrateCustomerAutoReplyReliability(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("customer auto reply reliability migration: db is nil")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if tx.Dialector.Name() == "postgres" {
			var conflicts int64
			conflictSQL := `
SELECT COUNT(*) FROM (
  SELECT conversation_id, external_message_id
  FROM customer_messages
  WHERE external_message_id IS NOT NULL AND BTRIM(external_message_id) <> ''
  GROUP BY conversation_id, external_message_id
  HAVING COUNT(DISTINCT CONCAT_WS(E'\x1f', role, content, language, message_type, source)) > 1
) conflicts`
			if err := tx.Raw(conflictSQL).Scan(&conflicts).Error; err != nil {
				return fmt.Errorf("inspect duplicate customer messages: %w", err)
			}
			if conflicts > 0 {
				return fmt.Errorf("customer message deduplication requires manual review: %d conflicting keys", conflicts)
			}
			var referencedDuplicates int64
			referencedSQL := `
WITH ranked AS (
  SELECT id, ROW_NUMBER() OVER (PARTITION BY conversation_id, external_message_id ORDER BY created_at, id) AS row_number
  FROM customer_messages
  WHERE external_message_id IS NOT NULL AND BTRIM(external_message_id) <> ''
)
SELECT COUNT(*)
FROM ranked
WHERE row_number > 1
	  AND (
	    EXISTS (SELECT 1 FROM customer_reply_suggestions suggestion WHERE suggestion.message_id = ranked.id)
	    OR EXISTS (SELECT 1 FROM customer_auto_reply_runs run WHERE run.message_id = ranked.id)
	    OR EXISTS (SELECT 1 FROM customer_auto_reply_runs run WHERE run.sent_message_id = ranked.id)
	  )`
			if err := tx.Raw(referencedSQL).Scan(&referencedDuplicates).Error; err != nil {
				return fmt.Errorf("inspect referenced duplicate customer messages: %w", err)
			}
			if referencedDuplicates > 0 {
				return fmt.Errorf("customer message deduplication requires manual review: %d duplicate rows are referenced", referencedDuplicates)
			}
			deleteSQL := `
DELETE FROM customer_messages duplicate
USING customer_messages keeper
WHERE duplicate.conversation_id = keeper.conversation_id
  AND duplicate.external_message_id = keeper.external_message_id
  AND duplicate.external_message_id IS NOT NULL
  AND BTRIM(duplicate.external_message_id) <> ''
  AND (duplicate.created_at, duplicate.id) > (keeper.created_at, keeper.id)`
			if err := tx.Exec(deleteSQL).Error; err != nil {
				return fmt.Errorf("deduplicate customer messages: %w", err)
			}
		}

		statements := []string{
			`CREATE UNIQUE INDEX IF NOT EXISTS ux_customer_messages_conversation_external ON customer_messages (conversation_id, external_message_id) WHERE external_message_id IS NOT NULL AND external_message_id <> ''`,
			`CREATE UNIQUE INDEX IF NOT EXISTS ux_customer_conversations_shop_platform_external ON customer_conversations (shop_id, platform, external_conversation_id) WHERE shop_id IS NOT NULL AND external_conversation_id IS NOT NULL AND external_conversation_id <> ''`,
			`CREATE UNIQUE INDEX IF NOT EXISTS ux_customer_sync_shop_inflight ON customer_message_sync_tasks (shop_id) WHERE status IN ('pending', 'running')`,
			`CREATE INDEX IF NOT EXISTS ix_customer_sync_shop_created ON customer_message_sync_tasks (shop_id, created_at DESC)`,
			`CREATE INDEX IF NOT EXISTS ix_customer_auto_reply_runs_shop_created ON customer_auto_reply_runs (shop_id, created_at DESC)`,
			`CREATE INDEX IF NOT EXISTS ix_customer_auto_reply_runs_shop_status_finished ON customer_auto_reply_runs (shop_id, status, finished_at)`,
			`CREATE INDEX IF NOT EXISTS ix_customer_messages_conversation_role_created ON customer_messages (conversation_id, role, created_at DESC, id DESC)`,
			`CREATE INDEX IF NOT EXISTS ix_customer_auto_reply_policies_due ON customer_auto_reply_policies (enabled, next_poll_at)`,
		}
		for _, statement := range statements {
			if err := tx.Exec(statement).Error; err != nil {
				return fmt.Errorf("customer auto reply reliability index: %w", err)
			}
		}
		return nil
	})
}
