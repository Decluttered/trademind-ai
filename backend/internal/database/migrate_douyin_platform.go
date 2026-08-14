package database

import (
	"fmt"

	"github.com/trademind-ai/trademind/backend/internal/modules/ordersync"
	"github.com/trademind-ai/trademind/backend/internal/modules/product"
	"github.com/trademind-ai/trademind/backend/internal/modules/shop"
	"gorm.io/gorm"
)

// migrateDouyinPlatform adds Douyin adapter models and optional column additions.
func migrateDouyinPlatform(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("migrate Douyin platform: db is nil")
	}

	if err := db.AutoMigrate(
		&shop.DouyinOAuthState{},
		&product.DouyinImageAsset{},
		&ordersync.DouyinSyncCursor{},
	); err != nil {
		return fmt.Errorf("Douyin platform AutoMigrate: %w", err)
	}

	// Add token_version and reauthorization_required to shop_auth_tokens if missing.
	if err := migrateDouyinShopAuthTokenColumns(db); err != nil {
		return err
	}

	// Partial indexes (PostgreSQL only).
	if err := migrateDouyinPlatformIndexes(db); err != nil {
		return err
	}

	return nil
}

func migrateDouyinShopAuthTokenColumns(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	driver := ""
	if db.Dialector != nil {
		driver = db.Dialector.Name()
	}
	var stmts []string
	switch driver {
	case "postgres":
		stmts = []string{
			`ALTER TABLE shop_auth_tokens ADD COLUMN IF NOT EXISTS token_version BIGINT NOT NULL DEFAULT 0`,
			`ALTER TABLE shop_auth_tokens ADD COLUMN IF NOT EXISTS reauthorization_required BOOLEAN NOT NULL DEFAULT FALSE`,
			`ALTER TABLE shop_auth_tokens ADD COLUMN IF NOT EXISTS last_refresh_error_code VARCHAR(128)`,
		}
	case "mysql":
		// MySQL: use IF NOT EXISTS equivalent via information_schema check is complex;
		// use raw ALTER and ignore "Duplicate column name" errors.
		stmts = []string{
			`ALTER TABLE shop_auth_tokens ADD COLUMN IF NOT EXISTS token_version BIGINT NOT NULL DEFAULT 0`,
			`ALTER TABLE shop_auth_tokens ADD COLUMN IF NOT EXISTS reauthorization_required TINYINT(1) NOT NULL DEFAULT 0`,
			`ALTER TABLE shop_auth_tokens ADD COLUMN IF NOT EXISTS last_refresh_error_code VARCHAR(128)`,
		}
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			// Tolerate "column already exists" errors
			if !isDuplicateColumnError(err) {
				return fmt.Errorf("Douyin shop_auth_tokens migration: %w", err)
			}
		}
	}
	return nil
}

func migrateDouyinPlatformIndexes(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	driver := ""
	if db.Dialector != nil {
		driver = db.Dialector.Name()
	}
	if driver != "postgres" {
		return nil
	}
	stmts := []string{
		`CREATE INDEX IF NOT EXISTS ix_douyin_image_assets_shop_hash ON douyin_image_assets (shop_id, content_hash)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS ux_douyin_image_assets_shop_hash ON douyin_image_assets (shop_id, content_hash) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS ix_douyin_oauth_states_expires ON douyin_oauth_states (expires_at) WHERE consumed_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS ix_douyin_sync_cursors_shop_type ON douyin_sync_cursors (shop_id, sync_type)`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("Douyin platform index: %w", err)
		}
	}
	return nil
}

// isDuplicateColumnError returns true for "column already exists" database errors.
func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, marker := range []string{
		"already exists",
		"Duplicate column",
		"column already exists",
	} {
		if len(msg) > 0 && containsInsensitive(msg, marker) {
			return true
		}
	}
	return false
}

func containsInsensitive(s, substr string) bool {
	if len(s) == 0 || len(substr) == 0 {
		return false
	}
	sl := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		sl[i] = c
	}
	subl := make([]byte, len(substr))
	for i := 0; i < len(substr); i++ {
		c := substr[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		subl[i] = c
	}
	return contains(string(sl), string(subl))
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
