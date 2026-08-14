package config

import (
	"strings"
	"testing"
)

func TestProductionCapabilityValidateZeroValueUsesSafeL0Defaults(t *testing.T) {
	t.Parallel()

	if err := (ProductionCapabilityConfig{}).Validate(EnvProduction); err != nil {
		t.Fatalf("zero-value production capability config should remain safely disabled: %v", err)
	}
}

func TestProductionCapabilityValidateBlankLevelCannotEnableRealCapabilities(t *testing.T) {
	t.Parallel()

	err := (ProductionCapabilityConfig{RealPlatformNetworkEnabled: true}).Validate(EnvDevelopment)
	if err == nil || !strings.Contains(err.Error(), ErrCodeProductionBoundaryViolation) {
		t.Fatalf("blank level must not bypass the L0 capability boundary: %v", err)
	}
}

func TestProductionCapabilityValidateL3RequiresCompleteDraftWriteControls(t *testing.T) {
	t.Parallel()

	err := (ProductionCapabilityConfig{CurrentAllowedLevel: "L3", RealProviderEnabled: true}).Validate(EnvProduction)
	if err == nil || !strings.Contains(err.Error(), ErrCodeProductionBoundaryViolation) {
		t.Fatalf("partial L3 configuration must fail closed: %v", err)
	}

	cfg := ProductionCapabilityConfig{
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

func TestConfigValidateRequiresPublishQueueForProductionDraftWrites(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		AppEnv:                 EnvDevelopment,
		DB:                     DBConfig{Driver: "postgres", User: "test", Name: "test"},
		StorageProvider:        "local",
		ProductionCapabilities: enabledDraftWriteConfig(),
		WorkerReaperEnabled:    true,
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "PRODUCT_PUBLISH_QUEUE_ENABLED") {
		t.Fatalf("production draft writes must require the publish queue: %v", err)
	}
	cfg.ProductPublishQueueEnabled = true
	cfg.WorkerReaperEnabled = false
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "WORKER_REAPER_ENABLED") {
		t.Fatalf("production draft writes must require the task reaper: %v", err)
	}
}

func enabledDraftWriteConfig() ProductionCapabilityConfig {
	return ProductionCapabilityConfig{
		CurrentAllowedLevel: "L3", RealProviderEnabled: true, RealPlatformNetworkEnabled: true,
		RealCredentialsEnabled: true, RealProductDraftWriteEnabled: true, BackgroundWorkerEnabled: true,
	}
}
