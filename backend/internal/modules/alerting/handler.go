package alerting

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/modules/operationlog"
	"github.com/trademind-ai/trademind/backend/internal/pkg/adminperm"
	"github.com/trademind-ai/trademind/backend/internal/pkg/ctxkey"
	"github.com/trademind-ai/trademind/backend/internal/pkg/response"
	"gorm.io/gorm"
)

// Handler exposes alert management APIs.
type Handler struct {
	Svc   *Service
	OpLog *operationlog.Service
	DB    interface { /* gorm */
	}
}

type alertListItem struct {
	ID              string    `json:"id"`
	RuleID          string    `json:"ruleId"`
	Severity        string    `json:"severity"`
	Status          string    `json:"status"`
	Module          string    `json:"module"`
	Summary         string    `json:"summary"`
	OccurrenceCount int       `json:"occurrenceCount"`
	LastSeenAt      time.Time `json:"lastSeenAt"`
}

func newAlertListItem(row AlertEvent) alertListItem {
	return alertListItem{
		ID:              row.ID,
		RuleID:          row.RuleID,
		Severity:        row.Severity,
		Status:          row.Status,
		Module:          row.Module,
		Summary:         row.Summary,
		OccurrenceCount: row.OccurrenceCount,
		LastSeenAt:      row.LastSeenAt,
	}
}

// Register mounts alerting routes under authed group.
func Register(r gin.IRouter, h *Handler) {
	if r == nil || h == nil || h.Svc == nil {
		return
	}
	g := r.Group("/observability/alerts")
	g.GET("", h.List)
	g.POST("/:id/ack", h.Ack)
	g.POST("/:id/silence", h.Silence)
}

// List returns recent alerts.
func (h *Handler) List(c *gin.Context) {
	if !adminperm.RequirePermission(c, h.Svc.DB, adminperm.PermAlertsRead) {
		return
	}
	page := positiveQueryInt(c.Query("page"), 1)
	pageSizeRaw := c.Query("pageSize")
	if pageSizeRaw == "" {
		pageSizeRaw = c.DefaultQuery("limit", "50")
	}
	pageSize := positiveQueryInt(pageSizeRaw, 50)
	if pageSize > 200 {
		pageSize = 200
	}
	var rows []AlertEvent
	q := h.Svc.DB.Model(&AlertEvent{})
	if st := strings.TrimSpace(c.Query("status")); st != "" {
		q = q.Where("status = ?", st)
	}
	if severity := strings.TrimSpace(c.Query("severity")); severity != "" {
		q = q.Where("severity = ?", severity)
	}
	if module := strings.TrimSpace(c.Query("module")); module != "" {
		q = q.Where("module = ?", module)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalError, err.Error())
		return
	}
	if err := q.Order("last_seen_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalError, err.Error())
		return
	}
	items := make([]alertListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, newAlertListItem(row))
	}
	totalPages := total / int64(pageSize)
	if total%int64(pageSize) != 0 {
		totalPages++
	}
	response.OK(c, gin.H{
		"items": items,
		"pagination": gin.H{
			"page":       page,
			"pageSize":   pageSize,
			"total":      total,
			"totalPages": totalPages,
		},
	})
}

// Ack acknowledges an alert.
func (h *Handler) Ack(c *gin.Context) {
	if !adminperm.RequireWrite(c, h.Svc.DB, adminperm.PermAlertsAck) {
		return
	}
	id := c.Param("id")
	if err := h.Svc.Acknowledge(c.Request.Context(), id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, http.StatusNotFound, response.CodeNotFound, "alert not found")
			return
		}
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
		return
	}
	h.writeAudit(c, "alert.acknowledge", id, adminperm.PermAlertsAck, "acknowledged")
	response.OK(c, gin.H{"id": id, "status": StatusAcknowledged})
}

type silenceReq struct {
	Reason        string `json:"reason"`
	DurationHours int    `json:"durationHours"`
}

// Silence silences an alert.
func (h *Handler) Silence(c *gin.Context) {
	if !adminperm.RequireWrite(c, h.Svc.DB, adminperm.PermAlertsSilence) {
		return
	}
	id := c.Param("id")
	var req silenceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" || len([]rune(reason)) > 256 {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "reason is required and must not exceed 256 characters")
		return
	}
	if req.DurationHours < 1 || req.DurationHours > 720 {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "durationHours must be between 1 and 720")
		return
	}
	by, _ := c.Get(ctxkey.AdminID)
	byStr, _ := by.(string)
	if byStr == "" {
		byStr = "system"
	}
	until := time.Now().UTC().Add(time.Duration(req.DurationHours) * time.Hour)
	if err := h.Svc.Silence(c.Request.Context(), id, reason, byStr, until); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, http.StatusNotFound, response.CodeNotFound, "alert not found")
			return
		}
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
		return
	}
	h.writeAudit(c, "alert.silence", id, adminperm.PermAlertsSilence, "silenced")
	response.OK(c, gin.H{"id": id, "status": StatusSilenced, "expiresAt": until})
}

func (h *Handler) writeAudit(c *gin.Context, action, alertID, permission, message string) {
	if h == nil || h.OpLog == nil {
		return
	}
	_ = h.OpLog.Write(c, operationlog.WriteOpts{
		Action:     action,
		Resource:   "alert_event",
		ResourceID: alertID,
		Permission: permission,
		Status:     "success",
		Message:    message,
	})
}

func positiveQueryInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func newUUID() string {
	return uuid.New().String()
}
