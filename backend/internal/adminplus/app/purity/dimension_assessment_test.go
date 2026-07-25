package purity

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/attribution"
	"github.com/stretchr/testify/require"
)

func TestBuildDimensionMatrixAlwaysReturnsStableTwelveDimensions(t *testing.T) {
	report := &PublicReport{
		Provider: ProviderOpenAI,
		Checks: []CheckResult{
			passCheck("model_identity", "模型身份", 10, "ok", nil),
			{ID: "streaming", Name: "流结构", Status: CheckStatusPass, Score: 2, MaxScore: 3, Message: "ok"},
			passCheck("wrapper_fingerprint", "协议与包装指纹", 5, "evidence", nil),
		},
	}

	dimensions := buildDimensionMatrix(report)

	require.Len(t, dimensions, 12)
	require.Equal(t, []string{
		"tag_check", "stream_structure", "non_stream", "websearch", "signature_proto", "output_config",
		"server_tool", "token_inject", "knowledge", "doc_recognition", "image_recognition", "fingerprint",
	}, dimensionIDs(dimensions))
	require.Equal(t, 7, dimensions[1].Score, "2/3 of a 10-point dimension must be rounded")
	require.False(t, dimensions[11].Scored)
	require.Zero(t, dimensions[11].Score)
	require.Contains(t, dimensions[11].Limitations, "gateway_fingerprint_not_protocol_score")
}

func TestBuildDimensionMatrixRepresentsNotRunAndBedrockLimits(t *testing.T) {
	report := &PublicReport{
		Provider: ProviderAnthropic,
		ChannelAttribution: &attribution.Result{
			Channel: "aws_bedrock",
			Status:  attribution.StatusIdentified,
		},
	}

	dimensions := buildDimensionMatrix(report)

	require.Equal(t, dimensionStatusUnsupportedByUpstream, dimensionByID(t, dimensions, "websearch").Status)
	for _, id := range []string{"token_inject", "knowledge", "doc_recognition"} {
		dimension := dimensionByID(t, dimensions, id)
		require.Equal(t, dimensionStatusNotRun, dimension.Status)
		require.False(t, dimension.Scored)
		require.NotEmpty(t, dimension.Limitations)
	}
}

func TestBuildAssessmentClassifiesChannelsAndConflicts(t *testing.T) {
	tests := []struct {
		name          string
		channel       string
		channelStatus string
		wrapper       []string
		identity      string
		wantKind      string
	}{
		{name: "anthropic native", channel: "anthropic_native", channelStatus: attribution.StatusIdentified, identity: CheckStatusPass, wantKind: assessmentKindOfficialNative},
		{name: "bedrock direct", channel: "aws_bedrock", channelStatus: attribution.StatusIdentified, identity: CheckStatusPass, wantKind: assessmentKindOfficialCloud},
		{name: "bedrock transparent relay", channel: "aws_bedrock", channelStatus: attribution.StatusIdentified, wrapper: []string{"cloudflare"}, identity: CheckStatusPass, wantKind: assessmentKindTransparentRelay},
		{name: "vertex", channel: "google_vertex", channelStatus: attribution.StatusIdentified, identity: CheckStatusPass, wantKind: assessmentKindOfficialCloud},
		{name: "unknown compatible", channel: "unknown", channelStatus: attribution.StatusUnknown, identity: CheckStatusPass, wantKind: assessmentKindCompatible},
		{name: "identity conflict", channel: "anthropic_native", channelStatus: attribution.StatusIdentified, identity: CheckStatusFail, wantKind: assessmentKindIdentityConflict},
		{name: "channel conflict", channel: "unknown", channelStatus: attribution.StatusConflicted, identity: CheckStatusPass, wantKind: assessmentKindChannelConflicted},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := assessmentFixture(test.channel, test.channelStatus, test.identity)
			report.WrapperSignals = test.wrapper
			result := buildAssessment(report)
			require.Equal(t, test.wantKind, result.Kind)
			require.Equal(t, "not_tested", result.MeteringStatus)
			require.Contains(t, result.ReasonCodes, "token_audit_not_requested")
			require.Contains(t, result.Summary, "Token 用量异常审计：未启用")
		})
	}
}

func assessmentFixture(channel string, channelStatus string, identityStatus string) *PublicReport {
	return &PublicReport{
		Provider:           ProviderAnthropic,
		Status:             RunStatusDone,
		ModelID:            "claude-opus-4-8",
		ExpectedModel:      "claude-opus-4-8",
		ResponseModel:      "claude-opus-4-8",
		ProtocolScore:      100,
		ModelIdentity:      &ModelIdentityResult{Status: identityStatus, RequestedModel: "claude-opus-4-8", ResponseModel: "claude-opus-4-8"},
		ChannelAttribution: &attribution.Result{Channel: channel, Status: channelStatus, Confidence: 0.95},
		DimensionMatrix:    []DimensionResult{{ID: "tag_check", Name: "LLM 指纹验证", Status: dimensionStatusPass, Scored: true}},
	}
}

func dimensionIDs(dimensions []DimensionResult) []string {
	ids := make([]string, len(dimensions))
	for index := range dimensions {
		ids[index] = dimensions[index].ID
	}
	return ids
}

func dimensionByID(t *testing.T, dimensions []DimensionResult, id string) DimensionResult {
	t.Helper()
	for _, dimension := range dimensions {
		if dimension.ID == id {
			return dimension
		}
	}
	t.Fatalf("dimension %q not found", id)
	return DimensionResult{}
}
