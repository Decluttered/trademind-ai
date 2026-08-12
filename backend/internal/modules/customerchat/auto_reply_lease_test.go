package customerchat

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/pkg/model"
	"gorm.io/gorm"
)

func openAutoReplyLeaseTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:auto_reply_lease_%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&CustomerConversation{}, &CustomerAutoReplyRun{}, &CustomerFailureEvent{}))
	return db
}

func newPendingAutoReplyRun(t *testing.T, db *gorm.DB) CustomerAutoReplyRun {
	t.Helper()
	run := CustomerAutoReplyRun{
		Base: model.Base{ID: uuid.New()}, TenantID: 7, ShopID: uuid.New(),
		ConversationID: uuid.New(), MessageID: uuid.New(), Status: AutoReplyRunPending,
	}
	require.NoError(t, db.Create(&run).Error)
	return run
}

func TestAutoReplyLeaseClaimHasOneWinner(t *testing.T) {
	db := openAutoReplyLeaseTestDB(t)
	svc := &Service{DB: db}
	run := newPendingAutoReplyRun(t, db)

	_, _, firstClaimed, err := svc.claimAutoReplyRun(context.Background(), run.ID, "worker-a")
	require.NoError(t, err)
	require.True(t, firstClaimed)
	_, _, secondClaimed, err := svc.claimAutoReplyRun(context.Background(), run.ID, "worker-b")
	require.NoError(t, err)
	require.False(t, secondClaimed)
}

func TestExpiredGeneratingRunReturnsToPending(t *testing.T) {
	db := openAutoReplyLeaseTestDB(t)
	svc := &Service{DB: db}
	run := newPendingAutoReplyRun(t, db)
	past := time.Now().UTC().Add(-time.Minute)
	require.NoError(t, db.Model(&run).Updates(map[string]any{
		"status": AutoReplyRunGenerating, "locked_by": "dead-worker", "locked_until": &past,
		"execution_id": uuid.NewString(), "lock_version": 1,
	}).Error)

	require.NoError(t, svc.recoverExpiredAutoReplyRun(context.Background(), run.ID))
	var recovered CustomerAutoReplyRun
	require.NoError(t, db.First(&recovered, "id = ?", run.ID).Error)
	require.Equal(t, AutoReplyRunPending, recovered.Status)
	require.Nil(t, recovered.LockedBy)
	require.Nil(t, recovered.LockedUntil)
}

func TestExpiredSendingRunRequiresManualReview(t *testing.T) {
	db := openAutoReplyLeaseTestDB(t)
	svc := &Service{DB: db}
	run := newPendingAutoReplyRun(t, db)
	shopID := run.ShopID
	conversation := CustomerConversation{
		Base: model.Base{ID: run.ConversationID}, TenantID: run.TenantID, Platform: "mock", ShopID: &shopID,
		CustomerName: "buyer", CustomerLanguage: "en", Status: StatusPendingReply,
	}
	require.NoError(t, db.Create(&conversation).Error)
	past := time.Now().UTC().Add(-time.Minute)
	require.NoError(t, db.Model(&run).Updates(map[string]any{
		"status": AutoReplyRunSending, "locked_by": "dead-worker", "locked_until": &past,
		"execution_id": uuid.NewString(), "lock_version": 1,
	}).Error)

	require.NoError(t, svc.recoverExpiredAutoReplyRun(context.Background(), run.ID))
	require.NoError(t, db.First(&run, "id = ?", run.ID).Error)
	require.Equal(t, AutoReplyRunHumanRequired, run.Status)
	require.Equal(t, "platform_send_result_unknown", run.ReasonCode)
	require.NotNil(t, run.FinishedAt)
	var failure CustomerFailureEvent
	require.NoError(t, db.Where("conversation_id = ? AND category = ? AND status = ?", run.ConversationID, FailureCategoryReplySendFailed, FailureEventStatusOpen).First(&failure).Error)
	require.Equal(t, "platform send result unknown; manual review required", failure.ErrorMessage)
}
