package purity

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestModelIdentityDetectsVersionAndTierDowngrade(t *testing.T) {
	report := &PublicReport{
		Provider:      ProviderOpenAI,
		ModelID:       "gpt5.5",
		ExpectedModel: "gpt5.5",
		ResponseModel: "gpt-5.4-mini",
	}
	check := buildModelIdentityCheck(report)
	require.Equal(t, CheckStatusFail, check.Status)
	require.NotNil(t, report.ModelIdentity)
	require.Equal(t, modelIdentityReasonVersionDowngrade, report.ModelIdentity.Reason)
	require.Equal(t, "openai", report.ModelIdentity.RequestedVendor)
	require.Equal(t, "openai", report.ModelIdentity.ResponseVendor)
	require.Contains(t, check.Message, "降级")

	claudeReport := &PublicReport{
		Provider:      ProviderAnthropic,
		ModelID:       "claude-opus-4-8",
		ExpectedModel: "claude-opus-4-8",
		ResponseModel: "claude-opus-4-6",
	}
	claudeCheck := buildModelIdentityCheck(claudeReport)
	require.Equal(t, CheckStatusFail, claudeCheck.Status)
	require.Equal(t, modelIdentityReasonVersionDowngrade, claudeReport.ModelIdentity.Reason)
}
func TestModelIdentityDetectsCrossVendorAndWrapperAlias(t *testing.T) {
	report := &PublicReport{
		Provider:      ProviderAnthropic,
		ModelID:       "claude-sonnet-latest",
		ExpectedModel: "claude-sonnet-latest",
		ResponseModel: "glm-5.2",
	}
	check := buildModelIdentityCheck(report)
	require.Equal(t, CheckStatusFail, check.Status)
	require.Equal(t, modelIdentityReasonCrossVendorAlias, report.ModelIdentity.Reason)
	require.Equal(t, "anthropic", report.ModelIdentity.RequestedVendor)
	require.Equal(t, "glm", report.ModelIdentity.ResponseVendor)

	forceMappedReport := &PublicReport{
		Provider:       ProviderAnthropic,
		ModelID:        "claude-opus-4-8",
		ExpectedModel:  "claude-opus-4-8",
		ResponseModel:  "claude-opus-4-8",
		WrapperSignals: []string{"antigravity"},
	}
	forceMappedCheck := buildModelIdentityCheck(forceMappedReport)
	require.Equal(t, CheckStatusFail, forceMappedCheck.Status)
	require.Equal(t, modelIdentityReasonWrapperVendorSignalMismatch, forceMappedReport.ModelIdentity.Reason)
	require.Equal(t, "google", forceMappedReport.ModelIdentity.Evidence["suspected_upstream_vendor"])
}
func TestModelIdentityDetectsProtocolVendorMismatchForceMappingAlias(t *testing.T) {
	report := &PublicReport{
		Provider:      ProviderOpenAI,
		ModelID:       "claude-opus-4.66",
		ExpectedModel: "claude-opus-4.66",
		ResponseModel: "claude-opus-4.66",
	}
	check := buildModelIdentityCheck(report)
	require.Equal(t, CheckStatusFail, check.Status)
	require.Equal(t, modelIdentityReasonProtocolVendorMismatch, report.ModelIdentity.Reason)
	require.Equal(t, "anthropic", report.ModelIdentity.RequestedVendor)
	require.Equal(t, "openai", report.ModelIdentity.Evidence["protocol_expected_vendor"])

	claudeReport := &PublicReport{
		Provider:      ProviderAnthropic,
		ModelID:       "gpt-5.5",
		ExpectedModel: "gpt-5.5",
		ResponseModel: "gpt-5.5",
	}
	claudeCheck := buildModelIdentityCheck(claudeReport)
	require.Equal(t, CheckStatusFail, claudeCheck.Status)
	require.Equal(t, modelIdentityReasonProtocolVendorMismatch, claudeReport.ModelIdentity.Reason)
	require.Equal(t, "openai", claudeReport.ModelIdentity.RequestedVendor)
	require.Equal(t, "anthropic", claudeReport.ModelIdentity.Evidence["protocol_expected_vendor"])
}
func TestModelIdentityDetectsUnexpectedReasoningTokens(t *testing.T) {
	report := &PublicReport{
		Provider:      ProviderOpenAI,
		ModelID:       "gpt-5.4",
		ExpectedModel: "gpt-5.4",
		ResponseModel: "gpt-5.4",
		Metrics: PublicCheckMetrics{
			Usage: &TokenUsage{InputTokens: 10, OutputTokens: 3, TotalTokens: 13, ReasoningTokens: 2},
		},
	}
	check := buildModelIdentityCheck(report)
	require.Equal(t, CheckStatusFail, check.Status)
	require.Equal(t, modelIdentityReasonReasoningTokensMismatch, report.ModelIdentity.Reason)
	require.Equal(t, int64(2), report.ModelIdentity.Evidence["reasoning_tokens"])

	codexReport := &PublicReport{
		Provider:      ProviderOpenAI,
		ModelID:       "gpt-5-codex",
		ExpectedModel: "gpt-5-codex",
		ResponseModel: "gpt-5-codex",
		Metrics: PublicCheckMetrics{
			Usage: &TokenUsage{InputTokens: 10, OutputTokens: 3, TotalTokens: 13, ReasoningTokens: 2},
		},
	}
	codexCheck := buildModelIdentityCheck(codexReport)
	require.Equal(t, CheckStatusPass, codexCheck.Status)
}
