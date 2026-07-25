package purity

import "github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/attribution"

const (
	scorePolicyCompatibleProtocol = "compatible_protocol"
	scorePolicyAnthropicNative    = "anthropic_native_messages"
	scorePolicyAWSBedrock         = "aws_bedrock_messages"
	scorePolicyGoogleVertex       = "google_vertex_claude"
	scorePolicyOpenAINative       = "openai_responses_native"
	scorePolicyGoogleAIStudio     = "google_ai_studio_native"
)

type scorePolicyDefinition struct {
	ID                 string
	Channel            string
	Baseline           string
	Dimensions         []ScorePolicyDimension
	ExcludedDimensions []string
}

func resolveScorePolicy(report *PublicReport) ScorePolicyResult {
	channel := scoringChannel(report)
	policy := compatibleScorePolicy(channel)
	switch channel {
	case "anthropic_native":
		policy = nativeAnthropicScorePolicy()
	case "aws_bedrock":
		policy = awsBedrockScorePolicy()
	case "google_vertex":
		policy = googleVertexScorePolicy()
	case "openai_native":
		policy = nativeOpenAIScorePolicy()
	case "google_ai_studio":
		policy = nativeGoogleAIScorePolicy()
	}
	return ScorePolicyResult{
		ID:                 policy.ID,
		Channel:            policy.Channel,
		Baseline:           policy.Baseline,
		Dimensions:         append([]ScorePolicyDimension(nil), policy.Dimensions...),
		ExcludedDimensions: append([]string(nil), policy.ExcludedDimensions...),
	}
}

func scoringChannel(report *PublicReport) string {
	if report != nil && report.ChannelAttribution != nil && report.ChannelAttribution.Status == attribution.StatusIdentified {
		return report.ChannelAttribution.Channel
	}
	return ""
}

func compatibleScorePolicy(channel string) scorePolicyDefinition {
	return scorePolicyDefinition{
		ID:       scorePolicyCompatibleProtocol,
		Channel:  channel,
		Baseline: "protocol_compatibility",
		Dimensions: []ScorePolicyDimension{
			{ID: "tag_check", ValidationID: "llm_fingerprint", MaxScore: 10},
			{ID: "structure", ValidationID: "schema_integrity", MaxScore: 20},
			{ID: "behavior", ValidationID: "behavior", MaxScore: 30},
			{ID: "signature_proto", ValidationID: "signature", MaxScore: 30},
			{ID: "multimodal", ValidationID: "multimodal", MaxScore: 10},
		},
		ExcludedDimensions: []string{"websearch", "fingerprint"},
	}
}

func nativeAnthropicScorePolicy() scorePolicyDefinition {
	policy := compatibleScorePolicy("anthropic_native")
	policy.ID = scorePolicyAnthropicNative
	policy.Baseline = "anthropic_messages"
	return policy
}

func awsBedrockScorePolicy() scorePolicyDefinition {
	return scorePolicyDefinition{
		ID:       scorePolicyAWSBedrock,
		Channel:  "aws_bedrock",
		Baseline: "aws_bedrock_messages",
		Dimensions: []ScorePolicyDimension{
			{ID: "tag_check", ValidationID: "llm_fingerprint", MaxScore: 10},
			{ID: "structure", ValidationID: "schema_integrity", MaxScore: 25},
			{ID: "behavior", ValidationID: "behavior", MaxScore: 35},
			{ID: "signature_proto", ValidationID: "signature", MaxScore: 20},
			{ID: "multimodal", ValidationID: "multimodal", MaxScore: 10},
		},
		ExcludedDimensions: []string{"websearch", "fingerprint"},
	}
}

func googleVertexScorePolicy() scorePolicyDefinition {
	policy := awsBedrockScorePolicy()
	policy.ID = scorePolicyGoogleVertex
	policy.Channel = "google_vertex"
	policy.Baseline = "google_vertex_claude"
	return policy
}

func nativeOpenAIScorePolicy() scorePolicyDefinition {
	policy := compatibleScorePolicy("openai_native")
	policy.ID = scorePolicyOpenAINative
	policy.Baseline = "openai_responses"
	return policy
}

func nativeGoogleAIScorePolicy() scorePolicyDefinition {
	policy := compatibleScorePolicy("google_ai_studio")
	policy.ID = scorePolicyGoogleAIStudio
	policy.Baseline = "google_generate_content"
	return policy
}
