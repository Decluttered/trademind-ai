package monitoring

import (
	"fmt"
	"math"
	"strings"
	"time"
)

type PlanInput struct {
	SourcePriceCents int64
	OldPriceCents    int64
	Availability     string
	RiskHigh         bool
	LastAppliedAt    *time.Time
	Now              time.Time
	Rule             PriceRule
}

type PlanResult struct {
	TargetPriceCents    int64
	LandedCostCents     int64
	ExpectedFeeCents    int64
	ExpectedMarginCents int64
	ExpectedMarginBPS   int64
	Outcome             DecisionOutcome
	AutoEligible        bool
	Reason              string
}

func Plan(in PlanInput) (PlanResult, error) {
	if in.SourcePriceCents < 0 || in.OldPriceCents <= 0 || in.Rule.TargetMarginBasisPoints < 0 || in.Rule.TargetMarginBasisPoints >= 10_000 || in.Rule.MinMarginBasisPoints < 0 || in.Rule.PlatformFeeBasisPoints < 0 || in.Rule.PlatformFeeBasisPoints >= 10_000 {
		return PlanResult{}, fmt.Errorf("invalid cents or basis-point input")
	}
	landed := in.SourcePriceCents + in.Rule.ShippingCents + in.Rule.ReserveCents
	denominator := int64(10_000) - in.Rule.TargetMarginBasisPoints - in.Rule.PlatformFeeBasisPoints
	if denominator <= 0 || landed > math.MaxInt64/10_000 {
		return PlanResult{}, fmt.Errorf("target margin and fee rule cannot produce a finite price")
	}
	target := (landed*10_000 + denominator - 1) / denominator
	if in.Rule.MaxPriceCents != nil && target > *in.Rule.MaxPriceCents {
		target = *in.Rule.MaxPriceCents
	}
	fee := (target*in.Rule.PlatformFeeBasisPoints + 9_999) / 10_000
	margin := target - landed - fee
	marginBPS := int64(0)
	if target > 0 {
		marginBPS = margin * 10_000 / target
	}
	result := PlanResult{TargetPriceCents: target, LandedCostCents: landed, ExpectedFeeCents: fee, ExpectedMarginCents: margin, ExpectedMarginBPS: marginBPS}
	if margin < 0 || marginBPS < in.Rule.MinMarginBasisPoints {
		result.Outcome, result.Reason = OutcomeBlockedMargin, "Mindestmarge wird unterschritten"
		return result, nil
	}
	if in.RiskHigh || !available(in.Availability) {
		result.Outcome, result.Reason = OutcomeBlockedPolicy, "Quelle ist nicht ausreichend verfügbar oder als hohes Risiko markiert"
		return result, nil
	}
	if in.LastAppliedAt != nil && in.Rule.CooldownSeconds > 0 && in.Now.Before(in.LastAppliedAt.Add(time.Duration(in.Rule.CooldownSeconds)*time.Second)) {
		result.Outcome, result.Reason = OutcomeBlockedCooldown, "Cooldown ist noch aktiv"
		return result, nil
	}
	delta := target - in.OldPriceCents
	if delta < 0 {
		delta = -delta
	}
	if in.Rule.MaxDeltaCents > 0 && delta > in.Rule.MaxDeltaCents {
		result.Outcome, result.Reason = OutcomeProposed, "Preisänderung überschreitet den absoluten Auto-Apply-Grenzwert"
		return result, nil
	}
	if in.Rule.MaxDeltaBasisPoints > 0 && delta*10_000/in.OldPriceCents > in.Rule.MaxDeltaBasisPoints {
		result.Outcome, result.Reason = OutcomeProposed, "Preisänderung überschreitet den relativen Auto-Apply-Grenzwert"
		return result, nil
	}
	if target == in.OldPriceCents {
		result.Outcome, result.Reason = OutcomeNoChange, "Zielpreis entspricht dem aktuellen Preis"
		return result, nil
	}
	result.Outcome = OutcomeProposed
	result.AutoEligible = in.Rule.AutoApply
	result.Reason = "Guardrails erfüllt; Preisänderung ist anwendbar"
	return result, nil
}

func available(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return false
	}
	return !strings.Contains(normalized, "nicht verfügbar") && !strings.Contains(normalized, "unavailable") && !strings.Contains(normalized, "ausverkauft") && !strings.Contains(normalized, "out of stock")
}
