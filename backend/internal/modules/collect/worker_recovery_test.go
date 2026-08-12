package collect

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func openCollectRecoveryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:collect_recovery_%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&CollectBatch{}, &CollectTask{}, &CollectTaskEvent{}))
	return db
}

func TestFailTaskMissingTenantPreservesStatusAndReconcilesBatch(t *testing.T) {
	for _, status := range []string{StatusPending, StatusRetrying} {
		t.Run(status, func(t *testing.T) {
			db := openCollectRecoveryTestDB(t)
			svc := &Service{DB: db}
			batch := CollectBatch{
				Source:       "1688",
				TotalCount:   1,
				PendingCount: 1,
				Status:       BatchStatusRunning,
			}
			require.NoError(t, db.Create(&batch).Error)
			task := CollectTask{
				TenantID:   0,
				BatchID:    &batch.ID,
				Source:     "1688",
				SourceURL:  "https://detail.1688.com/offer/1.html",
				Status:     status,
				MaxRetries: 3,
			}
			require.NoError(t, db.Create(&task).Error)

			require.NoError(t, svc.failTaskMissingTenant(context.Background(), &task))

			var stored CollectTask
			require.NoError(t, db.First(&stored, "id = ?", task.ID).Error)
			require.Equal(t, StatusFailed, stored.Status)
			require.NotEmpty(t, stored.ErrorMessage)
			require.NotNil(t, stored.FinishedAt)

			var events []CollectTaskEvent
			require.NoError(t, db.Where("task_id = ?", task.ID).Find(&events).Error)
			require.Len(t, events, 1)
			require.Equal(t, EventTaskFailed, events[0].EventType)
			require.NotNil(t, events[0].FromStatus)
			require.Equal(t, status, *events[0].FromStatus)
			require.NotNil(t, events[0].ToStatus)
			require.Equal(t, StatusFailed, *events[0].ToStatus)

			var storedBatch CollectBatch
			require.NoError(t, db.First(&storedBatch, "id = ?", batch.ID).Error)
			require.Equal(t, 0, storedBatch.PendingCount)
			require.Equal(t, 1, storedBatch.FailedCount)
			require.Equal(t, BatchStatusFailed, storedBatch.Status)

			require.NoError(t, svc.failTaskMissingTenant(context.Background(), &task))
			var eventCount int64
			require.NoError(t, db.Model(&CollectTaskEvent{}).Where("task_id = ?", task.ID).Count(&eventCount).Error)
			require.Equal(t, int64(1), eventCount)
		})
	}
}

func TestRecoverStalePendingKeysetIncludesIdenticalTimestamps(t *testing.T) {
	db := openCollectRecoveryTestDB(t)
	stamp := time.Now().UTC().Add(-2 * collectPendingCutoff)
	rows := make([]CollectTask, 0, collectRecoveryPageSize+1)
	for i := range collectRecoveryPageSize + 1 {
		id := uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1))
		rows = append(rows, CollectTask{
			TenantID:   1,
			Source:     "1688",
			SourceURL:  fmt.Sprintf("https://detail.1688.com/offer/%d.html", i+1),
			Status:     StatusPending,
			MaxRetries: 3,
		})
		rows[i].ID = id
		rows[i].CreatedAt = stamp
		rows[i].UpdatedAt = stamp
	}
	require.NoError(t, db.CreateInBatches(&rows, 50).Error)

	svc := &Service{DB: db}
	first, err := svc.stalePendingTasksPage(context.Background(), time.Now().UTC(), time.Time{}, uuid.Nil)
	require.NoError(t, err)
	require.Len(t, first, collectRecoveryPageSize)

	last := first[len(first)-1]
	second, err := svc.stalePendingTasksPage(context.Background(), time.Now().UTC(), last.CreatedAt, last.ID)
	require.NoError(t, err)
	require.Len(t, second, 1)
	require.Equal(t, rows[len(rows)-1].ID, second[0].ID)
}
