package customerchat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/modules/shop"
	platformp "github.com/trademind-ai/trademind/backend/internal/providers/platform"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SyncPlatformCustomerMessages upserts rows from a normalized provider pull result.
func (s *Service) SyncPlatformCustomerMessages(ctx context.Context, shopRow *shop.Shop, pull *platformp.PullMessagesResult) (conversationsTouched int, messagesInserted int, inboundMessageIDs []uuid.UUID, err error) {
	if s == nil || s.DB == nil {
		return 0, 0, nil, fmt.Errorf("customerchat: no db")
	}
	if shopRow == nil || pull == nil {
		return 0, 0, nil, fmt.Errorf("invalid sync payload")
	}
	shopID := shopRow.ID
	platformKey := strings.TrimSpace(shopRow.Platform)

	for i := range pull.Conversations {
		pc := &pull.Conversations[i]
		var touched, inserted int
		var inbound []uuid.UUID
		externalConversationID := strings.TrimSpace(pc.ExternalConversationID)
		if externalConversationID == "" {
			continue
		}
		lockID := deterministicConversationLockID(shopID, platformKey, externalConversationID)
		var existing CustomerConversation
		findExistingErr := s.DB.WithContext(ctx).Select("id").
			Where("shop_id = ? AND platform = ? AND external_conversation_id = ?", shopID, platformKey, externalConversationID).
			First(&existing).Error
		if findExistingErr == nil {
			lockID = existing.ID
		} else if !errors.Is(findExistingErr, gorm.ErrRecordNotFound) {
			return 0, 0, nil, findExistingErr
		}
		err = s.withConversationMutationLock(ctx, lockID, func() error {
			return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				ext := strings.TrimSpace(pc.ExternalConversationID)
				if ext == "" {
					return nil
				}

				var conv CustomerConversation
				q := tx.Where("shop_id = ? AND platform = ? AND external_conversation_id = ?", shopID, platformKey, ext)
				findErr := q.First(&conv).Error
				rawConv := platformp.TrimRawMap(pc.RawData, 12, 400)
				rawJSON, _ := json.Marshal(rawConv)

				custName := strings.TrimSpace(pc.CustomerName)
				if custName == "" {
					custName = "Customer"
				}
				lang := strings.TrimSpace(pc.CustomerLanguage)
				if lang == "" {
					lang = "en"
				}
				avatar := strings.TrimSpace(pc.CustomerAvatar)
				convStatus := strings.TrimSpace(pc.Status)
				if convStatus == "" {
					convStatus = StatusOpen
				}
				lastAt := pc.LastMessageAt

				if errors.Is(findErr, gorm.ErrRecordNotFound) {
					extCopy := ext
					conv = CustomerConversation{
						TenantID:               shopRow.TenantID,
						Platform:               platformKey,
						ShopID:                 &shopID,
						ExternalConversationID: &extCopy,
						CustomerName:           custName,
						CustomerAvatar:         avatar,
						CustomerLanguage:       lang,
						Status:                 convStatus,
						LastMessageAt:          lastAt,
						RawData:                datatypes.JSON(rawJSON),
					}
					created := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&conv)
					if created.Error != nil {
						return created.Error
					}
					if created.RowsAffected == 0 {
						if err := q.First(&conv).Error; err != nil {
							return err
						}
					} else {
						touched++
					}
				} else if findErr != nil {
					return findErr
				} else {
					updates := map[string]any{
						"customer_name":     custName,
						"customer_avatar":   avatar,
						"customer_language": lang,
						"status":            convStatus,
						"raw_data":          datatypes.JSON(rawJSON),
						"updated_at":        time.Now().UTC(),
					}
					if lastAt != nil {
						updates["last_message_at"] = lastAt
					}
					if err := tx.Model(&CustomerConversation{}).Where("id = ?", conv.ID).Updates(updates).Error; err != nil {
						return err
					}
					touched++
				}

				var latestRole string
				var latestAt *time.Time
				for j := range pc.Messages {
					pm := &pc.Messages[j]
					emid := strings.TrimSpace(pm.ExternalMessageID)
					if emid == "" {
						continue
					}
					role := normalizePlatformRole(pm.Role)
					mt := strings.TrimSpace(pm.MessageType)
					if mt == "" {
						mt = MessageTypeText
					}
					mlang := strings.TrimSpace(pm.Language)
					if mlang == "" {
						mlang = lang
					}
					content := strings.TrimSpace(pm.Content)
					emCopy := emid
					rawMsg := platformp.TrimRawMap(pm.RawData, 12, 400)
					rb, _ := json.Marshal(rawMsg)
					msg := &CustomerMessage{
						ConversationID:    conv.ID,
						Role:              role,
						Content:           content,
						Language:          mlang,
						MessageType:       mt,
						Source:            SourcePlatform,
						ExternalMessageID: &emCopy,
						RawData:           datatypes.JSON(rb),
					}
					if pm.CreatedAt != nil {
						msg.CreatedAt = pm.CreatedAt.UTC()
					}
					created := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(msg)
					if created.Error != nil {
						return created.Error
					}
					if created.RowsAffected == 0 {
						continue
					}
					inserted++
					if role == RoleCustomer && mt == MessageTypeText && content != "" {
						inbound = append(inbound, msg.ID)
					}
					latestRole = role
					if pm.CreatedAt != nil {
						t := *pm.CreatedAt
						latestAt = &t
					} else {
						t := time.Now().UTC()
						latestAt = &t
					}
				}

				if latestAt != nil {
					st := conv.Status
					switch latestRole {
					case RoleCustomer:
						st = StatusPendingReply
					case RoleAgent:
						st = StatusReplied
					}
					if err := tx.Model(&CustomerConversation{}).Where("id = ?", conv.ID).Updates(map[string]any{
						"last_message_at": latestAt,
						"status":          st,
						"updated_at":      time.Now().UTC(),
					}).Error; err != nil {
						return err
					}
				}
				return nil
			})
		})
		if err != nil {
			return 0, 0, nil, err
		}
		conversationsTouched += touched
		messagesInserted += inserted
		inboundMessageIDs = append(inboundMessageIDs, inbound...)
	}
	return conversationsTouched, messagesInserted, inboundMessageIDs, nil
}

func deterministicConversationLockID(shopID uuid.UUID, platformKey, externalConversationID string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(shopID.String()+"\x00"+platformKey+"\x00"+externalConversationID))
}

func normalizePlatformRole(r string) string {
	v := strings.TrimSpace(strings.ToLower(r))
	switch v {
	case "agent", "shop", "seller", "merchant", "operator":
		return RoleAgent
	case "system":
		return RoleAI
	case "customer", "buyer", "user", "":
		return RoleCustomer
	default:
		return RoleCustomer
	}
}
