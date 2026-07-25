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
