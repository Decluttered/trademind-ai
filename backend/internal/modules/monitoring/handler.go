package monitoring

import (
	"context"
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
	Idempotency *idempotency.Service
}

func (h *Handler) workspace(c *gin.Context, write bool) (int64, bool) {
	if h == nil || h.Svc == nil || h.Svc.DB == nil {
		response.Fail(c, 503, response.CodeInternalError, "monitoring unavailable")
		return 0, false
	}
	if write {
		if !adminperm.RequireWrite(c, h.Svc.DB, adminperm.PermProductWrite) {
			return 0, false
		}
	} else if !adminperm.RequirePermission(c, h.Svc.DB, adminperm.PermProductView) {
		return 0, false
	}
	w, err := adminperm.TenantIDFromGin(c)
	if err != nil {
		response.Fail(c, 401, response.CodeUnauthorized, "workspace unavailable")
		return 0, false
	}
	return w, true
}

func (h *Handler) CreateRule(c *gin.Context) {
	w, ok := h.workspace(c, true)
	if !ok {
		return
	}
	var in CreateRuleInput
	if c.ShouldBindJSON(&in) != nil {
		response.Fail(c, 400, response.CodeBadRequest, "invalid price rule")
		return
	}
	idempotency.ExecuteHTTPCommand(c, h.Idempotency, w, "mindbay.price-rules.create", in, func(ctx context.Context) (any, string, error) {
		row, err := h.Svc.CreateRule(ctx, w, in)
		if err != nil {
			return nil, "", err
		}
		return row, row.ID.String(), nil
	})
}

func (h *Handler) ListRules(c *gin.Context) {
	w, ok := h.workspace(c, false)
	if !ok {
		return
	}
	rows, err := h.Svc.ListRules(c, w)
	if err != nil {
		response.Fail(c, 500, response.CodeInternalError, "price rules unavailable")
		return
	}
	response.OK(c, gin.H{"items": rows})
}
func (h *Handler) ListMonitorable(c *gin.Context) {
	w, ok := h.workspace(c, false)
	if !ok {
		return
	}
	rows, err := h.Svc.ListMonitorable(c, w)
	if err != nil {
		response.Fail(c, 500, response.CodeInternalError, "monitorable listings unavailable")
		return
	}
	response.OK(c, gin.H{"items": rows})
}

func (h *Handler) Run(c *gin.Context) {
	w, ok := h.workspace(c, true)
	if !ok {
		return
	}
	var in RunInput
	if c.ShouldBindJSON(&in) != nil {
		response.Fail(c, 400, response.CodeBadRequest, "invalid monitor run")
		return
	}
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	correlation := key
	if value, exists := c.Get(ctxkey.TraceID); exists {
		correlation = strings.TrimSpace(value.(string))
	}
	idempotency.ExecuteHTTPCommand(c, h.Idempotency, w, "mindbay.monitor-runs.create", in, func(ctx context.Context) (any, string, error) {
		out, err := h.Svc.Run(ctx, w, key, correlation, in)
		if err != nil {
			return nil, "", err
		}
		return out, out.Run.ID.String(), nil
	})
}

func (h *Handler) ListDecisions(c *gin.Context) {
	w, ok := h.workspace(c, false)
	if !ok {
		return
	}
	rows, err := h.Svc.ListDecisions(c, w, c.Query("outcome"))
	if err != nil {
		response.Fail(c, 500, response.CodeInternalError, "price decisions unavailable")
		return
	}
	response.OK(c, gin.H{"items": rows})
}

func (h *Handler) Apply(c *gin.Context) {
	w, ok := h.workspace(c, true)
	if !ok {
		return
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.Fail(c, 400, response.CodeBadRequest, "invalid price decision id")
		return
	}
	idempotency.ExecuteHTTPCommand(c, h.Idempotency, w, "mindbay.price-decisions.apply:"+id.String(), gin.H{"id": id}, func(ctx context.Context) (any, string, error) {
		row, applyErr := h.Svc.Apply(ctx, w, id)
		if applyErr != nil {
			return nil, "", applyErr
		}
		return row, row.ID.String(), nil
	})
}

func (h *Handler) Profit(c *gin.Context) {
	w, ok := h.workspace(c, false)
	if !ok {
		return
	}
	parse := func(value string) time.Time { row, _ := time.Parse(time.RFC3339, strings.TrimSpace(value)); return row }
	report, err := h.Svc.Profit(c, w, parse(c.Query("from")), parse(c.Query("to")))
	if err != nil {
		response.Fail(c, 500, response.CodeInternalError, "profit report unavailable")
		return
	}
	response.OK(c, report)
}

func Register(r gin.IRouter, h *Handler) {
	r.GET("/monitorable-listings", h.ListMonitorable)
	r.POST("/monitor-runs", h.Run)
	r.GET("/price-rules", h.ListRules)
	r.POST("/price-rules", h.CreateRule)
	r.GET("/price-decisions", h.ListDecisions)
	r.POST("/price-decisions/:id/apply", h.Apply)
	r.GET("/profit/report", h.Profit)
}
