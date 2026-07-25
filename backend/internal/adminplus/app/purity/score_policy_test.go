package purity

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/attribution"
	"github.com/stretchr/testify/require"
)

func TestDeepSeekOfficialEndpointUsesCompatibleBaselineAndCanScoreOneHundred(t *testing.T) {
	report := &PublicReport{
		Provider: ProviderOpenAI,
		ChannelAttribution: &attribution.Result{
			Channel:    "deepseek",
			Status:     attribution.StatusIdentified,
			Confidence: 0.99,
		},
	}

	policy := resolveScorePolicy(report)
	require.Equal(t, scorePolicyCompatibleProtocol, policy.ID)
	require.Equal(t, "deepseek", policy.Channel)

	report.ScorePolicy = policy
	for _, dimension := range policy.Dimensions {
		report.Validations = append(report.Validations, ValidationResult{
			ID:     dimension.ValidationID,
			Status: CheckStatusPass,
		})
	}
	breakdown := scoreBreakdown(report)

	total := 0
	for _, score := range breakdown {
		total += score
	}
	require.Equal(t, 100, total)
	require.Equal(t, 100, officialScoreFromBreakdown(report, breakdown, 0))
}
