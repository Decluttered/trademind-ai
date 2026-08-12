package redis_test

import (
	"context"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/testing/safeenv"
)

func openTestRedis(t *testing.T) (*goredis.Client, context.Context) {
	t.Helper()
	cfg, ok, err := safeenv.TestRedisURLFromEnv()
	require.NoError(t, err)
	if !ok {
		t.Skip("TEST_REDIS_URL is not set; skipping Redis queue integration test")
	}
	options, err := goredis.ParseURL(cfg.URL)
	require.NoError(t, err)
	client := goredis.NewClient(options)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)
	require.NoError(t, client.Ping(ctx).Err())
	return client, ctx
}

func TestRedisListQueueRoundTripInIsolatedDB(t *testing.T) {
	client, ctx := openTestRedis(t)
	key := "test:trademind:queue:roundtrip"
	require.NoError(t, client.Del(ctx, key).Err())
	t.Cleanup(func() { _ = client.Del(context.Background(), key).Err() })

	require.NoError(t, client.LPush(ctx, key, `{"taskId":"test-task-1"}`).Err())
	item, err := client.BRPop(ctx, time.Second, key).Result()
	require.NoError(t, err)
	require.Equal(t, []string{key, `{"taskId":"test-task-1"}`}, item)
}

func TestRedisReliableListReservationAckAndRequeue(t *testing.T) {
	client, ctx := openTestRedis(t)
	ready := "test:trademind:queue:reliable"
	processing := ready + ":processing"
	payload := `{"taskId":"test-task-reliable"}`
	require.NoError(t, client.Del(ctx, ready, processing).Err())
	t.Cleanup(func() { _ = client.Del(context.Background(), ready, processing).Err() })

	require.NoError(t, client.LPush(ctx, ready, payload).Err())
	reserved, err := client.BRPopLPush(ctx, ready, processing, time.Second).Result()
	require.NoError(t, err)
	require.Equal(t, payload, reserved)
	require.Equal(t, int64(0), client.LLen(ctx, ready).Val())
	require.Equal(t, int64(1), client.LLen(ctx, processing).Val())

	_, err = client.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
		pipe.LPush(ctx, ready, reserved)
		pipe.LRem(ctx, processing, 1, reserved)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), client.LLen(ctx, ready).Val())
	require.Equal(t, int64(0), client.LLen(ctx, processing).Val())

	reserved, err = client.BRPopLPush(ctx, ready, processing, time.Second).Result()
	require.NoError(t, err)
	removed, err := client.LRem(ctx, processing, 1, reserved).Result()
	require.NoError(t, err)
	require.Equal(t, int64(1), removed)
	require.Equal(t, int64(0), client.LLen(ctx, ready).Val())
	require.Equal(t, int64(0), client.LLen(ctx, processing).Val())
}
