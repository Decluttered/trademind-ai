package collectruleai

import (
	"context"
	"strings"

	"github.com/trademind-ai/trademind/backend/internal/modules/collect"
	"github.com/trademind-ai/trademind/backend/internal/pkg/collectdomain"
)

type platformBlockError struct {
	RecommendedProvider string
	Message             string
}

func (e *platformBlockError) Error() string {
	if e != nil && strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	return "CUSTOM_COLLECT_PROVIDER_CONFLICT"
}

func plannedPlatformHint(platform collectdomain.PlatformID) string {
	switch platform {
	case collectdomain.PlatformAmazonDE:
		return "请使用「Amazon.de 采集器」采集该链接。"
	default:
		return "当前只支持 Amazon.de 作为采集源。"
	}
}

func checkPlatformForAIGenerate(ctx context.Context, resolver ProviderResolver, urlStr string) (plannedHint string, blockErr error) {
	if resolver == nil {
		return "", nil
	}
	host := collectdomain.HostnameFromURL(urlStr)
	platform, ok := collectdomain.DetectPlatform(host)
	if !ok {
		return "", nil
	}
	source := collectdomain.ProviderSourceForPlatform(platform)
	provs := resolver.ResolveCollectProviders(ctx)
	var p *collect.CollectProviderDTO
	for i := range provs {
		if strings.EqualFold(provs[i].Source, source) {
			p = &provs[i]
			break
		}
	}
	if p == nil {
		return plannedPlatformHint(platform), nil
	}
	status := strings.TrimSpace(strings.ToLower(p.Status))
	if status == "available" || status == "beta" {
		msg := customCollectConflictMessage(platform, p.Name)
		return "", &platformBlockError{RecommendedProvider: source, Message: msg}
	}
	return plannedPlatformHint(platform), nil
}

func customCollectConflictMessage(platform collectdomain.PlatformID, providerName string) string {
	name := strings.TrimSpace(providerName)
	switch platform {
	case collectdomain.PlatformAmazonDE:
		return "该链接属于 Amazon.de，请使用「Amazon.de 采集器」。专用采集器会保留 ASIN、币种和 raw 快照。"
	default:
		if name != "" {
			return "该链接属于已配置专用采集器的平台，请使用「" + name + "」。"
		}
		return "该链接属于已配置专用采集器的平台，请使用对应专用采集器。"
	}
}
