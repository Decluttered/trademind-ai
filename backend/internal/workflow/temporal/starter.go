package temporal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/trademind-ai/trademind/backend/internal/modules/publication"
	"go.temporal.io/api/serviceerror"
	temporalclient "go.temporal.io/sdk/client"
)

const TaskQueue = "mindbay-publication"

type Starter struct {
	Address   string
	Namespace string
}

type PublishListingInput struct {
	WorkspaceID   int64  `json:"workspaceId"`
	PublicationID string `json:"publicationId"`
}

func (s Starter) StartPublishListing(ctx context.Context, job publication.PublicationJob) error {
	address := strings.TrimSpace(s.Address)
	if address == "" {
		return fmt.Errorf("TEMPORAL_ADDRESS is required")
	}
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	namespace := strings.TrimSpace(s.Namespace)
	if namespace == "" {
		namespace = "default"
	}
	client, err := temporalclient.DialContext(dialCtx, temporalclient.Options{HostPort: address, Namespace: namespace})
	if err != nil {
		return err
	}
	defer client.Close()
	delay := time.Until(job.ScheduledFor)
	if delay < 0 {
		delay = 0
	}
	_, err = client.ExecuteWorkflow(ctx, temporalclient.StartWorkflowOptions{ID: job.WorkflowID, TaskQueue: TaskQueue, StartDelay: delay, WorkflowExecutionTimeout: 45 * 24 * time.Hour, WorkflowTaskTimeout: 10 * time.Second}, "PublishListingWorkflow", PublishListingInput{WorkspaceID: job.WorkspaceID, PublicationID: job.ID.String()})
	var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
	if errors.As(err, &alreadyStarted) {
		return nil
	}
	return err
}
