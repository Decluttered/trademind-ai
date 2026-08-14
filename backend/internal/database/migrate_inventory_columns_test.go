package database

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/modules/inventory"
	"gorm.io/gorm"
)

func TestMigrateLegacyInventoryPublicationSKUColumnPreservesRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
CREATE TABLE inventory_sync_tasks (
  id TEXT PRIMARY KEY,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  tenant_id INTEGER NOT NULL DEFAULT 0,
  product_id TEXT NOT NULL,
  publication_sk_uid TEXT,
  shop_id TEXT NOT NULL,
  platform TEXT NOT NULL,
  task_type TEXT NOT NULL,
  status TEXT NOT NULL,
  mode TEXT NOT NULL,
  target_stock INTEGER NOT NULL
)`).Error)

	taskID := uuid.New()
	publicationSKUID := uuid.New()
	now := time.Now().UTC()
	require.NoError(t, db.Exec(`
INSERT INTO inventory_sync_tasks (
  id, created_at, updated_at, tenant_id, product_id, publication_sk_uid,
  shop_id, platform, task_type, status, mode, target_stock
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		taskID, now, now, 7, uuid.New(), publicationSKUID,
		uuid.New(), "douyin_shop", inventory.TaskTypeInventorySync, inventory.StatusFailed, inventory.ModeManual, 3,
	).Error)

	require.NoError(t, migrateLegacyInventorySKUColumns(db))
	require.NoError(t, migrateLegacyInventorySKUColumns(db))
	require.True(t, db.Migrator().HasColumn(&inventory.InventorySyncTask{}, "publication_sku_id"))
	require.False(t, db.Migrator().HasColumn(&inventory.InventorySyncTask{}, "publication_sk_uid"))
	require.True(t, db.Migrator().HasIndex(&inventory.InventorySyncTask{}, "idx_inventory_sync_tasks_publication_sku_id"))
	require.False(t, db.Migrator().HasIndex(&inventory.InventorySyncTask{}, "idx_inventory_sync_tasks_publication_sk_uid"))

	var migratedPublicationSKUIDRaw string
	require.NoError(t, db.Raw(
		`SELECT publication_sku_id FROM inventory_sync_tasks WHERE id = ?`, taskID,
	).Scan(&migratedPublicationSKUIDRaw).Error)
	migratedPublicationSKUID, err := uuid.Parse(migratedPublicationSKUIDRaw)
	require.NoError(t, err)
	require.Equal(t, publicationSKUID, migratedPublicationSKUID)
}
