package customerchat

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/modules/shop"
	"github.com/trademind-ai/trademind/backend/internal/pkg/model"
	"github.com/trademind-ai/trademind/backend/internal/rdb"
	"github.com/trademind-ai/trademind/backend/internal/testing/safeenv"
	"gorm.io/gorm"
)

func openAutoReplyTestRedis(t *testing.T) (*goredis.Client, context.Context) {
	t.Helper()
	cfg, ok, err := safeenv.TestRedisURLFromEnv()
	require.NoError(t, err)
	if !ok {
		t.Skip("TEST_REDIS_URL is not set; skipping auto-reply Redis integration test")
	}
	options, err := goredis.ParseURL(cfg.URL)
	require.NoError(t, err)
	client := goredis.NewClient(options)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	require.NoError(t, client.Ping(ctx).Err())
	return client, ctx
}

func TestAutoReplyProcessingRecoveryRequeuesWithNewOwnerToken(t *testing.T) {
	client, ctx := openAutoReplyTestRedis(t)
	queueName := "test:customer:auto:reply:" + uuid.NewString()
	processingName := queueName + ":processing"

	db, err := gorm.Open(sqlite.Open("file:auto_reply_queue_"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&shop.Shop{}, &CustomerMessage{}, &CustomerAutoReplyRun{}))
	svc := &Service{DB: db, Redis: &rdb.Client{Client: client}, AutoReplyQueueName: queueName}

	shopRow := shop.Shop{Base: model.Base{ID: uuid.New()}, Platform: "mock", ShopName: "test", Status: shop.StatusActive, AuthStatus: shop.AuthAuthorized}
	require.NoError(t, db.Create(&shopRow).Error)
	message := CustomerMessage{ConversationID: uuid.New(), Role: RoleCustomer, Content: "hello", Language: "en", MessageType: MessageTypeText, Source: SourcePlatform}
	require.NoError(t, db.Create(&message).Error)
	run := CustomerAutoReplyRun{TenantID: shopRow.TenantID, ShopID: shopRow.ID, ConversationID: message.ConversationID, MessageID: message.ID, Status: AutoReplyRunPending}
	require.NoError(t, db.Create(&run).Error)

	dispatchKey := autoReplyDispatchKey(message.ID)
	t.Cleanup(func() { _ = client.Del(context.Background(), queueName, processingName, dispatchKey).Err() })
	require.NoError(t, client.Del(ctx, queueName, processingName, dispatchKey).Err())
	require.NoError(t, svc.enqueueAutoReplyWakeup(ctx, shopRow.ID, message.ID))
	firstToken, err := client.Get(ctx, dispatchKey).Result()
	require.NoError(t, err)
	payload, err := client.BRPopLPush(ctx, queueName, processingName, time.Second).Result()
	require.NoError(t, err)

	require.NoError(t, svc.recoverAutoReplyQueue(ctx, nil))
	require.Equal(t, int64(1), client.LLen(ctx, queueName).Val())
	require.Equal(t, int64(0), client.LLen(ctx, processingName).Val())
	secondToken, err := client.Get(ctx, dispatchKey).Result()
	require.NoError(t, err)
	require.NotEqual(t, firstToken, secondToken)

	requeuedPayload, err := client.BRPopLPush(ctx, queueName, processingName, time.Second).Result()
	require.NoError(t, err)
	queued, _, _, err := decodeAutoReplyQueueMessage(requeuedPayload)
	require.NoError(t, err)
	require.Equal(t, secondToken, queued.DispatchToken)
	require.NoError(t, svc.acknowledgeAutoReplyPayload(ctx, requeuedPayload, message.ID, queued.DispatchToken))
	require.Equal(t, int64(0), client.LLen(ctx, processingName).Val())
	require.ErrorIs(t, client.Get(ctx, dispatchKey).Err(), goredis.Nil)
	require.NotEmpty(t, payload)
}
