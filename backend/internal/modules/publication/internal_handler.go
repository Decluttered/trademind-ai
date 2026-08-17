package publication

import (
	"crypto/subtle"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/pkg/response"
)

type InternalHandler struct {
	Svc   *PublishService
	Token string
}

type internalCommand struct {
	WorkspaceID int64 `json:"workspaceId"`
}

func (h *InternalHandler) authorize(c *gin.Context) bool {
	expected := strings.TrimSpace(h.Token)
	provided := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
	if expected == "" || len(expected) != len(provided) || subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
		response.Fail(c, 401, response.CodeUnauthorized, "internal service authentication failed")
		return false
	}
	return true
}

func (h *InternalHandler) command(c *gin.Context) (internalCommand, uuid.UUID, bool) {
	if !h.authorize(c) {
		return internalCommand{}, uuid.Nil, false
	}
	var body internalCommand
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil || c.ShouldBindJSON(&body) != nil || body.WorkspaceID <= 0 {
		response.Fail(c, 400, response.CodeBadRequest, "invalid internal publication command")
		return internalCommand{}, uuid.Nil, false
	}
	return body, id, true
}

func (h *InternalHandler) Revalidate(c *gin.Context) {
	body, id, ok := h.command(c)
	if !ok {
		return
	}
	if err := h.Svc.Revalidate(c, body.WorkspaceID, id); err != nil {
		response.Fail(c, 409, response.CodeBadRequest, err.Error())
		return
	}
	response.OK(c, gin.H{"valid": true})
}

func (h *InternalHandler) Publish(c *gin.Context) {
	body, id, ok := h.command(c)
	if !ok {
		return
	}
	listing, err := h.Svc.Publish(c, body.WorkspaceID, id)
	if err != nil {
		response.Fail(c, 502, response.CodeInternalError, err.Error())
		return
	}
	response.OK(c, gin.H{"listing": listing})
}

func (h *InternalHandler) Reconcile(c *gin.Context) {
	body, id, ok := h.command(c)
	if !ok {
		return
	}
	listing, err := h.Svc.Reconcile(c, body.WorkspaceID, id)
	if err != nil {
		response.Fail(c, 409, response.CodeBadRequest, err.Error())
		return
	}
	response.OK(c, gin.H{"listing": listing})
}

func RegisterInternal(r gin.IRouter, h *InternalHandler) {
	r.POST("/publications/:id/revalidate", h.Revalidate)
	r.POST("/publications/:id/publish", h.Publish)
	r.POST("/publications/:id/reconcile", h.Reconcile)
}
