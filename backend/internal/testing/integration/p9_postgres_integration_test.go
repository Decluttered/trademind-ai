//go:build p9postgres

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/database"
	"github.com/trademind-ai/trademind/backend/internal/modules/admin"
	"github.com/trademind-ai/trademind/backend/internal/modules/product"
	"github.com/trademind-ai/trademind/backend/internal/pkg/ctxkey"
	"github.com/trademind-ai/trademind/backend/internal/pkg/model"
	"github.com/trademind-ai/trademind/backend/internal/pkg/response"
	"github.com/trademind-ai/trademind/backend/internal/pkg/security"
	"github.com/trademind-ai/trademind/backend/internal/testing/postgrestest"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type p9SKUSearchFixture struct {
	productA product.Product
	productB product.Product
}

func p9SeedTenantSKUs(t *testing.T, db interface {
	Create(value any) *gorm.DB
	Model(value any) *gorm.DB
}, tenantID int64, suffix string) (product.Product, []product.ProductSKU) {
	t.Helper()
	item := product.Product{TenantID: tenantID, Source: "manual", Title: "Shared PostgreSQL Product " + suffix, Status: product.StatusDraft}
	require.NoError(t, db.Create(&item).Error)
	stock := 5
	skus := []product.ProductSKU{
		{ProductID: item.ID, SKUCode: "P9PG-SHARED-CODE", SKUName: "P9PG Shared Blue " + suffix, Stock: &stock, StockStatus: "normal", RawData: datatypes.JSON([]byte(`{"barcode":"P9PG-SHARED-BARCODE"}`))},
		{ProductID: item.ID, SKUCode: "P9PG-SECOND-" + suffix, SKUName: "P9PG Shared Green " + suffix, Stock: &stock, StockStatus: "low_stock", RawData: datatypes.JSON([]byte(fmt.Sprintf(`{"barcode":"P9PG-%s-BARCODE"}`, suffix)))},
	}
	require.NoError(t, db.Create(&skus).Error)
	return item, skus
}

func p9SKUSearchRequest(t *testing.T, handler *product.Handler, tenantID int64, actorID uuid.UUID, query string) (response.Envelope, int) {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/product-skus/search"+query, nil)
	ctx.Set(ctxkey.TraceID, "trace-p9pg-sku-search")
	ctx.Set(ctxkey.TenantID, tenantID)
	ctx.Set(ctxkey.AdminID, actorID.String())
	security.SetGin(ctx, &security.TenantContext{TenantID: tenantID, UserID: actorID, RequestID: "trace-p9pg-sku-search", AuthSource: security.AuthSourceAccessToken})
	handler.SearchSKUs(ctx)
	var envelope response.Envelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	return envelope, recorder.Code
}

func p9SKUSearchRows(t *testing.T, envelope response.Envelope) []map[string]any {
	t.Helper()
	data, ok := envelope.Data.(map[string]any)
	require.True(t, ok)
	raw, ok := data["list"].([]any)
	require.True(t, ok)
	rows := make([]map[string]any, 0, len(raw))
	for _, value := range raw {
		row, ok := value.(map[string]any)
		require.True(t, ok)
		rows = append(rows, row)
	}
	return rows
}

func TestP9PostgresTenantScopedProductSKUSearch(t *testing.T) {
	harness := postgrestest.Require(t)
	harness.EmitMetadata(t)
	db := harness.DB
	require.NoError(t, database.AutoMigrate(db))
	gin.SetMode(gin.TestMode)

	productA, _ := p9SeedTenantSKUs(t, db, 910101, "TENANT-A")
	productB, _ := p9SeedTenantSKUs(t, db, 910202, "TENANT-B")
	require.NoError(t, db.Model(&product.Product{}).Where("id = ?", productA.ID).Update("updated_at", time.Now().UTC().Add(-time.Hour)).Error)
	require.NoError(t, db.Model(&product.Product{}).Where("id = ?", productB.ID).Update("updated_at", time.Now().UTC()).Error)
	actorA := uuid.New()
	actorB := uuid.New()
	require.NoError(t, db.Create(&admin.AdminUser{Base: model.Base{ID: actorA}, TenantID: 910101, Username: admin.NewInternalUsername(), PasswordHash: "test", Role: admin.RoleAdmin, Status: admin.StatusActive}).Error)
	require.NoError(t, db.Create(&admin.AdminUser{Base: model.Base{ID: actorB}, TenantID: 910202, Username: admin.NewInternalUsername(), PasswordHash: "test", Role: admin.RoleAdmin, Status: admin.StatusActive}).Error)
	handler := &product.Handler{Svc: &product.Service{DB: db}}

	cases := []struct {
		name      string
		tenantID  int64
		actorID   uuid.UUID
		query     string
		productID uuid.UUID
		want      int
	}{
		{name: "tenant A default", tenantID: 910101, actorID: actorA, productID: productA.ID, want: 2},
		{name: "tenant B default", tenantID: 910202, actorID: actorB, productID: productB.ID, want: 2},
		{name: "shared sku code", tenantID: 910101, actorID: actorA, query: "?keyword=P9PG-SHARED-CODE", productID: productA.ID, want: 1},
		{name: "shared sku name", tenantID: 910101, actorID: actorA, query: "?keyword=P9PG%20Shared%20Blue", productID: productA.ID, want: 1},
		{name: "shared product title", tenantID: 910101, actorID: actorA, query: "?keyword=Shared%20PostgreSQL%20Product", productID: productA.ID, want: 2},
		{name: "own product id", tenantID: 910101, actorID: actorA, query: "?productId=" + productA.ID.String(), productID: productA.ID, want: 2},
		{name: "own product plus keyword", tenantID: 910101, actorID: actorA, query: "?productId=" + productA.ID.String() + "&keyword=P9PG%20Shared%20Blue", productID: productA.ID, want: 1},
		{name: "tenant spoof ignored", tenantID: 910101, actorID: actorA, query: "?keyword=Shared&tenantId=910202&tenant_id=910202&status=normal&barcode=P9PG-SHARED-BARCODE", productID: productA.ID, want: 2},
		{name: "foreign rows cannot displace limit", tenantID: 910101, actorID: actorA, query: "?keyword=Shared&limit=1", productID: productA.ID, want: 1},
		{name: "limit two stays local", tenantID: 910101, actorID: actorA, query: "?keyword=Shared&limit=2", productID: productA.ID, want: 2},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			envelope, status := p9SKUSearchRequest(t, handler, test.tenantID, test.actorID, test.query)
			require.Equal(t, http.StatusOK, status)
			require.Equal(t, response.CodeOK, envelope.Code)
			require.Equal(t, "trace-p9pg-sku-search", envelope.TraceID)
			rows := p9SKUSearchRows(t, envelope)
			require.Len(t, rows, test.want)
			for _, row := range rows {
				require.Equal(t, test.productID.String(), row["productId"])
				require.NotContains(t, row, "tenantId")
			}
		})
	}

	for _, query := range []string{
		"?productId=" + productB.ID.String(),
		"?productId=" + productB.ID.String() + "&keyword=Shared",
		"?productId=" + productB.ID.String() + "&keyword=TENANT-B",
	} {
		envelope, status := p9SKUSearchRequest(t, handler, 910101, actorA, query)
		require.Equal(t, http.StatusOK, status)
		require.Empty(t, p9SKUSearchRows(t, envelope))
	}

	for range 3 {
		envelope, status := p9SKUSearchRequest(t, handler, 910101, actorA, "?keyword=Shared&limit=1")
		require.Equal(t, http.StatusOK, status)
		rows := p9SKUSearchRows(t, envelope)
		require.Len(t, rows, 1)
		require.Equal(t, productA.ID.String(), rows[0]["productId"])
	}

	barcodeOnly, status := p9SKUSearchRequest(t, handler, 910101, actorA, "?keyword=P9PG-SHARED-BARCODE")
	require.Equal(t, http.StatusOK, status)
	require.Empty(t, p9SKUSearchRows(t, barcodeOnly), "barcode remains outside the existing public search contract")
}

func TestP9PostgresAutoMigrateAgainstIsolatedDatabase(t *testing.T) {
	harness := postgrestest.Require(t)
	harness.EmitMetadata(t)
	db := harness.DB

	require.NoError(t, database.AutoMigrate(db))
	require.NoError(t, database.AutoMigrate(db))

	for _, table := range []string{
		"admin_users",
		"products",
		"product_skus",
		"p9_inventory_sync_runs",
		"p9_inventory_snapshot_items",
		"p9_sku_bindings",
		"p9_sku_binding_calibrations",
		"p9_manual_binding_requests",
		"p9_manual_binding_decisions",
	} {
		require.Truef(t, db.Migrator().HasTable(table), "expected migrated table %s", table)
	}
}
