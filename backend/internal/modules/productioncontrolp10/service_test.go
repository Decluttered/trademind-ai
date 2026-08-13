package productioncontrolp10

import (
	"context"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/config"
	"github.com/trademind-ai/trademind/backend/internal/modules/product"
	"github.com/trademind-ai/trademind/backend/internal/modules/shop"
	"github.com/trademind-ai/trademind/backend/internal/pkg/model"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openControlTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:production_control_%s?mode=memory&cache=shared", uuid.NewString())), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&RuntimeControl{}, &ScopeAllowlist{}, &GrayPolicy{}, &ControlAuditEvent{}, &shop.Shop{}, &product.Product{}))
	require.NoError(t, db.Exec(`CREATE TABLE inventory_sync_runs (
		id text PRIMARY KEY,
		tenant_id integer NOT NULL,
		shop_connection_id text NOT NULL,
		status text NOT NULL,
		provider_mode text NOT NULL,
		revision integer NOT NULL,
		request_id text,
		safe_error_metadata blob,
		started_at datetime,
		finished_at datetime,
		created_at datetime NOT NULL
	)`).Error)
	return db
}

func TestStatusAllowsLegacyTenantZeroOnlyInDevelopment(t *testing.T) {
	db := openControlTestDB(t)
	ctx := context.Background()

	development := &Service{DB: db, Config: &config.Config{AppEnv: config.EnvDevelopment}}
	status, err := development.Status(ctx, 0, nil)
	require.NoError(t, err)
	require.Equal(t, int64(0), status.Control.TenantID)
	require.True(t, status.Control.ProviderKillActive)
	require.False(t, status.ProductionReady)

	production := &Service{DB: db, Config: &config.Config{AppEnv: config.EnvProduction}}
	_, err = production.Status(ctx, 0, nil)
	require.ErrorIs(t, err, ErrInvalidControl)

	_, err = development.Status(ctx, -1, nil)
	require.ErrorIs(t, err, ErrInvalidControl)
}

func TestGrayApprovalRequiresTwoDifferentAdminsBeforeActivation(t *testing.T) {
	db := openControlTestDB(t)
	svc := &Service{DB: db, Config: &config.Config{P10: enabledDraftWriteConfig(), ProductPublishQueueEnabled: true}, ProviderWriteGuard: allowProviderWrite}
	ctx := context.Background()
	tenantID := int64(101)
	shopID := uuid.New()
	owner := Actor{TenantID: tenantID, UserID: uuid.New(), RequestID: "req-owner"}
	technicalLead := Actor{TenantID: tenantID, UserID: uuid.New(), RequestID: "req-tech"}

	gray, err := svc.SaveGrayDraft(ctx, owner, shopID, 20, 0)
	require.NoError(t, err)
	gray, err = svc.ApproveGray(ctx, owner, "owner", gray.Revision)
	require.NoError(t, err)
	_, err = svc.ApproveGray(ctx, owner, "technical_lead", gray.Revision)
	require.ErrorIs(t, err, ErrBlocked)
	gray, err = svc.ApproveGray(ctx, technicalLead, "technical_lead", gray.Revision)
	require.NoError(t, err)
	require.Equal(t, GrayApproved, gray.Status)
	require.NotEqual(t, *gray.OwnerApprovedBy, *gray.TechnicalLeadApprovedBy)
	gray, err = svc.ActivateGray(ctx, owner, gray.Revision)
	require.NoError(t, err)
	require.Equal(t, GrayActive, gray.Status)
}

func TestEvaluateWriteRequiresOwnedAuthorizedAllowlistedActiveScope(t *testing.T) {
	db := openControlTestDB(t)
	ctx := context.Background()
	tenantID := int64(101)
	shopID := uuid.New()
	productID := uuid.New()
	require.NoError(t, db.Create(&shop.Shop{Base: model.Base{ID: shopID}, TenantID: tenantID, Platform: "douyin_shop", ShopName: "Test shop", Status: shop.StatusActive, AuthStatus: shop.AuthAuthorized}).Error)
	require.NoError(t, db.Create(&product.Product{Base: model.Base{ID: productID}, TenantID: tenantID, Source: "test", Title: "Test product", Status: product.StatusDraft}).Error)
	require.NoError(t, db.Create(&RuntimeControl{TenantID: tenantID, Revision: 1}).Error)
	require.NoError(t, db.Model(&RuntimeControl{}).Where("tenant_id = ?", tenantID).Updates(map[string]any{
		"provider_kill_active": false,
		"tenant_kill_active":   false,
		"shop_kill_active":     false,
		"write_kill_active":    false,
	}).Error)
	require.NoError(t, db.Create(&ScopeAllowlist{TenantID: tenantID, ShopID: shopID, Enabled: true, Revision: 1}).Error)
	ownerID, leadID := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&GrayPolicy{TenantID: tenantID, ShopID: shopID, MaxSKU: 2, Status: GrayActive, OwnerApproved: true, TechnicalLeadApproved: true, OwnerApprovedBy: &ownerID, TechnicalLeadApprovedBy: &leadID, Revision: 1}).Error)
	svc := &Service{DB: db, Config: &config.Config{P10: enabledDraftWriteConfig(), ProductPublishQueueEnabled: true}, ProviderWriteGuard: allowProviderWrite}

	require.NoError(t, svc.EvaluateWrite(ctx, tenantID, shopID, productID, 2))
	svc.Config.ProductPublishQueueEnabled = false
	require.ErrorIs(t, svc.EvaluateWrite(ctx, tenantID, shopID, productID, 2), ErrBlocked)
	svc.Config.ProductPublishQueueEnabled = true
	require.ErrorIs(t, svc.EvaluateWrite(ctx, tenantID, shopID, productID, 3), ErrBlocked)
	require.ErrorIs(t, svc.EvaluateWrite(ctx, tenantID+1, shopID, productID, 1), ErrBlocked)
	require.NoError(t, db.Model(&RuntimeControl{}).Where("tenant_id = ?", tenantID).Update("write_kill_active", true).Error)
	require.ErrorIs(t, svc.EvaluateWrite(ctx, tenantID, shopID, productID, 1), ErrBlocked)
}

func TestProviderWriteGuardFailsClosedForEvaluationAndStatus(t *testing.T) {
	db := openControlTestDB(t)
	ctx := context.Background()
	tenantID, shopID, productID := seedReadyWriteScope(t, db)
	svc := &Service{
		DB:     db,
		Config: &config.Config{AppEnv: "production", P10: enabledDraftWriteConfig(), ProductPublishQueueEnabled: true},
		ProviderWriteGuard: func(context.Context, uuid.UUID) error {
			return fmt.Errorf("provider write disabled")
		},
	}

	require.ErrorIs(t, svc.EvaluateWrite(ctx, tenantID, shopID, productID, 1), ErrBlocked)
	status, err := svc.Status(ctx, tenantID, []uuid.UUID{shopID})
	require.NoError(t, err)
	require.False(t, status.ProviderWriteReady)
	require.False(t, status.ProductionReady)
}

func TestProviderWriteGuardAllowsReadyStatus(t *testing.T) {
	db := openControlTestDB(t)
	tenantID, shopID, _ := seedReadyWriteScope(t, db)
	svc := &Service{
		DB:                 db,
		Config:             &config.Config{AppEnv: "production", P10: enabledDraftWriteConfig(), ProductPublishQueueEnabled: true},
		ProviderWriteGuard: allowProviderWrite,
	}

	status, err := svc.Status(context.Background(), tenantID, []uuid.UUID{shopID})
	require.NoError(t, err)
	require.True(t, status.ProviderWriteReady)
	require.True(t, status.ProductionReady)
}

func allowProviderWrite(context.Context, uuid.UUID) error { return nil }

func seedReadyWriteScope(t *testing.T, db *gorm.DB) (int64, uuid.UUID, uuid.UUID) {
	t.Helper()
	tenantID := int64(101)
	shopID, productID := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&shop.Shop{Base: model.Base{ID: shopID}, TenantID: tenantID, Platform: "douyin_shop", ShopName: "Test shop", Status: shop.StatusActive, AuthStatus: shop.AuthAuthorized}).Error)
	require.NoError(t, db.Create(&product.Product{Base: model.Base{ID: productID}, TenantID: tenantID, Source: "test", Title: "Test product", Status: product.StatusDraft}).Error)
	require.NoError(t, db.Create(&RuntimeControl{TenantID: tenantID, Revision: 1}).Error)
	require.NoError(t, db.Model(&RuntimeControl{}).Where("tenant_id = ?", tenantID).Updates(map[string]any{
		"provider_kill_active": false,
		"tenant_kill_active":   false,
		"shop_kill_active":     false,
		"write_kill_active":    false,
	}).Error)
	require.NoError(t, db.Create(&ScopeAllowlist{TenantID: tenantID, ShopID: shopID, Enabled: true, Revision: 1}).Error)
	ownerID, leadID := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&GrayPolicy{TenantID: tenantID, ShopID: shopID, MaxSKU: 100, Status: GrayActive, OwnerApproved: true, TechnicalLeadApproved: true, OwnerApprovedBy: &ownerID, TechnicalLeadApprovedBy: &leadID, Revision: 1}).Error)
	return tenantID, shopID, productID
}

func TestScopeAllowlistEnforcesSingleTenantAndSingleShop(t *testing.T) {
	db := openControlTestDB(t)
	require.NoError(t, db.Create(&ScopeAllowlist{TenantID: 101, ShopID: uuid.New(), Enabled: true, Revision: 1}).Error)
	require.Error(t, db.Create(&ScopeAllowlist{TenantID: 101, ShopID: uuid.New(), Enabled: false, Revision: 1}).Error)
	require.Error(t, db.Create(&ScopeAllowlist{TenantID: 202, ShopID: uuid.New(), Enabled: true, Revision: 1}).Error)
	require.NoError(t, db.Create(&ScopeAllowlist{TenantID: 202, ShopID: uuid.New(), Enabled: false, Revision: 1}).Error)
}

func enabledDraftWriteConfig() config.P10Config {
	return config.P10Config{
		CurrentAllowedLevel: "L3", RealProviderEnabled: true, RealPlatformNetworkEnabled: true,
		RealCredentialsEnabled: true, RealProductDraftWriteEnabled: true, BackgroundWorkerEnabled: true,
	}
}
