package database

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type legacySchemaRename struct {
	legacy  string
	current string
}

const (
	legacyInventorySyncImmutableFunction  = "inventorysyncp9_reject_immutable_change"
	currentInventorySyncImmutableFunction = "inventorysync_reject_immutable_change"
)

var inventoryLegacyTableRenames = []legacySchemaRename{
	{legacy: "p9_inventory_sync_runs", current: "inventory_sync_runs"},
	{legacy: "p9_inventory_snapshot_items", current: "inventory_snapshot_items"},
	{legacy: "p9_sku_bindings", current: "sku_bindings"},
	{legacy: "p9_sku_binding_calibrations", current: "sku_binding_calibrations"},
	{legacy: "p9_manual_binding_requests", current: "manual_binding_requests"},
	{legacy: "p9_manual_binding_decisions", current: "manual_binding_decisions"},
}

var productionLegacyTableRenames = []legacySchemaRename{
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

// imageTaskTableRenames keeps the image task schema on the same domain naming
// convention as its parent image_tasks table. The old name is retained only
// as a one-time, data-preserving upgrade path.
var imageTaskTableRenames = []legacySchemaRename{
	{legacy: "ai_image_task_items", current: "image_task_items"},
}

var legacyObjectRenameTables = []string{
	"products",
	"orders",
	"inventory_sync_tasks",
	"collect_tasks",
	"webhook_events",
	"operation_logs",
	"files",
}

var legacyObjectNameRenames = []legacySchemaRename{
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
	{legacy: "idx_products_p7_tenant_created_id", current: "idx_products_tenant_created_id"},
	{legacy: "idx_orders_p7_tenant_created_id", current: "idx_orders_tenant_created_id"},
	{legacy: "idx_orders_p7_tenant_shop_created_id", current: "idx_orders_tenant_shop_created_id"},
	{legacy: "publication_sk_uid", current: "publication_sku_id"},
	{legacy: "idx_inventory_sync_tasks_p7_tenant_status_updated", current: "idx_inventory_sync_tasks_tenant_status_updated"},
	{legacy: "idx_collect_tasks_p7_tenant_updated_id", current: "idx_collect_tasks_tenant_updated_id"},
	{legacy: "idx_webhook_events_p7_tenant_status_created", current: "idx_webhook_events_tenant_status_created"},
	{legacy: "idx_operation_logs_p7_tenant_created_id", current: "idx_operation_logs_tenant_created_id"},
	{legacy: "idx_operation_logs_p7_chain_partition_created_id", current: "idx_operation_logs_chain_partition_created_id"},
	{legacy: "idx_files_p7_tenant_security_created", current: "idx_files_tenant_security_created"},
}

func migrateLegacyTableNames(db *gorm.DB, renames []legacySchemaRename) error {
	if db == nil {
		return fmt.Errorf("legacy table migration: db is nil")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, rename := range renames {
			hasLegacy := tx.Migrator().HasTable(rename.legacy)
			hasCurrent := tx.Migrator().HasTable(rename.current)
			if hasLegacy && hasCurrent {
				return fmt.Errorf("legacy table migration conflict: both %s and %s exist", rename.legacy, rename.current)
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
			if err := renamePostgresFunction(tx, legacyInventorySyncImmutableFunction, currentInventorySyncImmutableFunction); err != nil {
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

func migrateLegacyDatabaseObjectNames(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("legacy database object migration: db is nil")
	}
	if db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, table := range legacyObjectRenameTables {
			if !tx.Migrator().HasTable(table) {
				continue
			}
			if err := renamePostgresTableObjects(tx, table); err != nil {
				return err
			}
		}
		return nil
	})
}

func containsLegacyTablePrefix(renames []legacySchemaRename, prefix string) bool {
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
		current := stableObjectName(legacy)
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
		current := stableObjectName(legacy)
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
	existingIndexes := make(map[string]struct{}, len(indexes))
	for _, index := range indexes {
		existingIndexes[index] = struct{}{}
	}
	for _, legacy := range indexes {
		if _, exists := existingIndexes[legacy]; !exists {
			continue
		}
		current := stableObjectName(legacy)
		if current == legacy {
			continue
		}
		if _, currentExists := existingIndexes[current]; currentExists {
			legacyDefinition, err := postgresIndexDefinition(db, legacy)
			if err != nil {
				return err
			}
			currentDefinition, err := postgresIndexDefinition(db, current)
			if err != nil {
				return err
			}
			equivalent, err := equivalentPostgresIndexDefinitions(legacyDefinition, currentDefinition)
			if err != nil {
				return fmt.Errorf("compare index definitions %s and %s: %w", legacy, current, err)
			}
			if !equivalent {
				return fmt.Errorf("legacy index migration conflict: both %s and %s exist with different definitions", legacy, current)
			}
			stmt := fmt.Sprintf("DROP INDEX %s", quotePostgresIdentifier(legacy))
			if err := db.Exec(stmt).Error; err != nil {
				return fmt.Errorf("drop equivalent legacy index %s while keeping %s: %w", legacy, current, err)
			}
			delete(existingIndexes, legacy)
			continue
		}
		stmt := fmt.Sprintf("ALTER INDEX %s RENAME TO %s", quotePostgresIdentifier(legacy), quotePostgresIdentifier(current))
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("rename index %s to %s: %w", legacy, current, err)
		}
		delete(existingIndexes, legacy)
		existingIndexes[current] = struct{}{}
	}
	return nil
}

func postgresIndexDefinition(db *gorm.DB, index string) (string, error) {
	var row struct {
		Definition string
	}
	result := db.Raw(`
SELECT pg_get_indexdef(indexes.indexrelid) AS definition
FROM pg_index AS indexes
JOIN pg_class AS index_class ON index_class.oid = indexes.indexrelid
JOIN pg_namespace AS namespace ON namespace.oid = index_class.relnamespace
WHERE namespace.nspname = current_schema() AND index_class.relname = ?`, index).Scan(&row)
	if result.Error != nil {
		return "", fmt.Errorf("read index definition for %s: %w", index, result.Error)
	}
	if result.RowsAffected != 1 || strings.TrimSpace(row.Definition) == "" {
		return "", fmt.Errorf("read index definition for %s: expected one index in current schema", index)
	}
	return row.Definition, nil
}

func equivalentPostgresIndexDefinitions(left, right string) (bool, error) {
	normalizedLeft, err := normalizePostgresIndexDefinition(left)
	if err != nil {
		return false, err
	}
	normalizedRight, err := normalizePostgresIndexDefinition(right)
	if err != nil {
		return false, err
	}
	return normalizedLeft == normalizedRight, nil
}

// normalizePostgresIndexDefinition removes only the index name emitted by
// pg_get_indexdef. Uniqueness, table, method, expressions and predicates remain.
func normalizePostgresIndexDefinition(definition string) (string, error) {
	definition = strings.TrimSpace(definition)
	prefix := ""
	kind := ""
	switch {
	case strings.HasPrefix(definition, "CREATE UNIQUE INDEX "):
		prefix = "CREATE UNIQUE INDEX"
		kind = "unique"
	case strings.HasPrefix(definition, "CREATE INDEX "):
		prefix = "CREATE INDEX"
		kind = "nonunique"
	default:
		return "", fmt.Errorf("unsupported PostgreSQL index definition")
	}

	remainder := strings.TrimSpace(strings.TrimPrefix(definition, prefix))
	nameEnd, err := postgresIndexNameEnd(remainder)
	if err != nil {
		return "", err
	}
	remainder = strings.TrimSpace(remainder[nameEnd:])
	if !strings.HasPrefix(remainder, "ON ") {
		return "", fmt.Errorf("unsupported PostgreSQL index definition after index name")
	}
	return kind + "|" + remainder, nil
}

func postgresIndexNameEnd(value string) (int, error) {
	inQuotes := false
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case '"':
			if inQuotes && i+1 < len(value) && value[i+1] == '"' {
				i++
				continue
			}
			inQuotes = !inQuotes
		case ' ', '\t', '\r', '\n':
			if !inQuotes {
				if i == 0 {
					return 0, fmt.Errorf("PostgreSQL index definition has an empty index name")
				}
				return i, nil
			}
		}
	}
	if inQuotes {
		return 0, fmt.Errorf("PostgreSQL index definition has an unterminated quoted index name")
	}
	return 0, fmt.Errorf("PostgreSQL index definition is missing the indexed table")
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
		return fmt.Errorf("legacy function migration conflict: both %s and %s exist", legacy, current)
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

func stableObjectName(name string) string {
	for _, rename := range legacyObjectNameRenames {
		name = strings.ReplaceAll(name, rename.legacy, rename.current)
	}
	for _, rename := range imageTaskTableRenames {
		name = strings.ReplaceAll(name, rename.legacy, rename.current)
	}
	return name
}

func quotePostgresIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
