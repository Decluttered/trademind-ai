package monitoring

import (
	"testing"
	"time"
)

func TestPlanUsesCentsAndMarksEligibleChange(t *testing.T) {
	result, err := Plan(PlanInput{SourcePriceCents: 2_000, OldPriceCents: 3_500, Availability: "Auf Lager", Now: time.Unix(1_700_000_000, 0), Rule: PriceRule{TargetMarginBasisPoints: 2_000, MinMarginBasisPoints: 1_500, PlatformFeeBasisPoints: 1_200, ShippingCents: 300, ReserveCents: 100, MaxDeltaCents: 1_000, MaxDeltaBasisPoints: 3_000, AutoApply: true}})
	if err != nil {
		t.Fatal(err)
	}
	if result.TargetPriceCents != 3_530 || !result.AutoEligible || result.Outcome != OutcomeProposed {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestPlanBlocksMarginAfterMaxPrice(t *testing.T) {
	max := int64(2_500)
	result, err := Plan(PlanInput{SourcePriceCents: 2_300, OldPriceCents: 2_500, Availability: "Auf Lager", Now: time.Now(), Rule: PriceRule{TargetMarginBasisPoints: 2_000, MinMarginBasisPoints: 1_000, PlatformFeeBasisPoints: 1_200, MaxPriceCents: &max}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeBlockedMargin {
		t.Fatalf("expected margin block, got %+v", result)
	}
}

func TestPlanHonorsCooldown(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	last := now.Add(-time.Minute)
	result, err := Plan(PlanInput{SourcePriceCents: 1_000, OldPriceCents: 2_000, Availability: "Auf Lager", Now: now, LastAppliedAt: &last, Rule: PriceRule{TargetMarginBasisPoints: 2_000, MinMarginBasisPoints: 1_000, PlatformFeeBasisPoints: 1_000, CooldownSeconds: 300}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeBlockedCooldown {
		t.Fatalf("expected cooldown block, got %+v", result)
	}
}
