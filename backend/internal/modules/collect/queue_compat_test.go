package collect

import (
	"context"
	"net"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/rdb"
)

type redisCommandCaptureHook struct {
	name string
	args []interface{}
}

func (h *redisCommandCaptureHook) DialHook(next goredis.DialHook) goredis.DialHook {
	return next
}

func (h *redisCommandCaptureHook) ProcessHook(_ goredis.ProcessHook) goredis.ProcessHook {
	return func(_ context.Context, cmd goredis.Cmder) error {
		h.name = cmd.Name()
		h.args = append([]interface{}(nil), cmd.Args()...)
		if stringCmd, ok := cmd.(*goredis.StringCmd); ok {
			stringCmd.SetVal(`{"taskId":"test-task"}`)
		}
		return nil
	}
}

func (h *redisCommandCaptureHook) ProcessPipelineHook(next goredis.ProcessPipelineHook) goredis.ProcessPipelineHook {
	return next
}

func TestReserveTaskUsesRedisFiveCompatibleReliableMove(t *testing.T) {
	client := goredis.NewClient(&goredis.Options{
		Addr: "unused:6379",
		Dialer: func(context.Context, string, string) (net.Conn, error) {
			return nil, nil
		},
	})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	hook := &redisCommandCaptureHook{}
	client.AddHook(hook)
	svc := &Service{Redis: &rdb.Client{Client: client}}

	payload, err := svc.reserveTask(context.Background(), "collect:test", time.Second)
	require.NoError(t, err)
	require.Equal(t, `{"taskId":"test-task"}`, payload)
	require.Equal(t, "brpoplpush", hook.name)
	require.Equal(t, []interface{}{"brpoplpush", "collect:test", "collect:test:processing", int64(1)}, hook.args)
}
