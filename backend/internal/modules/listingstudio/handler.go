package listingstudio

import (
	"context"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/modules/idempotency"
	"github.com/trademind-ai/trademind/backend/internal/modules/operationlog"
	"github.com/trademind-ai/trademind/backend/internal/pkg/adminperm"
	"github.com/trademind-ai/trademind/backend/internal/pkg/ctxkey"
	"github.com/trademind-ai/trademind/backend/internal/pkg/response"
)

type Handler struct {
	Svc         *Service
	OpLog       *operationlog.Service
	Idempotency *idempotency.Service
}

func (h *Handler) actor(c *gin.Context) (int64, *uuid.UUID, bool) {
	w, err := adminperm.TenantIDFromGin(c)
	if err != nil {
		response.Fail(c, 401, response.CodeUnauthorized, "workspace unavailable")
		return 0, nil, false
	}
	var aid *uuid.UUID
	if raw, ok := c.Get(ctxkey.AdminID); ok {
		if s, ok := raw.(string); ok {
			if parsed, e := uuid.Parse(s); e == nil {
				aid = &parsed
			}
		}
	}
	return w, aid, true
}
func draftID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.Fail(c, 400, response.CodeBadRequest, "invalid draft id")
		return uuid.Nil, false
	}
	return id, true
}
func (h *Handler) Create(c *gin.Context) {
	w, _, ok := h.actor(c)
	if !ok {
		return
	}
	var b CreateDraftInput
	if c.ShouldBindJSON(&b) != nil {
		response.Fail(c, 400, response.CodeBadRequest, "invalid json body")
		return
	}
	idempotency.ExecuteHTTPCommand(c, h.Idempotency, w, "listing-drafts", b, func(ctx context.Context) (any, string, error) {
		out, err := h.Svc.Create(ctx, w, b)
		if err != nil {
			return nil, "", err
		}
		return out, out.ID.String(), nil
	})
}
func (h *Handler) Validate(c *gin.Context) {
	w, aid, ok := h.actor(c)
	if !ok {
		return
	}
	id, ok := draftID(c)
	if !ok {
		return
	}
	var b ValidationInput
	if c.Request.ContentLength > 0 && c.ShouldBindJSON(&b) != nil {
		response.Fail(c, 400, response.CodeBadRequest, "invalid json body")
		return
	}
	b.AdminID = aid
	idempotency.ExecuteHTTPCommand(c, h.Idempotency, w, "listing-drafts.validate:"+id.String(), b, func(ctx context.Context) (any, string, error) {
		out, err := h.Svc.Validate(ctx, w, id, b)
		if err != nil {
			return nil, "", err
		}
		if h.OpLog != nil {
			status := "success"
			if out.State == StateBlocked {
				status = "failed"
			}
			_ = h.OpLog.Write(c, operationlog.WriteOpts{TenantID: w, AdminUserID: aid, Action: "mindbay.listing.validate", Resource: "listing_draft", ResourceID: id.String(), Status: status, Message: strings.Join(out.Errors, "; ")})
			if out.GPSROverridden {
				_ = h.OpLog.Write(c, operationlog.WriteOpts{TenantID: w, AdminUserID: aid, Action: "mindbay.gpsr.override", Resource: "listing_draft", ResourceID: id.String(), Status: "success", Message: b.OverrideReason})
			}
		}
		return out, id.String(), nil
	})
}
func (h *Handler) Generate(c *gin.Context) {
	w, _, ok := h.actor(c)
	if !ok {
		return
	}
	id, ok := draftID(c)
	if !ok {
		return
	}
	idempotency.ExecuteHTTPCommand(c, h.Idempotency, w, "listing-drafts.generate:"+id.String(), gin.H{"id": id.String()}, func(ctx context.Context) (any, string, error) {
		out, err := h.Svc.Generate(ctx, w, id)
		if err != nil {
			return nil, "", err
		}
		return out, out.ID.String(), nil
	})
}
func (h *Handler) Get(c *gin.Context) {
	w, _, ok := h.actor(c)
	if !ok {
		return
	}
	id, ok := draftID(c)
	if !ok {
		return
	}
	d, v, err := h.Svc.Get(c, w, id)
	if err != nil {
		response.Fail(c, 404, response.CodeNotFound, "not found")
		return
	}
	response.OK(c, gin.H{"draft": d, "versions": v})
}
func (h *Handler) List(c *gin.Context) {
	w, _, ok := h.actor(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	out, err := h.Svc.List(c, w, DraftState(strings.TrimSpace(c.Query("state"))), limit, c.Query("cursor"))
	if err != nil {
		response.Fail(c, 400, response.CodeBadRequest, err.Error())
		return
	}
	response.OK(c, out)
}
func (h *Handler) CreateGPSRProfile(c *gin.Context) {
	w, _, ok := h.actor(c)
	if !ok {
		return
	}
	var b CreateGPSRProfileInput
	if c.ShouldBindJSON(&b) != nil {
		response.Fail(c, 400, response.CodeBadRequest, "invalid GPSR profile")
		return
	}
	idempotency.ExecuteHTTPCommand(c, h.Idempotency, w, "gpsr-profiles", b, func(ctx context.Context) (any, string, error) {
		out, err := h.Svc.CreateGPSRProfile(ctx, w, b)
		if err != nil {
			return nil, "", err
		}
		return out, out.ID.String(), nil
	})
}
func (h *Handler) ListGPSRProfiles(c *gin.Context) {
	w, _, ok := h.actor(c)
	if !ok {
		return
	}
	out, err := h.Svc.ListGPSRProfiles(c, w)
	if err != nil {
		response.Fail(c, 500, response.CodeInternalError, "GPSR profiles unavailable")
		return
	}
	response.OK(c, gin.H{"items": out})
}
func Register(r gin.IRouter, h *Handler, image ...*ImageHandler) {
	r.POST("/listing-drafts", h.Create)
	r.GET("/listing-drafts", h.List)
	r.GET("/listing-drafts/:id", h.Get)
	r.POST("/listing-drafts/:id/validate", h.Validate)
	r.POST("/listing-drafts/:id/generate", h.Generate)
	r.GET("/gpsr-profiles", h.ListGPSRProfiles)
	r.POST("/gpsr-profiles", h.CreateGPSRProfile)
	if len(image) > 0 && image[0] != nil {
		r.POST("/image-assets", image[0].Create)
	}
}
