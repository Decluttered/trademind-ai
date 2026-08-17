package publication

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/modules/idempotency"
	"github.com/trademind-ai/trademind/backend/internal/pkg/adminperm"
	"github.com/trademind-ai/trademind/backend/internal/pkg/ctxkey"
	"github.com/trademind-ai/trademind/backend/internal/pkg/response"
)

type Handler struct {
	Svc         *Service
	PublishSvc  *PublishService
	Idempotency *idempotency.Service
}

func (h *Handler) Approve(c *gin.Context) {
	if h == nil || h.Svc == nil || !adminperm.RequireWrite(c, h.Svc.DB, adminperm.PermProductWrite) {
		return
	}
	w, ok := workspace(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.Fail(c, 400, response.CodeBadRequest, "invalid publication id")
		return
	}
	if h.PublishSvc == nil {
		response.Fail(c, 503, response.CodeInternalError, "publication worker unavailable")
		return
	}
	var job PublicationJob
	if err := h.Svc.DB.WithContext(c).Where("id=? AND workspace_id=?", id, w).First(&job).Error; err != nil {
		response.Fail(c, 404, response.CodeNotFound, "publication not found")
		return
	}
	if !adminperm.RequireStoreOperate(c, h.Svc.DB, job.ShopID) {
		return
	}
	idempotency.ExecuteHTTPCommand(c, h.Idempotency, w, "publications.approve:"+id.String(), gin.H{"id": id.String()}, func(ctx context.Context) (any, string, error) {
		if err := h.PublishSvc.Revalidate(ctx, w, id); err != nil {
			return nil, "", err
		}
		out, err := h.PublishSvc.Publish(ctx, w, id)
		return gin.H{"listing": out}, id.String(), err
	})
}

func workspace(c *gin.Context) (int64, bool) {
	w, err := adminperm.TenantIDFromGin(c)
	if err != nil {
		response.Fail(c, 401, response.CodeUnauthorized, "workspace unavailable")
		return 0, false
	}
	return w, true
}

func (h *Handler) Preview(c *gin.Context) {
	if h == nil || h.Svc == nil || !adminperm.RequirePermission(c, h.Svc.DB, adminperm.PermProductView) {
		return
	}
	w, ok := workspace(c)
	if !ok {
		return
	}
	var in PreviewInput
	if c.ShouldBindJSON(&in) != nil {
		response.Fail(c, 400, response.CodeBadRequest, "invalid planner preview")
		return
	}
	out, err := h.Svc.Preview(c, w, in)
	if err != nil {
		response.Fail(c, 400, response.CodeBadRequest, err.Error())
		return
	}
	response.OK(c, out)
}

func (h *Handler) Apply(c *gin.Context) {
	if h == nil || h.Svc == nil || !adminperm.RequireWrite(c, h.Svc.DB, adminperm.PermProductWrite) {
		return
	}
	w, ok := workspace(c)
	if !ok {
		return
	}
	var in ApplyInput
	if c.ShouldBindJSON(&in) != nil {
		response.Fail(c, 400, response.CodeBadRequest, "invalid planner apply")
		return
	}
	if actor, exists := c.Get(ctxkey.AdminID); exists {
		in.Actor = "admin:" + fmt.Sprint(actor)
	}
	if correlationID, exists := c.Get(ctxkey.TraceID); exists {
		in.CorrelationID = fmt.Sprint(correlationID)
	}
	if !adminperm.RequireStoreOperate(c, h.Svc.DB, in.ShopID) {
		return
	}
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	idempotency.ExecuteHTTPCommand(c, h.Idempotency, w, "calendar.apply", in, func(ctx context.Context) (any, string, error) {
		out, err := h.Svc.Apply(ctx, w, key, in)
		if err != nil {
			return nil, "", err
		}
		resourceID := ""
		if len(out.Slots) > 0 {
			resourceID = out.Slots[0].ID.String()
		}
		return out, resourceID, nil
	})
}

func (h *Handler) List(c *gin.Context) {
	if h == nil || h.Svc == nil || !adminperm.RequirePermission(c, h.Svc.DB, adminperm.PermProductView) {
		return
	}
	w, ok := workspace(c)
	if !ok {
		return
	}
	parse := func(raw string) time.Time {
		value, _ := time.Parse(time.RFC3339, strings.TrimSpace(raw))
		return value
	}
	rows, err := h.Svc.ListSlots(c, w, parse(c.Query("from")), parse(c.Query("to")))
	if err != nil {
		response.Fail(c, 500, response.CodeInternalError, "calendar unavailable")
		return
	}
	response.OK(c, gin.H{"items": rows})
}

func Register(r gin.IRouter, h *Handler) {
	r.GET("/calendar/slots", h.List)
	r.POST("/calendar/preview", h.Preview)
	r.POST("/calendar/apply", h.Apply)
	r.POST("/publications/:id/approve", h.Approve)
}
