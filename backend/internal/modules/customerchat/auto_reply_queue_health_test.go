package customerchat

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAutoReplyQueueHealthRequiresRedisAndRunningWorker(t *testing.T) {
	out := BuildAutoReplyQueueHealthBlock(context.Background(), nil, "", 0)
	require.True(t, out.Enabled)
	require.Equal(t, "customer:auto:reply:tasks", out.QueueName)
	require.False(t, out.RedisAvailable)
	require.False(t, out.WorkerRunning)

	markAutoReplyWorkerStarted()
	defer markAutoReplyWorkerStopped()
	out = BuildAutoReplyQueueHealthBlock(context.Background(), nil, "custom:auto", 3)
	require.True(t, out.WorkerRunning)
	require.Equal(t, "custom:auto:processing", out.ProcessingName)
	require.Equal(t, 3, out.WorkerConcurrency)
}
