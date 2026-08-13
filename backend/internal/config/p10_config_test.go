package config

import (
	"strings"
	"testing"
)

func TestP10ValidateZeroValueUsesSafeL0Defaults(t *testing.T) {
	t.Parallel()

	if err := (P10Config{}).Validate(EnvProduction); err != nil {
		t.Fatalf("zero-value P10 config should remain safely disabled: %v", err)
	}
}

func TestP10ValidateBlankLevelCannotEnableRealCapabilities(t *testing.T) {
	t.Parallel()

	err := (P10Config{RealPlatformNetworkEnabled: true}).Validate(EnvDevelopment)
	if err == nil || !strings.Contains(err.Error(), ErrCodeP10BoundaryViolation) {
		t.Fatalf("blank level must not bypass the L0 capability boundary: %v", err)
	}
}

func TestP10ValidateL3RequiresCompleteDraftWriteControls(t *testing.T) {
	t.Parallel()

	err := (P10Config{CurrentAllowedLevel: "L3", RealProviderEnabled: true}).Validate(EnvProduction)
	if err == nil || !strings.Contains(err.Error(), ErrCodeP10BoundaryViolation) {
		t.Fatalf("partial L3 configuration must fail closed: %v", err)
	}

	cfg := P10Config{
		CurrentAllowedLevel:          "L3",
		RealProviderEnabled:          true,
		RealPlatformNetworkEnabled:   true,
		RealCredentialsEnabled:       true,
		RealProductDraftWriteEnabled: true,
		BackgroundWorkerEnabled:      true,
	}
	if err := cfg.Validate(EnvProduction); err != nil {
		t.Fatalf("complete L3 draft-write configuration should validate: %v", err)
	}
	cfg.AutomaticRetryEnabled = true
	if err := cfg.Validate(EnvProduction); err == nil {
		t.Fatal("real draft writes must reject automatic retry")
	}
}

func TestConfigValidateRequiresPublishQueueForP10DraftWrites(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		AppEnv:              EnvDevelopment,
		DB:                  DBConfig{Driver: "postgres", User: "test", Name: "test"},
		StorageProvider:     "local",
		P10:                 enabledP10DraftWriteConfig(),
		WorkerReaperEnabled: true,
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "PRODUCT_PUBLISH_QUEUE_ENABLED") {
		t.Fatalf("P10 draft writes must require the publish queue: %v", err)
	}
	cfg.ProductPublishQueueEnabled = true
	cfg.WorkerReaperEnabled = false
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "WORKER_REAPER_ENABLED") {
		t.Fatalf("P10 draft writes must require the task reaper: %v", err)
	}
}

func enabledP10DraftWriteConfig() P10Config {
	return P10Config{
		CurrentAllowedLevel: "L3", RealProviderEnabled: true, RealPlatformNetworkEnabled: true,
		RealCredentialsEnabled: true, RealProductDraftWriteEnabled: true, BackgroundWorkerEnabled: true,
	}
}
