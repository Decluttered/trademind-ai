package collect

import (
	"context"
	"strconv"
	"strings"
)

// BatchSourcePolicy controls throttling and retry behavior for bulk collect by source.
type BatchSourcePolicy struct {
	Source          string
	Concurrency     int
	DelayMinMs      int
	DelayMaxMs      int
	RetryOnBlocked  bool
	RetryOnTimeout  bool
	MaxRetries      int
	BatchRetryBoost bool // longer backoff for batch transient failures
}

func (p BatchSourcePolicy) ThrottleEnabled() bool {
	return p.Concurrency > 0 || p.DelayMaxMs > 0
}

func (s *Service) env1688BatchPolicy() BatchSourcePolicy {
	p := BatchSourcePolicy{Source: "1688"}
	if s == nil {
		return p
	}
	if s.Source1688BatchConcurrency > 0 {
		p.Concurrency = s.Source1688BatchConcurrency
	} else {
		p.Concurrency = 1
	}
	if p.Concurrency > 2 {
		p.Concurrency = 2
	}
	if s.Source1688BatchDelayMinMs > 0 {
		p.DelayMinMs = s.Source1688BatchDelayMinMs
	} else {
		p.DelayMinMs = 1500
	}
	if s.Source1688BatchDelayMaxMs > 0 {
		p.DelayMaxMs = s.Source1688BatchDelayMaxMs
	} else {
		p.DelayMaxMs = 5000
	}
	if s.Source1688BatchDelayMaxMs > 0 && s.Source1688BatchDelayMinMs > s.Source1688BatchDelayMaxMs {
		p.DelayMinMs, p.DelayMaxMs = p.DelayMaxMs, p.DelayMinMs
	}
	p.RetryOnBlocked = true
	p.RetryOnTimeout = true
	if s != nil {
		p.RetryOnBlocked = s.BatchRetryOnBlocked
		p.RetryOnTimeout = s.BatchRetryOnTimeout
	}
	if s.Source1688BatchMaxRetries > 0 {
		p.MaxRetries = s.Source1688BatchMaxRetries
	} else {
		p.MaxRetries = 2
	}
	p.BatchRetryBoost = true
	return p
}

func (s *Service) batchPolicyForSource(ctx context.Context, source string) BatchSourcePolicy {
	src := strings.TrimSpace(source)
	if isPinduoduoCollectSource(src) {
		return s.pinduoduoBatchPolicyFromSettings(ctx)
	}
	if isTaobaoTmallCollectSource(src) {
		return s.taobaoTmallBatchPolicyFromSettings(ctx)
	}
	if !strings.EqualFold(src, "1688") {
		return BatchSourcePolicy{Source: src}
	}
	p := s.env1688BatchPolicy()
	if s == nil || s.Settings == nil {
		return p
	}
	m, err := s.Settings.PlainByGroup(ctx, 0, "collector")
	if err != nil || len(m) == 0 {
		return p
	}
	if v := settingsInt(m, "collect_batch_concurrency_1688"); v > 0 {
		p.Concurrency = v
		if p.Concurrency > 2 {
			p.Concurrency = 2
		}
	}
	if v := settingsInt(m, "collect_batch_delay_min_ms_1688"); v >= 0 {
		p.DelayMinMs = v
	}
	if v := settingsInt(m, "collect_batch_delay_max_ms_1688"); v > 0 {
		p.DelayMaxMs = v
	}
	if p.DelayMinMs > p.DelayMaxMs {
		p.DelayMinMs, p.DelayMaxMs = p.DelayMaxMs, p.DelayMinMs
	}
	if v, ok := settingsBool(m, "collect_batch_retry_on_blocked"); ok {
		p.RetryOnBlocked = v
	}
	if v, ok := settingsBool(m, "collect_batch_retry_on_timeout"); ok {
		p.RetryOnTimeout = v
	}
	if v := settingsInt(m, "collect_batch_max_retries_1688"); v > 0 {
		p.MaxRetries = v
	}
	return p
}

func settingsInt(m map[string]string, key string) int {
	if m == nil {
		return 0
	}
	raw := strings.TrimSpace(m[key])
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return n
}

func settingsBool(m map[string]string, key string) (bool, bool) {
	if m == nil {
		return false, false
	}
	raw := strings.TrimSpace(strings.ToLower(m[key]))
	if raw == "" {
		return false, false
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}
