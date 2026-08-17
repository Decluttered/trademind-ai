package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/pkg/response"
	"strings"
)

type HTTPOperation func(context.Context) (any, string, error)

func ExecuteHTTPCommand(c *gin.Context, svc *Service, workspaceID int64, scope string, payload any, operation HTTPOperation) {
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if key == "" {
		response.Fail(c, 400, response.CodeBadRequest, "Idempotency-Key header is required")
		return
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		response.Fail(c, 400, response.CodeBadRequest, "invalid command payload")
		return
	}
	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])
	owner := "http:" + uuid.NewString()
	result, err := svc.Execute(c.Request.Context(), fmt.Sprintf("mindbay:%d:%s", workspaceID, scope), key, hash, owner, func(ctx context.Context, _ uuid.UUID) (CompleteResult, error) {
		value, resourceID, runErr := operation(ctx)
		if runErr != nil {
			return CompleteResult{}, runErr
		}
		encoded, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			return CompleteResult{}, marshalErr
		}
		return CompleteResult{ResponseCode: "OK", ResponseSummary: string(encoded), ResourceType: scope, ResourceID: resourceID}, nil
	})
	if err != nil {
		response.Fail(c, 409, response.CodeBadRequest, err.Error())
		return
	}
	var data any
	if err := json.Unmarshal([]byte(result.ReplaySummary), &data); err != nil {
		response.Fail(c, 500, response.CodeInternalError, "idempotency replay unavailable")
		return
	}
	response.OK(c, data)
}
