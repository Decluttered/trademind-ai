package listingstudio

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/trademind-ai/trademind/backend/internal/modules/idempotency"
	"github.com/trademind-ai/trademind/backend/internal/pkg/adminperm"
	"github.com/trademind-ai/trademind/backend/internal/pkg/response"
)

type ImageHandler struct {
	Pipeline    *ImagePipeline
	Idempotency *idempotency.Service
}

func (h *ImageHandler) Create(c *gin.Context) {
	w, err := adminperm.TenantIDFromGin(c)
	if err != nil {
		response.Fail(c, 401, response.CodeUnauthorized, "workspace unavailable")
		return
	}
	var b struct {
		OriginURL string `json:"originUrl"`
	}
	if c.ShouldBindJSON(&b) != nil {
		response.Fail(c, 400, response.CodeBadRequest, "invalid json body")
		return
	}
	idempotency.ExecuteHTTPCommand(c, h.Idempotency, w, "image-assets", b, func(ctx context.Context) (any, string, error) {
		out, err := h.Pipeline.Ingest(ctx, w, b.OriginURL)
		if err != nil {
			return nil, "", err
		}
		return out, out.ID.String(), nil
	})
}
