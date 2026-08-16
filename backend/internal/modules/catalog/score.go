package catalog

import "math"

const ScoreRuleVersion = "phase1-v1"

type ScoreFactors struct {
	Margin         *float64 `json:"margin,omitempty"`
	Demand         *float64 `json:"demand,omitempty"`
	Competition    *float64 `json:"competition,omitempty"`
	Availability   *float64 `json:"availability,omitempty"`
	ListingQuality *float64 `json:"listingQuality,omitempty"`
	Risk           *float64 `json:"risk,omitempty"`
}

type ScoreResult struct {
	Score       int          `json:"score"`
	Confidence  int          `json:"confidence"`
	RuleVersion string       `json:"ruleVersion"`
	Factors     ScoreFactors `json:"factors"`
}

func CalculateOpportunityScore(f ScoreFactors) ScoreResult {
	weighted, present := 0.0, 0
	add := func(v *float64, weight float64, inverse bool) {
		if v == nil {
			return
		}
		n := math.Max(0, math.Min(100, *v))
		if inverse {
			n = 100 - n
		}
		weighted += n * weight
		present++
	}
	add(f.Margin, .30, false)
	add(f.Demand, .20, false)
	add(f.Competition, .15, false)
	add(f.Availability, .15, false)
	add(f.ListingQuality, .10, false)
	add(f.Risk, .10, true)
	return ScoreResult{Score: int(math.Round(weighted)), Confidence: int(math.Round(float64(present) / 6 * 100)), RuleVersion: ScoreRuleVersion, Factors: f}
}
