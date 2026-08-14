package database

import (
	"fmt"

	"github.com/trademind-ai/trademind/backend/internal/modules/productpublish"
	"gorm.io/gorm"
)

// migrateProductPublishTenant backfills tenant ownership for publication records
// created before tenant columns became authoritative.
func migrateProductPublishTenant(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("product publish tenant migration: db is nil")
	}
	if err := db.AutoMigrate(&productpublish.ProductPublishTask{}, &productpublish.ProductPublishBatch{}, &productpublish.ProductPublication{}); err != nil {
		return err
	}
	statements := []string{
		`UPDATE product_publish_tasks
SET tenant_id = (SELECT products.tenant_id FROM products WHERE products.id = product_publish_tasks.product_id)
WHERE tenant_id = 0
  AND EXISTS (SELECT 1 FROM products WHERE products.id = product_publish_tasks.product_id AND products.tenant_id <> 0)`,
		`UPDATE product_publications
SET tenant_id = (SELECT products.tenant_id FROM products WHERE products.id = product_publications.product_id)
WHERE tenant_id = 0
  AND EXISTS (SELECT 1 FROM products WHERE products.id = product_publications.product_id AND products.tenant_id <> 0)`,
		`UPDATE product_publish_batches
SET tenant_id = (SELECT products.tenant_id FROM products WHERE products.id = product_publish_batches.product_id)
WHERE tenant_id = 0
  AND product_id IS NOT NULL
  AND EXISTS (SELECT 1 FROM products WHERE products.id = product_publish_batches.product_id AND products.tenant_id <> 0)
  AND NOT EXISTS (
    SELECT 1
    FROM product_publish_tasks
    WHERE product_publish_tasks.batch_id = product_publish_batches.id
      AND product_publish_tasks.tenant_id <> 0
      AND product_publish_tasks.tenant_id <> (
        SELECT products.tenant_id FROM products WHERE products.id = product_publish_batches.product_id
      )
  )`,
		`UPDATE product_publish_batches
SET tenant_id = (
  SELECT MIN(product_publish_tasks.tenant_id)
  FROM product_publish_tasks
  WHERE product_publish_tasks.batch_id = product_publish_batches.id
    AND product_publish_tasks.tenant_id <> 0
)
WHERE tenant_id = 0
  AND product_id IS NULL
  AND 1 = (
    SELECT COUNT(DISTINCT product_publish_tasks.tenant_id)
    FROM product_publish_tasks
    WHERE product_publish_tasks.batch_id = product_publish_batches.id
      AND product_publish_tasks.tenant_id <> 0
  )`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("product publish tenant backfill: %w", err)
		}
	}
	return nil
}
