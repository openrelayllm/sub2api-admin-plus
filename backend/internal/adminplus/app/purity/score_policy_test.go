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

func TestMaskedBedrockUsesBedrockPolicyAndAppliesAuditablePenalty(t *testing.T) {
	report := &PublicReport{
		Provider: ProviderAnthropic,
		ChannelAttribution: &attribution.Result{
			Channel:     "aws_bedrock",
			Status:      attribution.StatusIdentified,
			Confidence:  0.96,
			ReasonCodes: []string{"channel_evidence_matched", "bedrock_anthropic_signature_mask"},
		},
	}
	report.ScorePolicy = resolveScorePolicy(report)
	require.Equal(t, scorePolicyAWSBedrock, report.ScorePolicy.ID)
	for _, dimension := range report.ScorePolicy.Dimensions {
		report.Validations = append(report.Validations, ValidationResult{ID: dimension.ValidationID, Status: CheckStatusPass})
	}

	score := officialPurityScore(report, 0)
	require.Equal(t, 95, score)
	require.Len(t, report.ScoreAdjustments, 1)
	require.Equal(t, ScoreAdjustment{
		ID:           "bedrock_anthropic_signature_mask_penalty",
		Category:     "provenance_transparency",
		ReasonCode:   "bedrock_anthropic_signature_mask",
		CaseID:       "PURITY-BEDROCK-MASK-001",
		ClientImpact: clientImpactNone,
		ImpactScope:  "channel_attribution_only",
		BaseScore:    100,
		Points:       -5,
		ResultScore:  95,
		Evidence:     []string{"bedrock_metadata_family_present", "anthropic_native_metadata_present"},
	}, report.ScoreAdjustments[0])
	require.True(t, hasWrapperObfuscationFingerprint(report))

	check := buildWrapperFingerprintCheck(report)
	require.Equal(t, CheckStatusFail, check.Status)
	require.Equal(t, "PURITY-BEDROCK-MASK-001", check.Details["case_id"])
	require.Equal(t, 5, check.Details["score_penalty"])
	require.Equal(t, clientImpactNone, check.Details["client_impact"])
}

func TestStandardBedrockCanStillScoreOneHundred(t *testing.T) {
	report := &PublicReport{
		Provider: ProviderAnthropic,
		ChannelAttribution: &attribution.Result{
			Channel: "aws_bedrock",
			Status:  attribution.StatusIdentified,
		},
	}
	report.ScorePolicy = resolveScorePolicy(report)
	for _, dimension := range report.ScorePolicy.Dimensions {
		report.Validations = append(report.Validations, ValidationResult{ID: dimension.ValidationID, Status: CheckStatusPass})
	}

	require.Equal(t, 100, officialPurityScore(report, 0))
	require.Empty(t, report.ScoreAdjustments)
}

func TestFailedClientDimensionDeductsItsFullWeight(t *testing.T) {
	policy := awsBedrockScorePolicy()
	for _, failed := range policy.Dimensions {
		t.Run(failed.ID, func(t *testing.T) {
			report := &PublicReport{Provider: ProviderAnthropic}
			report.ScorePolicy = ScorePolicyResult{
				ID:         policy.ID,
				Channel:    policy.Channel,
				Baseline:   policy.Baseline,
				Dimensions: policy.Dimensions,
			}
			for _, dimension := range policy.Dimensions {
				status := CheckStatusPass
				if dimension.ID == failed.ID {
					status = CheckStatusFail
				}
				report.Validations = append(report.Validations, ValidationResult{ID: dimension.ValidationID, Status: status})
			}

			breakdown := scoreBreakdown(report)
			require.Equal(t, 0, breakdown[failed.ID])
			require.Equal(t, 100-failed.MaxScore, officialScoreFromBreakdown(report, breakdown, 0))
			require.Equal(t, scoreFailureFullDimension, failed.FailurePolicy)
			require.NotEqual(t, clientImpactNone, failed.ClientImpact)
		})
	}
}

func TestClientFailureDoesNotAlsoApplyProvenancePenalty(t *testing.T) {
	report := &PublicReport{
		Provider:       ProviderAnthropic,
		WrapperSignals: []string{"new-api-model-mapping"},
	}
	report.ScorePolicy = resolveScorePolicy(report)
	for _, dimension := range report.ScorePolicy.Dimensions {
		status := CheckStatusPass
		if dimension.ID == "behavior" {
			status = CheckStatusFail
		}
		report.Validations = append(report.Validations, ValidationResult{ID: dimension.ValidationID, Status: status})
	}

	require.Equal(t, 70, officialPurityScore(report, 0))
	require.Empty(t, report.ScoreAdjustments)
}
