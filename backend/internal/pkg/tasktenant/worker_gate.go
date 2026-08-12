package tasktenant

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BeginWorker builds a tenant-scoped worker context from explicit tenant and optional shop.
// When tenantID <= 0 and shopID is set, tenant is resolved from the shop row.
func BeginWorker(ctx context.Context, db *gorm.DB, tenantID int64, shopID uuid.UUID, operation string) (context.Context, TaskScope, error) {
	return beginWorker(ctx, db, tenantID, shopID, operation, false)
}

// BeginWorkerWithLegacyZero permits tenant zero only when the caller has
// already derived allowLegacyTenantZero from trusted environment config.
func BeginWorkerWithLegacyZero(ctx context.Context, db *gorm.DB, tenantID int64, shopID uuid.UUID, operation string, allowLegacyTenantZero bool) (context.Context, TaskScope, error) {
	return beginWorker(ctx, db, tenantID, shopID, operation, allowLegacyTenantZero)
}

func beginWorker(ctx context.Context, db *gorm.DB, tenantID int64, shopID uuid.UUID, operation string, allowLegacyTenantZero bool) (context.Context, TaskScope, error) {
	tid := tenantID
	if tid <= 0 && shopID != uuid.Nil && db != nil {
		resolved, err := ResolveShopTenant(ctx, db, shopID)
		if err != nil {
			return ctx, TaskScope{}, err
		}
		tid = resolved
	}
	if err := RequireTaskTenantForMode(tid, allowLegacyTenantZero); err != nil {
		return ctx, TaskScope{}, err
	}
	scope := TaskScope{TenantID: tid, ShopID: shopID, AllowLegacyTenantZero: allowLegacyTenantZero}
	wctx := BuildWorkerContext(scope, uuid.Nil, operation)
	return wctx, scope, nil
}

// BeginWorkerFromShop resolves tenant from shop and returns worker context.
func BeginWorkerFromShop(ctx context.Context, db *gorm.DB, shopID uuid.UUID, operation string) (context.Context, TaskScope, error) {
	return BeginWorker(ctx, db, 0, shopID, operation)
}
