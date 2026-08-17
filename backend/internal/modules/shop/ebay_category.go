package shop

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	platformp "github.com/trademind-ai/trademind/backend/internal/providers/platform"
	platformebay "github.com/trademind-ai/trademind/backend/internal/providers/platform/ebay"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

func (s *Service) ebayAppRuntime(ctx context.Context) (platformebay.RuntimeConfig, error) {
	req := platformp.TestConnectionRequest{Extra: map[string]string{}}
	if s.Settings != nil {
		plain, err := s.Settings.PlainByGroup(ctx, 0, "platform_ebay")
		if err != nil {
			return platformebay.RuntimeConfig{}, err
		}
		for key, value := range plain {
			req.Extra[key] = value
		}
	}
	if strings.TrimSpace(req.Extra["environment"]) == "" {
		req.Extra["environment"] = strings.TrimSpace(s.EbayEnv)
	}
	return platformebay.ResolveRuntime(req)
}

func (s *Service) SyncEbayCategoryAspects(ctx context.Context, categoryID string) ([]PlatformCategoryAttribute, error) {
	categoryID = strings.TrimSpace(categoryID)
	if categoryID == "" {
		return nil, fmt.Errorf("categoryId is required")
	}
	cfg, err := s.ebayAppRuntime(ctx)
	if err != nil {
		return nil, err
	}
	token, err := platformebay.ApplicationToken(ctx, cfg)
	if err != nil {
		return nil, err
	}
	client := platformebay.Client{Config: cfg}
	treeID, err := client.DefaultCategoryTreeID(ctx, token.AccessToken, cfg.Marketplace)
	if err != nil {
		return nil, err
	}
	aspects, _, err := client.CategoryAspects(ctx, token.AccessToken, treeID, categoryID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	rows := make([]PlatformCategoryAttribute, 0, len(aspects))
	for _, aspect := range aspects {
		options, _ := json.Marshal(aspect.Values)
		raw, _ := json.Marshal(aspect)
		row := PlatformCategoryAttribute{Platform: "ebay", CategoryID: categoryID, AttrID: aspect.Name, Name: aspect.Name, Required: aspect.Required, ValueType: aspect.Mode, Options: datatypes.JSON(options), Raw: datatypes.JSON(raw), SyncedAt: &now}
		if err := s.DB.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "platform"}, {Name: "category_id"}, {Name: "attr_id"}}, DoUpdates: clause.AssignmentColumns([]string{"name", "required", "value_type", "options", "raw", "synced_at", "updated_at"})}).Create(&row).Error; err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (s *Service) ListEbayCategoryAspects(ctx context.Context, categoryID string) ([]PlatformCategoryAttribute, error) {
	var rows []PlatformCategoryAttribute
	err := s.DB.WithContext(ctx).Where("platform=? AND category_id=?", "ebay", strings.TrimSpace(categoryID)).Order("required DESC, name ASC").Find(&rows).Error
	return rows, err
}
