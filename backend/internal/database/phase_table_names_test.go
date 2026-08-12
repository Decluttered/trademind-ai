package database

import (
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMigrateLegacyPhaseTableNamesRenamesTablesAndPreservesRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	allRenames := append(append([]phaseTableRename{}, inventoryPhaseTableRenames...), productionPhaseTableRenames...)
	for i, rename := range allRenames {
		create := fmt.Sprintf(`CREATE TABLE %s (id INTEGER PRIMARY KEY, marker TEXT NOT NULL)`, quotePostgresIdentifier(rename.legacy))
		insert := fmt.Sprintf(`INSERT INTO %s (id, marker) VALUES (?, ?)`, quotePostgresIdentifier(rename.legacy))
		require.NoError(t, db.Exec(create).Error)
		require.NoError(t, db.Exec(insert, i+1, rename.current).Error)
	}

	require.NoError(t, migrateLegacyPhaseTableNames(db, inventoryPhaseTableRenames))
	require.NoError(t, migrateLegacyPhaseTableNames(db, productionPhaseTableRenames))
	for i, rename := range allRenames {
		require.False(t, db.Migrator().HasTable(rename.legacy))
		require.True(t, db.Migrator().HasTable(rename.current))

		var marker string
		query := fmt.Sprintf(`SELECT marker FROM %s WHERE id = ?`, quotePostgresIdentifier(rename.current))
		require.NoError(t, db.Raw(query, i+1).Scan(&marker).Error)
		require.Equal(t, rename.current, marker)
	}
}

func TestMigrateLegacyPhaseTableNamesRejectsSplitSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE p10_runtime_controls (id TEXT PRIMARY KEY)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE production_runtime_controls (id TEXT PRIMARY KEY)`).Error)

	err = migrateLegacyPhaseTableNames(db, productionPhaseTableRenames)
	require.EqualError(t, err, "legacy phase table migration conflict: both p10_runtime_controls and production_runtime_controls exist")
	require.True(t, db.Migrator().HasTable("p10_runtime_controls"))
	require.True(t, db.Migrator().HasTable("production_runtime_controls"))
}

func TestPhaseFreeObjectName(t *testing.T) {
	tests := map[string]string{
		"idx_p9_inventory_sync_runs_tenant_shop_status":     "idx_inventory_sync_runs_tenant_shop_status",
		"ux_p9_inventory_snapshots_tenant_run_external_sku": "ux_inventory_snapshots_tenant_run_external_sku",
		"trg_p9_operation_logs_no_delete":                   "trg_inventory_operation_logs_no_delete",
		"ux_p10_credentials_tenant_platform_shop":           "ux_platform_credentials_tenant_platform_shop",
		"idx_p10_scope_allowlists_enabled":                  "idx_production_scope_allowlists_enabled",
		"p10_gray_policies_pkey":                            "production_rollout_policies_pkey",
		"idx_p10_credential_lifecycle_events_credential_id": "idx_platform_credential_lifecycle_events_credential_id",
	}
	for legacy, current := range tests {
		require.Equal(t, current, phaseFreeObjectName(legacy), legacy)
	}
	for _, rename := range append(append([]phaseTableRename{}, inventoryPhaseTableRenames...), productionPhaseTableRenames...) {
		require.Equal(t, rename.current+"_pkey", phaseFreeObjectName(rename.legacy+"_pkey"), rename.legacy)
	}
}
