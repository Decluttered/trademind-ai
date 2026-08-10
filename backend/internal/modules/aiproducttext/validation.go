package aiproducttext

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/modules/product"
)

func parseProductIDs(raw []string) ([]uuid.UUID, error) {
	seen := map[uuid.UUID]struct{}{}
	out := make([]uuid.UUID, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		id, err := uuid.Parse(item)
		if err != nil {
			return nil, fmt.Errorf("商品 ID 无效")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("请至少选择一个商品")
	}
	return out, nil
}

func normalizeOperationTypes(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("请选择优化内容")
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, operation := range raw {
		operation = strings.TrimSpace(strings.ToLower(operation))
		switch operation {
		case OpTitle, OpDescription:
		default:
			return nil, fmt.Errorf("不支持的处理类型")
		}
		if _, ok := seen[operation]; ok {
			continue
		}
		seen[operation] = struct{}{}
		out = append(out, operation)
	}
	sort.Strings(out)
	return out, nil
}

func contentHash(content string) string {
	hash := sha256.Sum256([]byte(strings.TrimSpace(content)))
	return hex.EncodeToString(hash[:])
}

func promptTitle(item *product.Product) string {
	if item == nil {
		return ""
	}
	if originalTitle := strings.TrimSpace(item.OriginalTitle); originalTitle != "" {
		return originalTitle
	}
	return strings.TrimSpace(item.Title)
}
