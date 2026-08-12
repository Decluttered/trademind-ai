package database

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type phaseTableRename struct {
	legacy  string
	current string
}

var inventoryPhaseTableRenames = []phaseTableRename{
	{legacy: "p9_inventory_sync_runs", current: "inventory_sync_runs"},
	{legacy: "p9_inventory_snapshot_items", current: "inventory_snapshot_items"},
	{legacy: "p9_sku_bindings", current: "sku_bindings"},
	{legacy: "p9_sku_binding_calibrations", current: "sku_binding_calibrations"},
	{legacy: "p9_manual_binding_requests", current: "manual_binding_requests"},
	{legacy: "p9_manual_binding_decisions", current: "manual_binding_decisions"},
}

var productionPhaseTableRenames = []phaseTableRename{
	{legacy: "p10_platform_credentials", current: "platform_credentials"},
	{legacy: "p10_oauth_credentials", current: "platform_oauth_credentials"},
	{legacy: "p10_credential_bindings", current: "platform_credential_bindings"},
	{legacy: "p10_credential_versions", current: "platform_credential_versions"},
	{legacy: "p10_credential_lifecycle_events", current: "platform_credential_lifecycle_events"},
	{legacy: "p10_oauth_states", current: "platform_oauth_states"},
	{legacy: "p10_runtime_controls", current: "production_runtime_controls"},
	{legacy: "p10_scope_allowlists", current: "production_scope_allowlists"},
	{legacy: "p10_gray_policies", current: "production_rollout_policies"},
	{legacy: "p10_control_audit_events", current: "production_control_audit_events"},
}

var phaseObjectNameRenames = []phaseTableRename{
	{legacy: "p9_inventory_snapshot_items", current: "inventory_snapshot_items"},
	{legacy: "p9_inventory_sync_runs", current: "inventory_sync_runs"},
	{legacy: "p9_sku_binding_calibrations", current: "sku_binding_calibrations"},
	{legacy: "p9_manual_binding_requests", current: "manual_binding_requests"},
	{legacy: "p9_manual_binding_decisions", current: "manual_binding_decisions"},
	{legacy: "p9_inventory_snapshots", current: "inventory_snapshots"},
	{legacy: "p9_sku_calibrations", current: "sku_calibrations"},
	{legacy: "p9_sku_bindings", current: "sku_bindings"},
	{legacy: "p9_operation_logs", current: "inventory_operation_logs"},
	{legacy: "p10_credential_lifecycle_events", current: "platform_credential_lifecycle_events"},
	{legacy: "p10_platform_credentials", current: "platform_credentials"},
	{legacy: "p10_oauth_credentials", current: "platform_oauth_credentials"},
	{legacy: "p10_credential_bindings", current: "platform_credential_bindings"},
	{legacy: "p10_credential_versions", current: "platform_credential_versions"},
	{legacy: "p10_oauth_states", current: "platform_oauth_states"},
	{legacy: "p10_runtime_controls", current: "production_runtime_controls"},
	{legacy: "p10_scope_allowlists", current: "production_scope_allowlists"},
	{legacy: "p10_scope_allowlist", current: "production_scope_allowlist"},
	{legacy: "p10_gray_policies", current: "production_rollout_policies"},
	{legacy: "p10_control_audit_events", current: "production_control_audit_events"},
	{legacy: "p10_credentials", current: "platform_credentials"},
}

func migrateLegacyPhaseTableNames(db *gorm.DB, renames []phaseTableRename) error {
	if db == nil {
		return fmt.Errorf("legacy phase table migration: db is nil")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, rename := range renames {
			hasLegacy := tx.Migrator().HasTable(rename.legacy)
			hasCurrent := tx.Migrator().HasTable(rename.current)
			if hasLegacy && hasCurrent {
				return fmt.Errorf("legacy phase table migration conflict: both %s and %s exist", rename.legacy, rename.current)
			}
			if !hasLegacy {
				continue
			}
			if err := tx.Migrator().RenameTable(rename.legacy, rename.current); err != nil {
				return fmt.Errorf("rename table %s to %s: %w", rename.legacy, rename.current, err)
			}
		}

		if tx.Dialector.Name() != "postgres" {
			return nil
		}
		if containsLegacyTablePrefix(renames, "p9_") {
			if err := renamePostgresFunction(tx, "inventorysyncp9_reject_immutable_change", "inventorysync_reject_immutable_change"); err != nil {
				return err
			}
			if err := renamePostgresTableObjects(tx, "operation_logs"); err != nil {
				return err
			}
		}
		for _, rename := range renames {
			if !tx.Migrator().HasTable(rename.current) {
				continue
			}
			if err := renamePostgresTableObjects(tx, rename.current); err != nil {
				return err
			}
		}
		return nil
	})
}

func containsLegacyTablePrefix(renames []phaseTableRename, prefix string) bool {
	for _, rename := range renames {
		if strings.HasPrefix(rename.legacy, prefix) {
			return true
		}
	}
	return false
}

func renamePostgresTableObjects(db *gorm.DB, table string) error {
	var constraints []string
	if err := db.Raw(`SELECT conname FROM pg_constraint WHERE conrelid = to_regclass(?) ORDER BY conname`, table).Scan(&constraints).Error; err != nil {
		return fmt.Errorf("list constraints for %s: %w", table, err)
	}
	for _, legacy := range constraints {
		current := phaseFreeObjectName(legacy)
		if current == legacy {
			continue
		}
		stmt := fmt.Sprintf("ALTER TABLE %s RENAME CONSTRAINT %s TO %s", quotePostgresIdentifier(table), quotePostgresIdentifier(legacy), quotePostgresIdentifier(current))
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("rename constraint %s to %s: %w", legacy, current, err)
		}
	}

	var triggers []string
	if err := db.Raw(`SELECT tgname FROM pg_trigger WHERE tgrelid = to_regclass(?) AND NOT tgisinternal ORDER BY tgname`, table).Scan(&triggers).Error; err != nil {
		return fmt.Errorf("list triggers for %s: %w", table, err)
	}
	for _, legacy := range triggers {
		current := phaseFreeObjectName(legacy)
		if current == legacy {
			continue
		}
		stmt := fmt.Sprintf("ALTER TRIGGER %s ON %s RENAME TO %s", quotePostgresIdentifier(legacy), quotePostgresIdentifier(table), quotePostgresIdentifier(current))
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("rename trigger %s to %s: %w", legacy, current, err)
		}
	}

	var indexes []string
	if err := db.Raw(`SELECT indexname FROM pg_indexes WHERE schemaname = current_schema() AND tablename = ? ORDER BY indexname`, table).Scan(&indexes).Error; err != nil {
		return fmt.Errorf("list indexes for %s: %w", table, err)
	}
	for _, legacy := range indexes {
		current := phaseFreeObjectName(legacy)
		if current == legacy {
			continue
		}
		stmt := fmt.Sprintf("ALTER INDEX %s RENAME TO %s", quotePostgresIdentifier(legacy), quotePostgresIdentifier(current))
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("rename index %s to %s: %w", legacy, current, err)
		}
	}
	return nil
}

func renamePostgresFunction(db *gorm.DB, legacy, current string) error {
	var legacyExists, currentExists bool
	if err := db.Raw(`SELECT to_regprocedure(?) IS NOT NULL`, legacy+"()").Scan(&legacyExists).Error; err != nil {
		return fmt.Errorf("inspect function %s: %w", legacy, err)
	}
	if err := db.Raw(`SELECT to_regprocedure(?) IS NOT NULL`, current+"()").Scan(&currentExists).Error; err != nil {
		return fmt.Errorf("inspect function %s: %w", current, err)
	}
	if legacyExists && currentExists {
		return fmt.Errorf("legacy phase function migration conflict: both %s and %s exist", legacy, current)
	}
	if !legacyExists {
		return nil
	}
	stmt := fmt.Sprintf("ALTER FUNCTION %s() RENAME TO %s", quotePostgresIdentifier(legacy), quotePostgresIdentifier(current))
	if err := db.Exec(stmt).Error; err != nil {
		return fmt.Errorf("rename function %s to %s: %w", legacy, current, err)
	}
	return nil
}

func phaseFreeObjectName(name string) string {
	for _, rename := range phaseObjectNameRenames {
		name = strings.ReplaceAll(name, rename.legacy, rename.current)
	}
	return name
}

func quotePostgresIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
