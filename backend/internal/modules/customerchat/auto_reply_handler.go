package customerchat

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/pkg/adminperm"
	"github.com/trademind-ai/trademind/backend/internal/pkg/response"
	"gorm.io/gorm"
)

func (h *Handler) GetAutoReplySetting(c *gin.Context) {
	if h == nil || h.Svc == nil {
		response.Fail(c, 500, response.CodeInternalError, "customer chat unavailable")
		return
	}
	out, err := h.Svc.GetAutoReplySetting(c)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) UpdateAutoReplySetting(c *gin.Context) {
	if h == nil || h.Svc == nil {
		response.Fail(c, 500, response.CodeInternalError, "customer chat unavailable")
		return
	}
	if !adminperm.CanManageSettings(c, h.Svc.DB) || !adminperm.CanWriteCustomer(c, h.Svc.DB) {
		response.Fail(c, 403, response.CodeForbidden, "无权管理 AI 自动回复设置")
		return
	}
	var body UpdateAutoReplySettingBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, 400, response.CodeBadRequest, "invalid json body")
		return
	}
	out, err := h.Svc.UpdateAutoReplySetting(c, body, adminUUID(c))
	if err != nil {
		response.Fail(c, 400, response.CodeBadRequest, err.Error())
		return
	}
	response.OK(c, out)
}

func autoReplyShopID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param("shopId")))
	if err != nil || id == uuid.Nil {
		response.Fail(c, 400, response.CodeBadRequest, "invalid shopId")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) GetAutoReplyPolicy(c *gin.Context) {
	if h == nil || h.Svc == nil {
		response.Fail(c, 500, response.CodeInternalError, "customer chat unavailable")
		return
	}
	shopID, ok := autoReplyShopID(c)
	if !ok {
		return
	}
	out, err := h.Svc.GetAutoReplyPolicy(c, shopID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, 404, response.CodeNotFound, "not found")
			return
		}
		response.HandleError(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) ListAutoReplyRuns(c *gin.Context) {
	if h == nil || h.Svc == nil {
		response.Fail(c, 500, response.CodeInternalError, "customer chat unavailable")
		return
	}
	shopID, ok := autoReplyShopID(c)
	if !ok {
		return
	}
	out, err := h.Svc.ListAutoReplyRuns(c, shopID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, 404, response.CodeNotFound, "not found")
			return
		}
		response.HandleError(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) UpdateAutoReplyPolicy(c *gin.Context) {
	if h == nil || h.Svc == nil {
		response.Fail(c, 500, response.CodeInternalError, "customer chat unavailable")
		return
	}
	if !adminperm.CanManageSettings(c, h.Svc.DB) || !adminperm.CanWriteCustomer(c, h.Svc.DB) {
		response.Fail(c, 403, response.CodeForbidden, "无权管理 AI 自动回复策略")
		return
	}
	shopID, ok := autoReplyShopID(c)
	if !ok {
		return
	}
	var body UpdateAutoReplyPolicyBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, 400, response.CodeBadRequest, "invalid json body")
		return
	}
	out, err := h.Svc.UpdateAutoReplyPolicy(c, shopID, body, adminUUID(c))
	if err != nil {
		if c.IsAborted() {
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, 404, response.CodeNotFound, "not found")
			return
		}
		response.Fail(c, 400, response.CodeBadRequest, err.Error())
		return
	}
	response.OK(c, out)
}
