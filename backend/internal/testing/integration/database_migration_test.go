package integration

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/database"
	"github.com/trademind-ai/trademind/backend/internal/modules/customerchat"
	"github.com/trademind-ai/trademind/backend/internal/modules/customersync"
	"github.com/trademind-ai/trademind/backend/internal/modules/productioncontrolp10"
	"github.com/trademind-ai/trademind/backend/internal/testing/postgrestest"
	"github.com/trademind-ai/trademind/backend/internal/testing/safeenv"
)

type legacyRuntimeControl struct {
	ID                 uuid.UUID `gorm:"type:char(36);primaryKey"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	TenantID           int64 `gorm:"not null;uniqueIndex"`
	ProviderKillActive bool  `gorm:"not null;default:true"`
	TenantKillActive   bool  `gorm:"not null;default:true"`
	ShopKillActive     bool  `gorm:"not null;default:true"`
	ReadKillActive     bool  `gorm:"not null;default:true"`
	WriteKillActive    bool  `gorm:"not null;default:true"`
	Revision           int   `gorm:"not null;default:1"`
}

func TestCustomerAutoReplyReliabilityConstraints(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	_, ok, err := safeenv.TestDatabaseURLFromEnv()
	require.NoError(t, err)
	if !ok {
		t.Skip("TEST_DATABASE_URL is not set; skipping customer auto-reply PostgreSQL constraints")
	}

	db := postgrestest.Require(t).DB
	require.NoError(t, database.AutoMigrate(db))
	require.NoError(t, database.AutoMigrate(db))

	shopID := uuid.New()
	conversationID := uuid.New()
	externalMessageID := "platform-message-1"
	first := customerchat.CustomerMessage{ConversationID: conversationID, Role: customerchat.RoleCustomer, Content: "hello", Language: "en", MessageType: customerchat.MessageTypeText, Source: customerchat.SourcePlatform, ExternalMessageID: &externalMessageID}
	require.NoError(t, db.Create(&first).Error)
	second := first
	second.ID = uuid.Nil
	require.Error(t, db.Create(&second).Error)

	task := customersync.CustomerMessageSyncTask{ShopID: shopID, Platform: "mock", TaskType: customersync.TaskTypeCustomerMessageSync, Status: customersync.StatusPending, Mode: customersync.ModeIncremental}
	require.NoError(t, db.Create(&task).Error)
	duplicateTask := task
	duplicateTask.ID = uuid.New()
	require.Error(t, db.Create(&duplicateTask).Error)
}

func TestCustomerAutoReplyMigrationRejectsDuplicateReferencedAsSentMessage(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	_, ok, err := safeenv.TestDatabaseURLFromEnv()
	require.NoError(t, err)
	if !ok {
		t.Skip("TEST_DATABASE_URL is not set; skipping customer auto-reply PostgreSQL migration guard")
	}

	db := postgrestest.Require(t).DB
	require.NoError(t, db.AutoMigrate(
		&customerchat.CustomerMessage{},
		&customerchat.CustomerReplySuggestion{},
		&customerchat.CustomerAutoReplyRun{},
	))
	conversationID := uuid.New()
	externalMessageID := "duplicate-platform-message"
	keeper := customerchat.CustomerMessage{
		ConversationID: conversationID, Role: customerchat.RoleAgent, Content: "sent reply", Language: "en",
		MessageType: customerchat.MessageTypeText, Source: customerchat.SourcePlatform, ExternalMessageID: &externalMessageID,
		CreatedAt: time.Now().UTC().Add(-time.Minute),
	}
	require.NoError(t, db.Create(&keeper).Error)
	duplicate := keeper
	duplicate.ID = uuid.Nil
	duplicate.CreatedAt = time.Now().UTC()
	require.NoError(t, db.Create(&duplicate).Error)
	sourceMessageID := uuid.New()
	run := customerchat.CustomerAutoReplyRun{
		TenantID: 7, ShopID: uuid.New(), ConversationID: conversationID, MessageID: sourceMessageID,
		SentMessageID: &duplicate.ID, Status: customerchat.AutoReplyRunSent,
	}
	require.NoError(t, db.Create(&run).Error)

	err = database.AutoMigrate(db)
	require.ErrorContains(t, err, "duplicate rows are referenced")
	var duplicateCount int64
	require.NoError(t, db.Model(&customerchat.CustomerMessage{}).Where("conversation_id = ? AND external_message_id = ?", conversationID, externalMessageID).Count(&duplicateCount).Error)
	require.Equal(t, int64(2), duplicateCount)
}

func (legacyRuntimeControl) TableName() string { return "p10_runtime_controls" }

func TestAutoMigrateAgainstIsolatedPostgres(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	_, ok, err := safeenv.TestDatabaseURLFromEnv()
	require.NoError(t, err)
	if !ok {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration migration test")
	}

	harness := postgrestest.Require(t)
	db := harness.DB

	require.NoError(t, database.AutoMigrateWithP10(db))

	for _, table := range []string{
		"admin_users",
		"products",
		"product_skus",
		"product_publish_tasks",
		"inventory_sync_tasks",
		"inventory_sync_runs",
		"inventory_snapshot_items",
		"sku_bindings",
		"sku_binding_calibrations",
		"manual_binding_requests",
		"manual_binding_decisions",
		"platform_credentials",
		"platform_oauth_credentials",
		"platform_credential_bindings",
		"platform_credential_versions",
		"platform_credential_lifecycle_events",
		"platform_oauth_states",
		"production_runtime_controls",
		"production_scope_allowlists",
		"production_rollout_policies",
		"production_control_audit_events",
	} {
		require.Truef(t, db.Migrator().HasTable(table), "expected migrated table %s", table)
	}
	for _, legacy := range []string{
		"p9_inventory_sync_runs",
		"p9_inventory_snapshot_items",
		"p9_sku_bindings",
		"p9_sku_binding_calibrations",
		"p9_manual_binding_requests",
		"p9_manual_binding_decisions",
		"p10_platform_credentials",
		"p10_oauth_credentials",
		"p10_credential_bindings",
		"p10_credential_versions",
		"p10_credential_lifecycle_events",
		"p10_oauth_states",
		"p10_runtime_controls",
		"p10_scope_allowlists",
		"p10_gray_policies",
		"p10_control_audit_events",
	} {
		require.Falsef(t, db.Migrator().HasTable(legacy), "legacy table must not be created: %s", legacy)
	}
}

func TestAutoMigrateWithP10RenamesLegacySchemaWithoutDataLoss(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	_, ok, err := safeenv.TestDatabaseURLFromEnv()
	require.NoError(t, err)
	if !ok {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration migration test")
	}

	db := postgrestest.Require(t).DB
	require.NoError(t, db.AutoMigrate(&legacyRuntimeControl{}))
	legacy := legacyRuntimeControl{
		ID:                 uuid.New(),
		TenantID:           900101,
		ProviderKillActive: true,
		TenantKillActive:   true,
		ShopKillActive:     true,
		ReadKillActive:     true,
		WriteKillActive:    true,
		Revision:           7,
	}
	require.NoError(t, db.Create(&legacy).Error)

	require.NoError(t, database.AutoMigrateWithP10(db))
	require.NoError(t, database.AutoMigrateWithP10(db))
	require.False(t, db.Migrator().HasTable("p10_runtime_controls"))
	require.True(t, db.Migrator().HasTable("production_runtime_controls"))
	require.True(t, db.Migrator().HasIndex(&productioncontrolp10.RuntimeControl{}, "idx_production_runtime_controls_tenant_id"))

	var migrated productioncontrolp10.RuntimeControl
	require.NoError(t, db.Where("id = ?", legacy.ID).First(&migrated).Error)
	require.Equal(t, legacy.TenantID, migrated.TenantID)
	require.Equal(t, legacy.Revision, migrated.Revision)
}
