package customerchat

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const conversationMutationLockTTL = 150 * time.Second

func (s *Service) withConversationMutationLock(ctx context.Context, conversationID uuid.UUID, fn func() error) error {
	if s == nil || s.Redis == nil || s.Redis.Client == nil {
		return fn()
	}
	key := "customer:conversation:mutation:" + conversationID.String()
	token := uuid.NewString()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		locked, err := s.Redis.SetNX(ctx, key, token, conversationMutationLockTTL).Result()
		if err != nil {
			return err
		}
		if locked {
			defer s.releaseConversationMutationLock(key, token)
			return fn()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("customer conversation is busy")
		case <-ticker.C:
		}
	}
}

func (s *Service) releaseConversationMutationLock(key, token string) {
	const script = `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`
	_ = s.Redis.Eval(context.Background(), script, []string{key}, token).Err()
}
