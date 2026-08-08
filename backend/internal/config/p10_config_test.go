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
