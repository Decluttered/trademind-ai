package customerchat

import (
	"time"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/pkg/id"
	"github.com/trademind-ai/trademind/backend/internal/pkg/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// CustomerConversation is a manual/customer-service conversation (no platform sync in MVP).
type CustomerConversation struct {
	model.Base
	TenantID               int64          `gorm:"not null;default:0;index" json:"tenantId"`
	Platform               string         `gorm:"size:64;index;not null" json:"platform"`
	ShopID                 *uuid.UUID     `gorm:"type:char(36);index" json:"shopId,omitempty"`
	ExternalConversationID *string        `gorm:"size:255;index" json:"externalConversationId,omitempty"`
	CustomerName           string         `gorm:"size:255;not null" json:"customerName"`
	CustomerAvatar         string         `gorm:"type:text" json:"customerAvatar,omitempty"`
	CustomerLanguage       string         `gorm:"size:32;default:en;not null" json:"customerLanguage"`
	Status                 string         `gorm:"size:32;index;not null" json:"status"`
	LastMessageAt          *time.Time     `json:"lastMessageAt,omitempty"`
	OrderID                *uuid.UUID     `gorm:"type:char(36);index" json:"orderId,omitempty"`
	RawData                datatypes.JSON `gorm:"type:jsonb" json:"rawData,omitempty"`
	CreatedBy              *uuid.UUID     `gorm:"type:char(36);index" json:"createdBy,omitempty"`
}

func (CustomerConversation) TableName() string { return "customer_conversations" }

// CustomerMessage is one line in a conversation timeline.
type CustomerMessage struct {
	ID                uuid.UUID      `gorm:"type:char(36);primaryKey" json:"id"`
	ConversationID    uuid.UUID      `gorm:"type:char(36);index;not null" json:"conversationId"`
	ClientMessageID   string         `gorm:"size:128;index" json:"clientMessageId,omitempty"`
	Role              string         `gorm:"size:32;index;not null" json:"role"`
	Content           string         `gorm:"type:text;not null" json:"content"`
	Language          string         `gorm:"size:32;not null" json:"language"`
	MessageType       string         `gorm:"size:32;default:text;not null" json:"messageType"`
	Source            string         `gorm:"size:32;index;not null" json:"source"`
	ExternalMessageID *string        `gorm:"size:255" json:"externalMessageId,omitempty"`
	RawData           datatypes.JSON `gorm:"type:jsonb" json:"rawData,omitempty"`
	CreatedBy         *uuid.UUID     `gorm:"type:char(36);index" json:"createdBy,omitempty"`
	CreatedAt         time.Time      `json:"createdAt"`
}

func (CustomerMessage) TableName() string { return "customer_messages" }

// BeforeCreate assigns UUID when missing.
func (m *CustomerMessage) BeforeCreate(tx *gorm.DB) error {
	id.Ensure(&m.ID)
	return nil
}

// CustomerReplySuggestion stores AI-suggested replies. Manual mode requires confirmation;
// explicitly enabled low-risk auto-reply policies may send a generated suggestion.
type CustomerReplySuggestion struct {
	model.HardDeleteBase
	ConversationID uuid.UUID      `gorm:"type:char(36);index;not null" json:"conversationId"`
	MessageID      *uuid.UUID     `gorm:"type:char(36);index" json:"messageId,omitempty"`
	AITaskID       *uuid.UUID     `gorm:"type:char(36);index" json:"aiTaskId,omitempty"`
	Provider       string         `gorm:"size:64" json:"provider,omitempty"`
	Model          string         `gorm:"size:128" json:"model,omitempty"`
	PromptCode     string         `gorm:"size:64;index" json:"promptCode,omitempty"`
	SuggestedReply string         `gorm:"type:text" json:"suggestedReply,omitempty"`
	EditedReply    string         `gorm:"type:text" json:"editedReply,omitempty"`
	RejectReason   string         `gorm:"type:text" json:"rejectReason,omitempty"`
	Status         string         `gorm:"size:32;index;not null" json:"status"`
	Language       string         `gorm:"size:32" json:"language,omitempty"`
	Tone           string         `gorm:"size:64" json:"tone,omitempty"`
	Input          datatypes.JSON `gorm:"type:jsonb" json:"input,omitempty"`
	Output         datatypes.JSON `gorm:"type:jsonb" json:"output,omitempty"`
	CreatedBy      *uuid.UUID     `gorm:"type:char(36);index" json:"createdBy,omitempty"`
}

func (CustomerReplySuggestion) TableName() string { return "customer_reply_suggestions" }

// CustomerAutoReplyPolicy controls fail-closed AI auto replies for one shop.
// Disabled is the default; enabling requires an explicit Admin action.
type CustomerAutoReplyPolicy struct {
	model.Base
	TenantID            int64      `gorm:"not null;default:0;index" json:"tenantId"`
	ShopID              uuid.UUID  `gorm:"type:char(36);uniqueIndex;not null" json:"shopId"`
	Enabled             bool       `gorm:"not null;default:false;index" json:"enabled"`
	Tone                string     `gorm:"size:64;not null;default:professional" json:"tone"`
	ShopPolicy          string     `gorm:"type:text" json:"shopPolicy,omitempty"`
	MaxReplyRunes       int        `gorm:"not null;default:600" json:"maxReplyRunes"`
	MaxRepliesPerHour   int        `gorm:"not null;default:20" json:"maxRepliesPerHour"`
	RequireOrderContext bool       `gorm:"not null;default:true" json:"requireOrderContext"`
	LowRiskOnly         bool       `gorm:"not null;default:true" json:"lowRiskOnly"`
	LastEnabledAt       *time.Time `json:"lastEnabledAt,omitempty"`
	LastEnabledBy       *uuid.UUID `gorm:"type:char(36);index" json:"lastEnabledBy,omitempty"`
	NextPollAt          *time.Time `gorm:"index" json:"-"`
}

func (CustomerAutoReplyPolicy) TableName() string { return "customer_auto_reply_policies" }

// CustomerAutoReplySetting stores tenant-wide runtime controls managed from Admin.
// Missing rows are treated as disabled so a deployment never enables automatic sends by accident.
type CustomerAutoReplySetting struct {
	model.Base
	TenantID            int64      `gorm:"not null;default:0;uniqueIndex" json:"tenantId"`
	MessageSyncEnabled  bool       `gorm:"not null;default:false" json:"messageSyncEnabled"`
	AutoReplyEnabled    bool       `gorm:"not null;default:false" json:"autoReplyEnabled"`
	PollIntervalSeconds int        `gorm:"not null;default:60" json:"pollIntervalSeconds"`
	UpdatedBy           *uuid.UUID `gorm:"type:char(36);index" json:"updatedBy,omitempty"`
}

func (CustomerAutoReplySetting) TableName() string { return "customer_auto_reply_settings" }

// CustomerAutoReplyRun is one idempotent decision for one inbound customer message.
type CustomerAutoReplyRun struct {
	model.Base
	TenantID       int64      `gorm:"not null;default:0;index" json:"tenantId"`
	ShopID         uuid.UUID  `gorm:"type:char(36);index;not null" json:"shopId"`
	ConversationID uuid.UUID  `gorm:"type:char(36);index;not null" json:"conversationId"`
	MessageID      uuid.UUID  `gorm:"type:char(36);uniqueIndex;not null" json:"messageId"`
	SuggestionID   *uuid.UUID `gorm:"type:char(36);index" json:"suggestionId,omitempty"`
	SentMessageID  *uuid.UUID `gorm:"type:char(36);index" json:"sentMessageId,omitempty"`
	Status         string     `gorm:"size:32;index;not null" json:"status"`
	RiskLevel      string     `gorm:"size:32;index" json:"riskLevel,omitempty"`
	ReasonCode     string     `gorm:"size:128;index" json:"reasonCode,omitempty"`
	ErrorMessage   string     `gorm:"type:text" json:"errorMessage,omitempty"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	FinishedAt     *time.Time `json:"finishedAt,omitempty"`
	LockedBy       *string    `gorm:"size:220;index" json:"-"`
	LockedUntil    *time.Time `gorm:"index" json:"-"`
	LockVersion    int        `gorm:"default:0;not null" json:"-"`
	HeartbeatAt    *time.Time `gorm:"index" json:"-"`
	ExecutionID    *string    `gorm:"size:36;index" json:"-"`
}

func (CustomerAutoReplyRun) TableName() string { return "customer_auto_reply_runs" }
