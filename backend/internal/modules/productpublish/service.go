package productpublish

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/modules/idempotency"
	"github.com/trademind-ai/trademind/backend/internal/modules/operationlog"
	"github.com/trademind-ai/trademind/backend/internal/modules/productcheck"
	"github.com/trademind-ai/trademind/backend/internal/modules/productioncontrolp10"
	"github.com/trademind-ai/trademind/backend/internal/modules/settings"
	"github.com/trademind-ai/trademind/backend/internal/modules/shop"
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

	QueueEnabled bool
	QueueName    string
	TaskTimeout  time.Duration

	BatchMaxProducts int
	BatchMaxTargets  int
	BatchMaxTasks    int
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
