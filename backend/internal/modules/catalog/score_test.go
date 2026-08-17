package catalog

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func f(v float64) *float64 { return &v }
func TestOpportunityScoreIsExplainable(t *testing.T) {
	got := CalculateOpportunityScore(ScoreFactors{Margin: f(100), Demand: f(50), Competition: f(50), Availability: f(100), ListingQuality: f(100), Risk: f(0)})
	require.Equal(t, 83, got.Score)
	require.Equal(t, 100, got.Confidence)
	require.Equal(t, ScoreRuleVersion, got.RuleVersion)
}
func TestMissingFactorsLowerConfidence(t *testing.T) {
	got := CalculateOpportunityScore(ScoreFactors{Availability: f(100)})
	require.Equal(t, 15, got.Score)
	require.Equal(t, 17, got.Confidence)
}
