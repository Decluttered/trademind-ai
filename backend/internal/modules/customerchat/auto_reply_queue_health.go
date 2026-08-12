package customerchat

import (
	"context"
	"strings"
	"time"

	"github.com/trademind-ai/trademind/backend/internal/rdb"
)

// AutoReplyQueueHealthBlock exposes queue and consumer state for production health checks.
type AutoReplyQueueHealthBlock struct {
	Enabled                  bool   `json:"enabled"`
	QueueName                string `json:"queueName,omitempty"`
	ProcessingName           string `json:"processingName,omitempty"`
	RedisOK                  bool   `json:"redisOk"`
	RedisAvailable           bool   `json:"redisAvailable"`
	ReadyLength              int64  `json:"readyLength,omitempty"`
	ProcessingLength         int64  `json:"processingLength,omitempty"`
	WorkerRunning            bool   `json:"workerRunning"`
	MessageSyncWorkerRunning bool   `json:"messageSyncWorkerRunning"`
	SchedulerRunning         bool   `json:"schedulerRunning"`
	WorkerConcurrency        int    `json:"workerConcurrency,omitempty"`
}

func normalizedAutoReplyHealthQueueName(queueName string) string {
	if q := strings.TrimSpace(queueName); q != "" {
		return q
	}
	return "customer:auto:reply:tasks"
}

// BuildAutoReplyQueueHealthBlock checks Redis connectivity and both reliable-list queues.
func BuildAutoReplyQueueHealthBlock(ctx context.Context, redis *rdb.Client, queueName string, workerConcurrency int) AutoReplyQueueHealthBlock {
	queueName = normalizedAutoReplyHealthQueueName(queueName)
	if workerConcurrency < 1 {
		workerConcurrency = 1
	}
	out := AutoReplyQueueHealthBlock{
		Enabled:           true,
		QueueName:         queueName,
		ProcessingName:    queueName + ":processing",
		WorkerRunning:     AutoReplyWorkersRunning(),
		WorkerConcurrency: workerConcurrency,
	}
	if redis == nil || redis.Client == nil {
		return out
	}
	if err := redis.Ping(ctx).Err(); err != nil {
		return out
	}
	out.RedisOK = true
	out.RedisAvailable = true
	pipe := redis.Pipeline()
	ready := pipe.LLen(ctx, out.QueueName)
	processing := pipe.LLen(ctx, out.ProcessingName)
	if _, err := pipe.Exec(ctx); err != nil {
		out.RedisAvailable = false
		return out
	}
	out.ReadyLength = ready.Val()
	out.ProcessingLength = processing.Val()
	return out
}

func (s *Service) autoReplyRuntimeAvailable(ctx context.Context) bool {
	if s == nil || s.Redis == nil || s.Redis.Client == nil || !AutoReplyWorkersRunning() {
		return false
	}
	if s.AutoReplyDependenciesAvailable != nil && !s.AutoReplyDependenciesAvailable() {
		return false
	}
	pingCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return s.Redis.Ping(pingCtx).Err() == nil
}
