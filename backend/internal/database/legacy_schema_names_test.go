package database

import (
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMigrateLegacyTableNamesRenamesTablesAndPreservesRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	allRenames := append(append([]legacySchemaRename{}, inventoryLegacyTableRenames...), productionLegacyTableRenames...)
	for i, rename := range allRenames {
		create := fmt.Sprintf(`CREATE TABLE %s (id INTEGER PRIMARY KEY, marker TEXT NOT NULL)`, quotePostgresIdentifier(rename.legacy))
		insert := fmt.Sprintf(`INSERT INTO %s (id, marker) VALUES (?, ?)`, quotePostgresIdentifier(rename.legacy))
		require.NoError(t, db.Exec(create).Error)
		require.NoError(t, db.Exec(insert, i+1, rename.current).Error)
	}

	require.NoError(t, migrateLegacyTableNames(db, inventoryLegacyTableRenames))
	require.NoError(t, migrateLegacyTableNames(db, productionLegacyTableRenames))
	for i, rename := range allRenames {
		require.False(t, db.Migrator().HasTable(rename.legacy))
		require.True(t, db.Migrator().HasTable(rename.current))

		var marker string
		query := fmt.Sprintf(`SELECT marker FROM %s WHERE id = ?`, quotePostgresIdentifier(rename.current))
		require.NoError(t, db.Raw(query, i+1).Scan(&marker).Error)
		require.Equal(t, rename.current, marker)
	}
}

func TestMigrateLegacyTableNamesRejectsSplitSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE p10_runtime_controls (id TEXT PRIMARY KEY)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE production_runtime_controls (id TEXT PRIMARY KEY)`).Error)

	err = migrateLegacyTableNames(db, productionLegacyTableRenames)
	require.EqualError(t, err, "legacy table migration conflict: both p10_runtime_controls and production_runtime_controls exist")
	require.True(t, db.Migrator().HasTable("p10_runtime_controls"))
	require.True(t, db.Migrator().HasTable("production_runtime_controls"))
}

func TestMigrateLegacyTableNamesRenamesImageTaskItems(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE ai_image_task_items (id INTEGER PRIMARY KEY, marker TEXT NOT NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ai_image_task_items (id, marker) VALUES (1, 'preserved')`).Error)

	require.NoError(t, migrateLegacyTableNames(db, imageTaskTableRenames))
	require.False(t, db.Migrator().HasTable("ai_image_task_items"))
	require.True(t, db.Migrator().HasTable("image_task_items"))

	var marker string
	require.NoError(t, db.Raw(`SELECT marker FROM image_task_items WHERE id = 1`).Scan(&marker).Error)
	require.Equal(t, "preserved", marker)
}

func TestMigrateLegacyTableNamesRejectsImageTaskSplitSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE ai_image_task_items (id INTEGER PRIMARY KEY)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE image_task_items (id INTEGER PRIMARY KEY)`).Error)

	err = migrateLegacyTableNames(db, imageTaskTableRenames)
	require.EqualError(t, err, "legacy table migration conflict: both ai_image_task_items and image_task_items exist")
	require.True(t, db.Migrator().HasTable("ai_image_task_items"))
	require.True(t, db.Migrator().HasTable("image_task_items"))
}

func TestStableObjectName(t *testing.T) {
	tests := map[string]string{
		"idx_p9_inventory_sync_runs_tenant_shop_status":     "idx_inventory_sync_runs_tenant_shop_status",
		"ux_p9_inventory_snapshots_tenant_run_external_sku": "ux_inventory_snapshots_tenant_run_external_sku",
		"trg_p9_operation_logs_no_delete":                   "trg_inventory_operation_logs_no_delete",
		"ux_p10_credentials_tenant_platform_shop":           "ux_platform_credentials_tenant_platform_shop",
		"idx_p10_scope_allowlists_enabled":                  "idx_production_scope_allowlists_enabled",
		"p10_gray_policies_pkey":                            "production_rollout_policies_pkey",
		"idx_p10_credential_lifecycle_events_credential_id": "idx_platform_credential_lifecycle_events_credential_id",
		"idx_products_p7_tenant_created_id":                 "idx_products_tenant_created_id",
		"idx_orders_p7_tenant_shop_created_id":              "idx_orders_tenant_shop_created_id",
		"idx_inventory_sync_tasks_publication_sk_uid":       "idx_inventory_sync_tasks_publication_sku_id",
		"idx_files_p7_tenant_security_created":              "idx_files_tenant_security_created",
	}
	for legacy, current := range tests {
		require.Equal(t, current, stableObjectName(legacy), legacy)
	}
	for _, rename := range append(append([]legacySchemaRename{}, inventoryLegacyTableRenames...), productionLegacyTableRenames...) {
		require.Equal(t, rename.current+"_pkey", stableObjectName(rename.legacy+"_pkey"), rename.legacy)
	}
	for _, rename := range legacyObjectNameRenames {
		require.Equal(t, rename.current, stableObjectName(rename.legacy), rename.legacy)
	}
	require.Equal(t, "idx_image_task_items_task_id", stableObjectName("idx_ai_image_task_items_task_id"))
}

func TestLegacyInventorySyncFunctionNameRemainsCompatible(t *testing.T) {
	require.Equal(t, "inventorysyncp9_reject_immutable_change", legacyInventorySyncImmutableFunction)
	require.Equal(t, "inventorysync_reject_immutable_change", currentInventorySyncImmutableFunction)
}

func TestNormalizePostgresIndexDefinitionRemovesOnlyIndexName(t *testing.T) {
	definition := `CREATE UNIQUE INDEX "idx legacy" ON public.inventory_sync_tasks USING btree (lower((platform)::text), publication_sku_id) WHERE (status = 'failed'::text)`

	normalized, err := normalizePostgresIndexDefinition(definition)
	require.NoError(t, err)
	require.Equal(t,
		`unique|ON public.inventory_sync_tasks USING btree (lower((platform)::text), publication_sku_id) WHERE (status = 'failed'::text)`,
		normalized,
	)
}

func TestEquivalentPostgresIndexDefinitions(t *testing.T) {
	legacy := `CREATE INDEX idx_inventory_sync_tasks_publication_sk_uid ON public.inventory_sync_tasks USING btree (publication_sku_id)`
	current := `CREATE INDEX idx_inventory_sync_tasks_publication_sku_id ON public.inventory_sync_tasks USING btree (publication_sku_id)`

	equivalent, err := equivalentPostgresIndexDefinitions(legacy, current)
	require.NoError(t, err)
	require.True(t, equivalent)

	uniqueCurrent := `CREATE UNIQUE INDEX idx_inventory_sync_tasks_publication_sku_id ON public.inventory_sync_tasks USING btree (publication_sku_id)`
	equivalent, err = equivalentPostgresIndexDefinitions(legacy, uniqueCurrent)
	require.NoError(t, err)
	require.False(t, equivalent)

	partialCurrent := `CREATE INDEX idx_inventory_sync_tasks_publication_sku_id ON public.inventory_sync_tasks USING btree (publication_sku_id) WHERE (status = 'failed'::text)`
	equivalent, err = equivalentPostgresIndexDefinitions(legacy, partialCurrent)
	require.NoError(t, err)
	require.False(t, equivalent)
}

func TestNormalizePostgresIndexDefinitionRejectsMalformedInput(t *testing.T) {
	_, err := normalizePostgresIndexDefinition(`DROP INDEX idx_inventory_sync_tasks_publication_sk_uid`)
	require.EqualError(t, err, "unsupported PostgreSQL index definition")

	_, err = normalizePostgresIndexDefinition(`CREATE INDEX "unterminated ON inventory_sync_tasks (publication_sku_id)`)
	require.EqualError(t, err, "PostgreSQL index definition has an unterminated quoted index name")
}
