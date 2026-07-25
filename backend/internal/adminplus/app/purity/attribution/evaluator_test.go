package attribution

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/signature"
	"github.com/stretchr/testify/require"
)

func TestEvaluateIdentifiesBedrockBehindTransparentWrapper(t *testing.T) {
	result := Evaluate(Input{
		Provider:       "anthropic",
		Host:           "relay.example.com",
		Model:          "claude-opus-4-8",
		CheckedAt:      time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC),
		WrapperSignals: []string{"new-api"},
		Signatures: []SignatureObservation{
			{
				Transport: "stream",
				Classification: signature.Classification{
					Channel:     "aws_bedrock",
					Status:      signature.ClassificationIdentified,
					Confidence:  0.96,
					FamilyID:    "aws-bedrock-2026-07",
					SampleCount: 4,
					SourceType:  "authorized_redacted",
				},
			},
		},
		SignatureFound: 1,
	})

	require.Equal(t, StatusIdentified, result.Status)
	require.Equal(t, "aws_bedrock", result.Channel)
	require.InDelta(t, 0.96, result.Confidence, 0.001)
	require.Len(t, result.Evidence, 1)
	require.Empty(t, result.Contradictions)
}

func TestEvaluateIdentifiesBedrockAndPreservesAnthropicMaskRisk(t *testing.T) {
	classification := signature.Classification{
		Channel:     "aws_bedrock",
		Status:      signature.ClassificationIdentified,
		Confidence:  0.96,
		FamilyID:    "aws-bedrock-anthropic-mask-2026-07",
		SampleCount: 1,
		SourceType:  "authorized_redacted",
		RiskCodes:   []string{"bedrock_anthropic_signature_mask"},
	}
	result := Evaluate(Input{
		Provider:       "anthropic",
		Host:           "relay.example.com",
		Model:          "claude-opus-4-8",
		WrapperSignals: []string{"new-api"},
		Signatures: []SignatureObservation{
			{Transport: "non_stream", Classification: classification},
			{Transport: "stream", Classification: classification},
		},
		SignatureFound: 2,
	})

	require.Equal(t, StatusIdentified, result.Status)
	require.Equal(t, "aws_bedrock", result.Channel)
	require.Contains(t, result.ReasonCodes, "bedrock_anthropic_signature_mask")
	require.Len(t, result.Evidence, 2)
	for _, evidence := range result.Evidence {
		require.Equal(t, "signature_family_match_with_anthropic_mask", evidence.Code)
		require.Contains(t, evidence.RiskCodes, "bedrock_anthropic_signature_mask")
	}
}

func TestEvaluateReportsStreamNonStreamConflict(t *testing.T) {
	result := Evaluate(Input{
		Provider: "anthropic",
		Host:     "relay.example.com",
		Signatures: []SignatureObservation{
			{Transport: "stream", Classification: signature.Classification{Channel: "aws_bedrock", Status: signature.ClassificationIdentified, Confidence: 0.96}},
			{Transport: "non_stream", Classification: signature.Classification{Channel: "anthropic_native", Status: signature.ClassificationIdentified, Confidence: 0.96}},
		},
		SignatureFound: 2,
	})

	require.Equal(t, StatusConflicted, result.Status)
	require.Equal(t, "unknown", result.Channel)
	require.NotEmpty(t, result.Evidence)
	require.NotEmpty(t, result.Contradictions)
}

func TestEvaluateUnknownCompatibleEndpointIsNotAFailure(t *testing.T) {
	result := Evaluate(Input{Provider: "anthropic", Host: "relay.example.com"})

	require.Equal(t, StatusUnknown, result.Status)
	require.Equal(t, "anthropic_compatible", result.Channel)
	require.Zero(t, result.Confidence)
}

func TestEvaluateSingleHeaderFamilyStaysLikely(t *testing.T) {
	result := Evaluate(Input{
		Provider: "anthropic",
		Host:     "relay.example.com",
		HeaderSets: []map[string]string{{
			"x-amzn-requestid": "redacted",
			"x-amzn-trace-id":  "redacted",
		}},
	})

	require.Equal(t, StatusLikely, result.Status)
	require.Equal(t, "aws_bedrock", result.Channel)
	require.Less(t, result.Confidence, 0.9)
	require.Len(t, result.Evidence, 1)
}

func TestEvaluateOfficialVertexEndpoint(t *testing.T) {
	result := Evaluate(Input{Provider: "anthropic", Host: "us-central1-aiplatform.googleapis.com"})

	require.Equal(t, StatusIdentified, result.Status)
	require.Equal(t, "google_vertex", result.Channel)
	require.InDelta(t, 0.99, result.Confidence, 0.001)
}

func TestEvaluateKnownLanguageModelEndpoints(t *testing.T) {
	tests := []struct {
		host    string
		channel string
		kind    string
	}{
		{host: "api.deepseek.com", channel: "deepseek", kind: ChannelKindOfficialNative},
		{host: "dashscope.aliyuncs.com", channel: "alibaba_bailian", kind: ChannelKindOfficialNative},
		{host: "api.moonshot.cn", channel: "moonshot", kind: ChannelKindOfficialNative},
		{host: "open.bigmodel.cn", channel: "zhipu_bigmodel", kind: ChannelKindOfficialNative},
		{host: "ark.cn-beijing.volces.com", channel: "volcengine_ark", kind: ChannelKindOfficialNative},
		{host: "api.x.ai", channel: "xai", kind: ChannelKindOfficialNative},
		{host: "api.z.ai", channel: "zai_coding", kind: ChannelKindOfficialNative},
		{host: "api.kimi.com", channel: "kimi_coding", kind: ChannelKindOfficialNative},
		{host: "chatgpt.com", channel: "openai_codex_subscription", kind: ChannelKindOfficialNative},
		{host: "tenant.openai.azure.com", channel: "azure_openai", kind: ChannelKindOfficialCloud},
		{host: "api.cloudflare.com", channel: "cloudflare_workers_ai", kind: ChannelKindOfficialCloud},
		{host: "openrouter.ai", channel: "openrouter", kind: ChannelKindAggregator},
		{host: "api.dify.ai", channel: "dify", kind: ChannelKindAggregator},
		{host: "api.coze.cn", channel: "coze", kind: ChannelKindAggregator},
		{host: "fastgpt.run", channel: "fastgpt", kind: ChannelKindAggregator},
		{host: "llm.submodel.ai", channel: "submodel", kind: ChannelKindAggregator},
		{host: "api.aiproxy.io", channel: "aiproxy", kind: ChannelKindAggregator},
	}

	for _, test := range tests {
		t.Run(test.channel, func(t *testing.T) {
			result := Evaluate(Input{Provider: "openai", Host: test.host})

			require.Equal(t, StatusIdentified, result.Status)
			require.Equal(t, test.channel, result.Channel)
			require.Equal(t, test.kind, ChannelKind(result.Channel))
			require.InDelta(t, 0.99, result.Confidence, 0.001)
		})
	}
}

func TestEvaluateDoesNotTreatLookalikeCloudHostAsOfficial(t *testing.T) {
	result := Evaluate(Input{Provider: "openai", Host: "bedrock.amazonaws.com.attacker.example"})

	require.Equal(t, StatusUnknown, result.Status)
	require.Equal(t, "openai_compatible", result.Channel)
}
