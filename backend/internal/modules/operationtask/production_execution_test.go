package operationtask_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/modules/admin"
	"github.com/trademind-ai/trademind/backend/internal/modules/operationtask"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type allowProductionWrite struct {
	calls int
}

func (g *allowProductionWrite) EvaluateWrite(context.Context, int64, uuid.UUID, uuid.UUID, int) error {
	g.calls++
	return nil
}

type fakeProductionFactory struct {
	taskID uuid.UUID
}

func (f fakeProductionFactory) CreateFrozenDouyinDraftTask(context.Context, *gorm.DB, operationtask.ProductionDownstreamInput) (uuid.UUID, error) {
	return f.taskID, nil
}

type recordingOutboxDelivery struct {
	calls int
	err   error
}

func (d *recordingOutboxDelivery) EnqueueProductionTask(context.Context, uuid.UUID) error {
	d.calls++
	return d.err
}

func TestParseDouyinDraftIntentRejectsTrailingContent(t *testing.T) {
	valid := `{"schemaVersion":"douyin_draft_v1","productId":"` + uuid.NewString() + `","shopId":"` + uuid.NewString() + `","publishMode":"save_as_platform_draft"}`
	_, err := operationtask.ParseDouyinDraftIntent([]byte(valid))
	require.NoError(t, err)

	for _, raw := range []string{valid + `{}`, valid + ` trailing`} {
		_, err := operationtask.ParseDouyinDraftIntent([]byte(raw))
		require.ErrorIs(t, err, operationtask.ErrValidation)
	}
}

func TestProductionDraftDetailDoesNotExposeFrozenProviderRequest(t *testing.T) {
	db := openOperationTaskTestDB(t)
	ctx := context.Background()
	tenantID := int64(101)
	actorID := createAdminUser(t, db, tenantID, admin.RoleAdmin, admin.StatusActive)
	productID := uuid.New()
	shopID := uuid.New()
	frozenRaw, err := json.Marshal(operationtask.FrozenDouyinDraft{
		SchemaVersion: operationtask.DouyinDraftSchemaVersion,
		ProductID:     productID, ShopID: shopID, PublishMode: operationtask.DouyinDraftPublishMode, SKUCount: 1,
		Request:         json.RawMessage(`{"name":"Frozen title","publishConfig":{"default_mobile":"13800000000"},"specPricesV2":[{"price":100}]}`),
		Review:          json.RawMessage(`{"name":"Frozen title"}`),
		MappingSnapshot: json.RawMessage(`{"platform":"douyin_shop","productId":"` + productID.String() + `","shopId":"` + shopID.String() + `"}`),
		MappingHash:     hash1,
	})
	require.NoError(t, err)
	payloadHash, err := operationtask.ComputePayloadHash(frozenRaw)
	require.NoError(t, err)
	task := operationtask.OperationTask{
		TenantID: tenantID, SourceType: operationtask.OperationTaskSourceManual, SourceReference: productID.String(),
		TaskType: operationtask.OperationTaskTypeProductPublish, Platform: operationtask.PlatformDouyin, Title: "Reviewed platform draft",
		Payload: datatypes.JSON([]byte(`{"schemaVersion":"douyin_draft_v1"}`)), Status: operationtask.OperationTaskStatusPendingReview,
		Priority: operationtask.OperationTaskPriorityNormal, Revision: 1,
	}
	require.NoError(t, operationtask.NewOperationTaskRepository(db).Create(ctx, &task))
	draft := operationtask.PlatformDraft{
		TenantID: tenantID, OperationTaskID: task.ID, Platform: operationtask.PlatformDouyin, AdapterMode: operationtask.AdapterModeProductionDraft,
		DraftVersion: 1, Payload: datatypes.JSON(frozenRaw), PayloadHash: payloadHash, Status: operationtask.PlatformDraftStatusPendingReview,
	}
	require.NoError(t, operationtask.NewPlatformDraftRepository(db).CreateVersion(ctx, &draft))

	detail, err := operationtask.NewAPIService(db).GetTask(ctx, operationtask.APIActor{TenantID: tenantID, ActorID: actorID, Role: admin.RoleAdmin}, task.ID)
	require.NoError(t, err)
	require.Equal(t, operationtask.AdapterModeProductionDraft, detail.LatestDraft.AdapterMode)
	encoded, err := json.Marshal(detail.LatestDraft.Payload)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "default_mobile")
	require.NotContains(t, string(encoded), "13800000000")
	require.Contains(t, string(encoded), "mappingSnapshot")
}

func TestProductionExecutionOutboxAndUnknownResultLifecycle(t *testing.T) {
	db := openOperationTaskTestDB(t)
	ctx := context.Background()
	tenantID := int64(101)
	actorID := createAdminUser(t, db, tenantID, admin.RoleAdmin, admin.StatusActive)
	productID := uuid.New()
	shopID := uuid.New()
	downstreamID := uuid.New()
	frozenRaw, err := json.Marshal(operationtask.FrozenDouyinDraft{
		SchemaVersion:   operationtask.DouyinDraftSchemaVersion,
		ProductID:       productID,
		ShopID:          shopID,
		PublishMode:     operationtask.DouyinDraftPublishMode,
		SKUCount:        1,
		Request:         json.RawMessage(`{"name":"Frozen title","specPricesV2":[{"price":100}]}`),
		Review:          json.RawMessage(`{"title":"Frozen title"}`),
		MappingSnapshot: json.RawMessage(`{"platform":"douyin_shop","productId":"` + productID.String() + `","shopId":"` + shopID.String() + `"}`),
		MappingHash:     hash1,
	})
	require.NoError(t, err)
	frozenHash, err := operationtask.ComputePayloadHash(frozenRaw)
	require.NoError(t, err)
	task := operationtask.OperationTask{
		TenantID: tenantID, SourceType: operationtask.OperationTaskSourceManual, SourceReference: productID.String(),
		TaskType: operationtask.OperationTaskTypeProductPublish, Platform: operationtask.PlatformDouyin, Title: "Create reviewed Douyin draft",
		Payload: datatypes.JSON([]byte(`{"schemaVersion":"douyin_draft_v1"}`)), Status: operationtask.OperationTaskStatusApproved,
		Priority: operationtask.OperationTaskPriorityNormal, Revision: 1,
	}
	require.NoError(t, operationtask.NewOperationTaskRepository(db).Create(ctx, &task))
	draft := operationtask.PlatformDraft{
		TenantID: tenantID, OperationTaskID: task.ID, Platform: operationtask.PlatformDouyin, AdapterMode: operationtask.AdapterModeProductionDraft,
		DraftVersion: 1, Payload: datatypes.JSON(frozenRaw), PayloadHash: frozenHash, Status: operationtask.PlatformDraftStatusApproved,
	}
	require.NoError(t, operationtask.NewPlatformDraftRepository(db).CreateVersion(ctx, &draft))
	approval := operationtask.ApprovalRecord{
		TenantID: tenantID, OperationTaskID: task.ID, PlatformDraftID: draft.ID, Decision: operationtask.ApprovalDecisionApproved,
		DraftVersion: 1, DraftPayloadHash: frozenHash, ReviewerID: uuid.New(), ReviewerRole: operationtask.ReviewerRoleAdmin,
	}
	require.NoError(t, operationtask.NewApprovalRecordRepository(db).CreateDecision(ctx, &approval))

	writeGuard := &allowProductionWrite{}
	now := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)
	svc := &operationtask.ProductionExecutionService{
		DB: db, Authorizer: allowExecutionAuthorizer{}, WriteGuard: writeGuard,
		Factory: fakeProductionFactory{taskID: downstreamID}, Now: func() time.Time { return now },
	}
	apiSvc := operationtask.NewAPIService(db)
	apiSvc.Production = svc
	_, err = apiSvc.Execute(ctx, operationtask.APIActor{TenantID: tenantID, ActorID: actorID, Role: admin.RoleAdmin}, task.ID, operationtask.ExecuteRequest{
		ExpectedTaskRevision: task.Revision, AdapterMode: operationtask.AdapterModeLocalDraftOnly,
	}, "req-wrong-production-mode", "idem-wrong-production-mode")
	require.ErrorIs(t, err, operationtask.ErrExecutionModeForbidden)
	require.Zero(t, writeGuard.calls)
	out, err := svc.Queue(ctx, operationtask.ExecutionInput{
		TenantID: tenantID, OperationTaskID: task.ID, ActorID: actorID, RequestID: "req-production",
		IdempotencyKey: "idem-production", AdapterMode: operationtask.AdapterModeProductionDraft,
	}, false)
	require.NoError(t, err)
	require.Equal(t, operationtask.ExecutionAttemptStatusQueued, out.Attempt.Status)
	require.Equal(t, 1, writeGuard.calls)
	require.Equal(t, downstreamID, *out.Attempt.DownstreamTaskID)

	var outbox operationtask.ExecutionOutbox
	require.NoError(t, db.First(&outbox, "execution_attempt_id = ?", out.Attempt.ID).Error)
	require.Equal(t, operationtask.ExecutionOutboxStatusPending, outbox.Status)

	delivery := &recordingOutboxDelivery{err: errors.New("redis unavailable")}
	dispatcher := &operationtask.OutboxDispatcher{DB: db, Delivery: delivery, Now: func() time.Time { return now }}
	count, err := dispatcher.DispatchPending(ctx, 50)
	require.NoError(t, err)
	require.Zero(t, count)
	now = now.Add(5 * time.Second)
	delivery.err = nil
	count, err = dispatcher.DispatchPending(ctx, 50)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Equal(t, 2, delivery.calls)
	require.NoError(t, db.First(&outbox, "id = ?", outbox.ID).Error)
	require.Equal(t, operationtask.ExecutionOutboxStatusPending, outbox.Status)

	resultSvc := &operationtask.ProductionResultService{DB: db, Now: func() time.Time { return now }}
	require.NoError(t, resultSvc.MarkRunning(ctx, out.Attempt.ID, downstreamID))
	require.NoError(t, db.First(&outbox, "id = ?", outbox.ID).Error)
	require.Equal(t, operationtask.ExecutionOutboxStatusPending, outbox.Status)
	require.NoError(t, resultSvc.MarkFailed(ctx, out.Attempt.ID, downstreamID, "DOUYIN_RESULT_UNKNOWN", "平台结果待对账", true, true, datatypes.JSON([]byte(`{"recoveryStatus":"result_unknown"}`))))
	require.NoError(t, db.First(&outbox, "id = ?", outbox.ID).Error)
	require.Equal(t, operationtask.ExecutionOutboxStatusDelivered, outbox.Status)

	var attempt operationtask.ExecutionAttempt
	require.NoError(t, db.First(&attempt, "id = ?", out.Attempt.ID).Error)
	require.Equal(t, operationtask.ExecutionAttemptStatusResultUnknown, attempt.Status)
	require.Equal(t, "result_unknown", attempt.ResultType)
	updatedTask, err := operationtask.NewOperationTaskRepository(db).GetByID(ctx, tenantID, task.ID)
	require.NoError(t, err)
	require.Equal(t, operationtask.OperationTaskStatusResultUnknown, updatedTask.Status)
	failure, err := operationtask.NewExecutionErrorRepository(db).GetLatestByAttempt(ctx, tenantID, attempt.ID)
	require.NoError(t, err)
	require.False(t, failure.Retryable)

	// A result_unknown attempt can only converge to success after the recovery
	// path confirms the already-created platform draft. It is never re-queued.
	require.NoError(t, resultSvc.MarkSucceeded(ctx, out.Attempt.ID, downstreamID, "platform-draft-recovered", "req-recovery", datatypes.JSON([]byte(`{"recoveryStatus":"recovered"}`))))
	require.NoError(t, db.First(&attempt, "id = ?", out.Attempt.ID).Error)
	require.Equal(t, operationtask.ExecutionAttemptStatusSucceeded, attempt.Status)
	require.Equal(t, "platform_draft", attempt.ResultType)
	updatedTask, err = operationtask.NewOperationTaskRepository(db).GetByID(ctx, tenantID, task.ID)
	require.NoError(t, err)
	require.Equal(t, operationtask.OperationTaskStatusDraftWritten, updatedTask.Status)
}
