package catalog

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/modules/idempotency"
	"github.com/trademind-ai/trademind/backend/internal/pkg/adminperm"
	"github.com/trademind-ai/trademind/backend/internal/pkg/response"
	"strconv"
	"strings"
)

type Handler struct {
	Svc         *Service
	Idempotency *idempotency.Service
}

func workspace(c *gin.Context) (int64, bool) {
	id, err := adminperm.TenantIDFromGin(c)
	if err != nil {
		response.Fail(c, 401, response.CodeUnauthorized, "workspace unavailable")
		return 0, false
	}
	return id, true
}
func limit(c *gin.Context) int { n, _ := strconv.Atoi(c.Query("limit")); return n }
func (h *Handler) Discovery(c *gin.Context) {
	w, ok := workspace(c)
	if !ok {
		return
	}
	var b struct {
		URL string `json:"url"`
	}
	if c.ShouldBindJSON(&b) != nil || strings.TrimSpace(b.URL) == "" {
		response.Fail(c, 400, response.CodeBadRequest, "url is required")
		return
	}
	idempotency.ExecuteHTTPCommand(c, h.Idempotency, w, "discovery-runs", b, func(ctx context.Context) (any, string, error) {
		out, err := h.Svc.RunDiscovery(ctx, w, b.URL)
		if err != nil {
			return nil, "", err
		}
		return out, out.Snapshot.ID.String(), nil
	})
}
func (h *Handler) Products(c *gin.Context) {
	w, ok := workspace(c)
	if !ok {
		return
	}
	filter := ProductFilter{Query: c.Query("q")}
	if raw := strings.TrimSpace(c.Query("collectionId")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Fail(c, 400, response.CodeBadRequest, "invalid collectionId")
			return
		}
		filter.CollectionID = &id
	}
	if raw := strings.TrimSpace(c.Query("minScore")); raw != "" {
		score, err := strconv.Atoi(raw)
		if err != nil || score < 0 || score > 100 {
			response.Fail(c, 400, response.CodeBadRequest, "minScore must be between 0 and 100")
			return
		}
		filter.MinScore = &score
	}
	out, err := h.Svc.ListProducts(c, w, limit(c), c.Query("cursor"), filter)
	if err != nil {
		response.Fail(c, 400, response.CodeBadRequest, err.Error())
		return
	}
	response.OK(c, out)
}
func (h *Handler) CreateCollection(c *gin.Context) {
	w, ok := workspace(c)
	if !ok {
		return
	}
	var b CreateCollectionInput
	if c.ShouldBindJSON(&b) != nil {
		response.Fail(c, 400, response.CodeBadRequest, "invalid collection")
		return
	}
	idempotency.ExecuteHTTPCommand(c, h.Idempotency, w, "collections:create", b, func(ctx context.Context) (any, string, error) {
		out, err := h.Svc.CreateCollection(ctx, w, b)
		if err != nil {
			return nil, "", err
		}
		return out, out.ID.String(), nil
	})
}
func (h *Handler) AddCollectionProduct(c *gin.Context) {
	w, ok := workspace(c)
	if !ok {
		return
	}
	collectionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, 400, response.CodeBadRequest, "invalid collection id")
		return
	}
	var b struct {
		SourceProductID uuid.UUID `json:"sourceProductId"`
		Reason          string    `json:"reason"`
	}
	if c.ShouldBindJSON(&b) != nil || b.SourceProductID == uuid.Nil {
		response.Fail(c, 400, response.CodeBadRequest, "sourceProductId is required")
		return
	}
	idempotency.ExecuteHTTPCommand(c, h.Idempotency, w, "collections:add-product:"+collectionID.String(), b, func(ctx context.Context) (any, string, error) {
		out, err := h.Svc.AddProductToCollection(ctx, w, collectionID, b.SourceProductID, b.Reason)
		if err != nil {
			return nil, "", err
		}
		return out, out.ID.String(), nil
	})
}
func (h *Handler) Collections(c *gin.Context) {
	w, ok := workspace(c)
	if !ok {
		return
	}
	out, err := h.Svc.ListCollections(c, w, limit(c), c.Query("cursor"))
	if err != nil {
		response.Fail(c, 400, response.CodeBadRequest, err.Error())
		return
	}
	response.OK(c, out)
}
func Register(r gin.IRouter, h *Handler) {
	r.POST("/discovery-runs", h.Discovery)
	r.GET("/products", h.Products)
	r.GET("/collections", h.Collections)
	r.POST("/collections", h.CreateCollection)
	r.POST("/collections/:id/products", h.AddCollectionProduct)
}
