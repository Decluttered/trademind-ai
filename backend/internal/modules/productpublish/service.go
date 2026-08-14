package productpublish

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/modules/idempotency"
	"github.com/trademind-ai/trademind/backend/internal/modules/operationlog"
	"github.com/trademind-ai/trademind/backend/internal/modules/productcheck"
	"github.com/trademind-ai/trademind/backend/internal/modules/productioncontrolp10"
	"github.com/trademind-ai/trademind/backend/internal/modules/settings"
	"github.com/trademind-ai/trademind/backend/internal/modules/shop"
	"github.com/trademind-ai/trademind/backend/internal/pkg/adminperm"
	"github.com/trademind-ai/trademind/backend/internal/pkg/security"
	"github.com/trademind-ai/trademind/backend/internal/rdb"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Service wires DB + outbound provider execution for product_publish_tasks.
type Service struct {
	DB               *gorm.DB
	Redis            *rdb.Client
	Shops            *shop.Service
	Settings         *settings.Service
	OpLog            *operationlog.Service
	Readiness        *productcheck.Service
	Idempotency      *idempotency.Service
	WriteControl     *productioncontrolp10.Service
	OperationResults OperationTaskResultSink
	ProductionOutbox ProductionOutboxDispatcher
	Environment      string

	QueueEnabled bool
	QueueName    string
	TaskTimeout  time.Duration

	BatchMaxProducts int
	BatchMaxTargets  int
	BatchMaxTasks    int
}

func (s *Service) traditionalPublishAllowed() bool {
	if s == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(s.Environment)) {
	case "staging", "production":
		return false
	default:
		return true
	}
}

func (s *Service) ensureStoreOperate(c *gin.Context, shopID uuid.UUID) error {
	if s == nil || s.DB == nil || c == nil || shopID == uuid.Nil {
		return gorm.ErrRecordNotFound
	}
	principal, err := adminperm.LoadPrincipal(c, s.DB)
	if err != nil {
		return err
	}
	if principal == nil || !principal.CanOperateStore(shopID) {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Service) ensureTargetStoresOperate(c *gin.Context, targets []PublishTargetRef) error {
	for _, target := range targets {
		if target.ShopID == nil || strings.TrimSpace(*target.ShopID) == "" {
			continue
		}
		shopID, err := uuid.Parse(strings.TrimSpace(*target.ShopID))
		if err != nil || shopID == uuid.Nil {
			return gorm.ErrRecordNotFound
		}
		if _, err := s.loadTargetStore(c.Request.Context(), shopID, target.Platform); err != nil {
			return err
		}
		if err := s.ensureStoreOperate(c, shopID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) validateTargetStores(ctx context.Context, targets []PublishTargetRef) error {
	for _, target := range targets {
		if target.ShopID == nil || strings.TrimSpace(*target.ShopID) == "" {
			continue
		}
		shopID, err := uuid.Parse(strings.TrimSpace(*target.ShopID))
		if err != nil || shopID == uuid.Nil {
			return gorm.ErrRecordNotFound
		}
		if _, err := s.loadTargetStore(ctx, shopID, target.Platform); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) loadTargetStore(ctx context.Context, shopID uuid.UUID, platform string) (*shop.Shop, error) {
	if s == nil || s.DB == nil || shopID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	tenant, err := security.RequireTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	var row shop.Shop
	if err := s.DB.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", shopID, tenant.TenantID).
		First(&row).Error; err != nil {
		return nil, err
	}
	if strings.TrimSpace(strings.ToLower(row.Platform)) != strings.TrimSpace(strings.ToLower(platform)) {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

type ProductionOutboxDispatcher interface {
	DispatchPending(ctx context.Context, limit int) (int, error)
}

type OperationTaskResultSink interface {
	MarkRunning(ctx context.Context, attemptID, downstreamTaskID uuid.UUID) error
	MarkSucceeded(ctx context.Context, attemptID, downstreamTaskID uuid.UUID, externalReference, requestID string, metadata datatypes.JSON) error
	MarkFailed(ctx context.Context, attemptID, downstreamTaskID uuid.UUID, code, safeMessage string, retryable, resultUnknown bool, metadata datatypes.JSON) error
}

func (s *Service) normalizedQueueName() string {
	q := strings.TrimSpace(s.QueueName)
	if q == "" {
		return "product:publish:tasks"
	}
	return q
}
