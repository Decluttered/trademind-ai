package productpublish

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/trademind-ai/trademind/backend/internal/modules/operationlog"
	"github.com/trademind-ai/trademind/backend/internal/pkg/adminperm"
	"github.com/trademind-ai/trademind/backend/internal/pkg/repository"
	"gorm.io/gorm"
)

// CancelTask marks a pending/running publish task as cancelled.
func (s *Service) CancelTask(c *gin.Context, taskID uuid.UUID, adminID *uuid.UUID) (*TaskDTO, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("productpublish: no db")
	}
	tid, err := adminperm.TenantIDFromGin(c)
	if err != nil {
		return nil, err
	}
	var task ProductPublishTask
	if err := repository.FindByID(c.Request.Context(), s.DB, &task, tid, taskID); err != nil {
		return nil, err
	}
	principal, err := adminperm.LoadPrincipal(c, s.DB)
	if err != nil {
		return nil, err
	}
	if principal == nil || !principal.CanOperateStore(task.ShopID) {
		return nil, gorm.ErrRecordNotFound
	}
	st := strings.TrimSpace(task.Status)
	if st != TaskPending && st != TaskRunning {
		return nil, fmt.Errorf("only pending or running tasks can be cancelled")
	}
	fin := time.Now().UTC()
	if err := s.DB.WithContext(c.Request.Context()).Model(&ProductPublishTask{}).Where("id = ? AND tenant_id = ?", taskID, tid).
		Updates(map[string]any{
			"status":         TaskCancelled,
			"publish_status": TaskCancelled,
			"finished_at":    &fin,
			"locked_by":      nil,
			"locked_until":   nil,
			"updated_at":     fin,
		}).Error; err != nil {
		return nil, err
	}
	if snap, ok := parseDouyinDraftSnapshot(task.Input); ok {
		_ = s.DB.WithContext(c.Request.Context()).Model(&ProductPublication{}).Where("id = ? AND tenant_id = ?", snap.PublicationID, tid).
			Updates(map[string]any{"status": TaskCancelled, "publish_status": TaskCancelled, "updated_at": fin}).Error
	} else if rid, ok := snapshotPublicationFromTask(&task); ok {
		_ = s.DB.WithContext(c.Request.Context()).Model(&ProductPublication{}).Where("id = ? AND tenant_id = ?", rid, tid).
			Updates(map[string]any{"status": TaskCancelled, "publish_status": TaskCancelled, "updated_at": fin}).Error
	}
	action := "product.publish.cancel"
	if task.Platform == "douyin_shop" {
		action = "douyin.product.publish_task.cancel"
	}
	if s.OpLog != nil {
		_ = s.OpLog.Write(c, operationlog.WriteOpts{
			AdminUserID: adminID,
			Action:      action,
			Resource:    "product_publish_task",
			ResourceID:  taskID.String(),
			Status:      "success",
			Message:     fmt.Sprintf("taskId=%s platform=%s", taskID, task.Platform),
		})
	}
	out, err := s.GetDTO(c.Request.Context(), tid, taskID)
	return &out, err
}
