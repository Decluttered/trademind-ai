package customerchat

import "testing"

func TestDefaultAutoReplyPolicyIsFailClosed(t *testing.T) {
	policy := defaultAutoReplyPolicy(nil)
	if policy.Enabled {
		t.Fatal("automatic replies must default to disabled")
	}
	if !policy.LowRiskOnly || !policy.RequireOrderContext {
		t.Fatalf("expected production guards enabled, got %+v", policy)
	}
	if policy.MaxRepliesPerHour != defaultAutoRepliesPerHour || policy.MaxReplyRunes != defaultAutoReplyMaxRunes {
		t.Fatalf("unexpected safe defaults: %+v", policy)
	}
}

func TestDefaultAutoReplySettingIsFailClosed(t *testing.T) {
	setting := defaultAutoReplySetting(7)
	if setting.MessageSyncEnabled || setting.AutoReplyEnabled {
		t.Fatalf("runtime settings must default to disabled: %+v", setting)
	}
	if setting.PollIntervalSeconds != defaultAutoReplyPollIntervalSeconds {
		t.Fatalf("unexpected poll interval: %d", setting.PollIntervalSeconds)
	}
}

func TestNormalizeAutoReplySettingRequiresMessageSync(t *testing.T) {
	_, err := normalizeAutoReplySetting(UpdateAutoReplySettingBody{
		MessageSyncEnabled: false, AutoReplyEnabled: true, PollIntervalSeconds: 60,
	})
	if err == nil {
		t.Fatal("auto reply must require message sync")
	}
	if _, err := normalizeAutoReplySetting(UpdateAutoReplySettingBody{PollIntervalSeconds: 14}); err == nil {
		t.Fatal("poll intervals below the safety bound must be rejected")
	}
}

func TestNormalizeAutoReplyPolicyRejectsRiskGuardRemoval(t *testing.T) {
	_, err := normalizeAutoReplyPolicy(UpdateAutoReplyPolicyBody{
		Enabled:             true,
		Tone:                "professional",
		MaxReplyRunes:       600,
		MaxRepliesPerHour:   20,
		RequireOrderContext: true,
		LowRiskOnly:         false,
	})
	if err == nil {
		t.Fatal("enabled policy must reject lowRiskOnly=false")
	}
}

func TestNormalizeAutoReplyPolicyAppliesBoundedDefaults(t *testing.T) {
	body, err := normalizeAutoReplyPolicy(UpdateAutoReplyPolicyBody{
		Enabled:             true,
		Tone:                " FRIENDLY ",
		RequireOrderContext: true,
		LowRiskOnly:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if body.Tone != "friendly" || body.MaxReplyRunes != defaultAutoReplyMaxRunes || body.MaxRepliesPerHour != defaultAutoRepliesPerHour {
		t.Fatalf("unexpected normalization: %+v", body)
	}
}

func TestNormalizeAutoReplyPolicyRejectsUnsafeBounds(t *testing.T) {
	cases := []UpdateAutoReplyPolicyBody{
		{Tone: "professional", MaxReplyRunes: 49, MaxRepliesPerHour: 20, LowRiskOnly: true},
		{Tone: "professional", MaxReplyRunes: 600, MaxRepliesPerHour: 101, LowRiskOnly: true},
		{Tone: "unsupported", MaxReplyRunes: 600, MaxRepliesPerHour: 20, LowRiskOnly: true},
	}
	for _, body := range cases {
		if _, err := normalizeAutoReplyPolicy(body); err == nil {
			t.Fatalf("expected validation failure for %+v", body)
		}
	}
}

func TestAutoReplyOutputGuardBlocksFinancialCommitments(t *testing.T) {
	blocked := []string{
		"我们会立即为您退款",
		"We guarantee a refund today.",
		"可以为您安排赔付。",
	}
	for _, reply := range blocked {
		if !autoReplyOutputRequiresHuman(reply) {
			t.Fatalf("expected human review for %q", reply)
		}
	}
	if autoReplyOutputRequiresHuman("您的订单已发货，可在订单详情查看物流进度。") {
		t.Fatal("ordinary logistics response should remain eligible")
	}
}
