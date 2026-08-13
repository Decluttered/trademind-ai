package api

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/middleware"
	"github.com/trademind-ai/trademind/backend/internal/modules/admin"
	"github.com/trademind-ai/trademind/backend/internal/modules/auth"
	"github.com/trademind-ai/trademind/backend/internal/modules/credentialp10"
	"github.com/trademind-ai/trademind/backend/internal/modules/inventoryreadp10"
	"github.com/trademind-ai/trademind/backend/internal/modules/productioncontrolp10"
	"github.com/trademind-ai/trademind/backend/internal/pkg/metrics"
	platformdouyin "github.com/trademind-ai/trademind/backend/internal/providers/platform/douyinshop"
)

// RegisterP10 mounts P10-owned routes without changing the frozen P9 API router.
func RegisterP10(r gin.IRouter, dep *Deps) {
	if dep == nil {
		dep = &Deps{}
	}

	var metricCatalog *metrics.Catalog
	if dep.Obs != nil {
		metricCatalog = dep.Obs.Catalog
	}
	sessions := &auth.SessionService{
		Cfg: dep.Config, DB: dep.DB, Admins: &admin.Store{DB: dep.DB}, Metrics: metricCatalog,
	}
	authed := r.Group("/api/v1")
	authed.Use(middleware.BearerAuthWithDB(dep.Config, dep.DB, sessions))

	controlSvc := &productioncontrolp10.Service{DB: dep.DB, Config: dep.Config, ProviderWriteGuard: func(ctx context.Context, shopID uuid.UUID) error {
		if guardErr := platformdouyin.GuardWorkerWithShop(ctx, shopID.String(), platformdouyin.FeatureProductDraft, true, false); guardErr != nil {
			return guardErr
		}
		return nil
	}}
	productioncontrolp10.Register(authed, &productioncontrolp10.Handler{Service: controlSvc})

	var cipher credentialp10.CredentialCipher
	if dep.Config != nil && strings.TrimSpace(dep.Config.P10.LocalCredentialKey) != "" {
		if keys, err := credentialp10.NewLocalKeyProvider(dep.Config.P10.LocalCredentialKeyRef, dep.Config.P10.LocalCredentialKey); err == nil {
			cipher = credentialp10.AESGCMCredentialCipher{Keys: keys}
		}
	}
	credentialSvc := &credentialp10.Service{
		DB: dep.DB, Cipher: cipher, Metrics: metricCatalog, Environment: "unknown",
	}
	if dep.Config != nil {
		credentialSvc.Environment = dep.Config.AppEnv
		credentialSvc.OfflineEnabled = dep.Config.P10.OfflineOAuthEnabled
		credentialSvc.RedirectAllowlist = append([]string(nil), dep.Config.P10.RedirectAllowlist...)
		credentialSvc.OAuthStateTTL = dep.Config.P10.OAuthStateTTL
	}
	credentialp10.Register(authed, &credentialp10.Handler{Service: credentialSvc})

	provider := &inventoryreadp10.DouyinReadOnlyInventoryProvider{
		Config: dep.Config, Credentials: credentialSvc, Guard: controlSvc,
		Source: inventoryreadp10.PublicationSource{DB: dep.DB},
	}
	pageSize, maxPages := 50, 100
	if dep.Config != nil {
		pageSize, maxPages = dep.Config.P10.SKUPageSize, dep.Config.P10.PaginationLimit
	}
	readSvc := inventoryreadp10.NewManualReadService(dep.DB, provider, metricCatalog, pageSize, maxPages)
	if dep.Config != nil {
		readSvc.Environment = dep.Config.AppEnv
	}
	inventoryreadp10.Register(authed, &inventoryreadp10.Handler{Service: readSvc})
}
