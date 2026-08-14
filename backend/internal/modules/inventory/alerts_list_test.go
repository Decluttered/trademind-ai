package inventory

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/modules/product"
	"github.com/trademind-ai/trademind/backend/internal/modules/productpublish"
	"github.com/trademind-ai/trademind/backend/internal/modules/shop"
	"gorm.io/gorm"
)

func newInventoryAlertsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, db.AutoMigrate(
		&product.Product{},
		&product.ProductSKU{},
		&productpublish.ProductPublication{},
		&productpublish.ProductPublicationSKU{},
		&shop.Shop{},
		&InventorySyncTask{},
	))
	return db
}

func TestListInventoryAlertsUsesCanonicalPublicationSKUColumn(t *testing.T) {
	db := newInventoryAlertsTestDB(t)
	item := product.Product{Source: "manual", Title: "Low stock item", Status: product.StatusDraft}
	require.NoError(t, db.Create(&item).Error)

	stock := 0
	sku := product.ProductSKU{
		ProductID:    item.ID,
		SKUCode:      "ALERT-SKU-1",
		SKUName:      "Alert SKU",
		Stock:        &stock,
		WarningStock: 5,
	}
	require.NoError(t, db.Create(&sku).Error)

	result, err := (&Service{DB: db}).ListInventoryAlerts(context.Background(), AlertsListQuery{
		Page:     1,
		PageSize: 20,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, sku.ID, result.Items[0].ProductSKUID)
	require.Contains(t, result.Items[0].AlertTypes, AlertTypeOutOfStock)
}
