package purity

import "github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/attribution"

const (
	scorePolicyCompatibleProtocol = "compatible_protocol"
	scorePolicyAnthropicNative    = "anthropic_native_messages"
	scorePolicyAWSBedrock         = "aws_bedrock_messages"
	scorePolicyGoogleVertex       = "google_vertex_claude"
	scorePolicyOpenAINative       = "openai_responses_native"
	scorePolicyGoogleAIStudio     = "google_ai_studio_native"
	scoreFailureFullDimension     = "full_dimension_deduction"
	clientImpactNone              = "none"
	clientImpactLimited           = "limited"
	clientImpactBreaking          = "breaking"
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
			scorePolicyDimension("tag_check", "llm_fingerprint", 10, clientImpactBreaking),
			scorePolicyDimension("structure", "schema_integrity", 20, clientImpactBreaking),
			scorePolicyDimension("behavior", "behavior", 30, clientImpactLimited),
			scorePolicyDimension("signature_proto", "signature", 30, clientImpactLimited),
			scorePolicyDimension("multimodal", "multimodal", 10, clientImpactLimited),
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
			scorePolicyDimension("tag_check", "llm_fingerprint", 10, clientImpactBreaking),
			scorePolicyDimension("structure", "schema_integrity", 25, clientImpactBreaking),
			scorePolicyDimension("behavior", "behavior", 35, clientImpactLimited),
			scorePolicyDimension("signature_proto", "signature", 20, clientImpactLimited),
			scorePolicyDimension("multimodal", "multimodal", 10, clientImpactLimited),
		},
		ExcludedDimensions: []string{"websearch", "fingerprint"},
	}
}

func scorePolicyDimension(id string, validationID string, maxScore int, clientImpact string) ScorePolicyDimension {
	return ScorePolicyDimension{
		ID:            id,
		ValidationID:  validationID,
		MaxScore:      maxScore,
		ClientImpact:  clientImpact,
		FailurePolicy: scoreFailureFullDimension,
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
