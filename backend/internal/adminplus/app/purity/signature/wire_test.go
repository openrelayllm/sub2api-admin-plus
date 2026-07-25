package signature

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnalyzeAndClassifyCalibratedFamilies(t *testing.T) {
	registry, err := DefaultRegistry()
	require.NoError(t, err)

	tests := []struct {
		name     string
		metadata []int
		channel  string
		riskCode string
	}{
		{name: "anthropic native", metadata: []int{1, 3, 5, 6, 7, 8, 11}, channel: "anthropic_native"},
		{name: "aws bedrock", metadata: []int{1, 2, 3, 5, 6, 7, 8}, channel: "aws_bedrock"},
		{name: "aws bedrock with anthropic mask", metadata: []int{1, 2, 3, 5, 6, 7, 8, 11}, channel: "aws_bedrock", riskCode: "bedrock_anthropic_signature_mask"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded := syntheticSignature(test.metadata)
			fingerprint, analyzeErr := Analyze(encoded)
			require.NoError(t, analyzeErr)
			require.Equal(t, []int{2, 3}, fingerprint.TopLevelFields)
			require.Equal(t, []int{1, 2, 3, 4, 5}, fingerprint.EnvelopeFields)
			require.Equal(t, test.metadata, fingerprint.MetadataFields)

			classification := registry.Classify(fingerprint, "claude-opus-4-8")
			require.Equal(t, ClassificationIdentified, classification.Status)
			require.Equal(t, test.channel, classification.Channel)
			if test.riskCode != "" {
				require.GreaterOrEqual(t, classification.SampleCount, 1)
				require.Contains(t, classification.RiskCodes, test.riskCode)
			} else {
				require.GreaterOrEqual(t, classification.SampleCount, 3)
				require.Empty(t, classification.RiskCodes)
			}
		})
	}
}

func TestClassifyCalibratedFamiliesKeepsChannelCasesDistinct(t *testing.T) {
	registry, err := DefaultRegistry()
	require.NoError(t, err)

	tests := []struct {
		name     string
		metadata []int
		channel  string
		status   string
	}{
		{name: "standard bedrock", metadata: []int{1, 2, 3, 5, 6, 7, 8}, channel: "aws_bedrock", status: ClassificationIdentified},
		{name: "masked bedrock", metadata: []int{1, 2, 3, 5, 6, 7, 8, 11}, channel: "aws_bedrock", status: ClassificationIdentified},
		{name: "anthropic native", metadata: []int{1, 3, 5, 6, 7, 8, 11}, channel: "anthropic_native", status: ClassificationIdentified},
		{name: "unknown family", metadata: []int{1, 3, 5, 6, 7, 8, 9}, status: ClassificationUnknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fingerprint, analyzeErr := Analyze(syntheticSignature(test.metadata))
			require.NoError(t, analyzeErr)
			classification := registry.Classify(fingerprint, "claude-opus-4-8")
			require.Equal(t, test.status, classification.Status)
			require.Equal(t, test.channel, classification.Channel)
		})
	}
}

func TestAnalyzeUnknownFamilyDoesNotClaimAChannel(t *testing.T) {
	registry, err := DefaultRegistry()
	require.NoError(t, err)

	fingerprint, err := Analyze(syntheticSignature([]int{1, 3, 5, 6, 7, 8, 9}))
	require.NoError(t, err)
	classification := registry.Classify(fingerprint, "claude-opus-4-8")

	require.Equal(t, ClassificationUnknown, classification.Status)
	require.Empty(t, classification.Channel)
}

func TestAnalyzeClaudeJSONKeepsOnlyStructuralFingerprints(t *testing.T) {
	encoded := syntheticSignature([]int{1, 2, 3, 5, 6, 7, 8})
	body, err := json.Marshal(map[string]any{
		"content": []map[string]any{
			{"type": "thinking", "thinking": "redacted", "signature": encoded},
			{"type": "text", "text": "379?"},
		},
	})
	require.NoError(t, err)

	analysis := AnalyzeClaudeJSON(body)
	require.Equal(t, 1, analysis.Found)
	require.Zero(t, analysis.ParseErrors)
	require.Len(t, analysis.Fingerprints, 1)

	publicJSON, err := json.Marshal(analysis.Fingerprints[0])
	require.NoError(t, err)
	require.NotContains(t, string(publicJSON), encoded)
	require.NotContains(t, string(publicJSON), analysis.Fingerprints[0].DedupHash)
}

func TestAnalyzeRejectsMalformedInput(t *testing.T) {
	tests := []string{
		"",
		"not-base64",
		base64.StdEncoding.EncodeToString([]byte{0x00}),
		base64.StdEncoding.EncodeToString([]byte{0x08, 0x80}),
		base64.StdEncoding.EncodeToString([]byte{0x0f}),
		base64.StdEncoding.EncodeToString([]byte{0x12, 0x08, 0x01}),
	}
	for _, encoded := range tests {
		_, err := Analyze(encoded)
		require.Error(t, err)
	}
}

func FuzzAnalyzeNeverPanics(f *testing.F) {
	f.Add(syntheticSignature([]int{1, 2, 3, 5, 6, 7, 8}))
	f.Add("not-base64")
	f.Fuzz(func(t *testing.T, encoded string) {
		_, _ = Analyze(encoded)
	})
}

func syntheticSignature(metadataFields []int) string {
	metadata := make([]byte, 0, 128)
	for _, number := range metadataFields {
		switch number {
		case 5:
			metadata = appendBytesField(metadata, number, []byte{0xff, 0xfe, 0xfd, 0xfc})
		case 6:
			metadata = appendBytesField(metadata, number, []byte("claude-opus-4-8"))
		case 8:
			metadata = appendBytesField(metadata, number, []byte("thinking"))
		case 11:
			metadata = appendBytesField(metadata, number, []byte("00000000-0000-0000-0000-000000000000"))
		default:
			metadata = appendVarintField(metadata, number, 1)
		}
	}
	envelope := appendBytesField(nil, 1, metadata)
	for _, number := range []int{2, 3, 4, 5} {
		envelope = appendVarintField(envelope, number, uint64(number))
	}
	root := appendBytesField(nil, 2, envelope)
	root = appendVarintField(root, 3, 1)
	return base64.StdEncoding.EncodeToString(root)
}

func appendVarintField(dst []byte, number int, value uint64) []byte {
	dst = appendVarint(dst, uint64(number<<3))
	return appendVarint(dst, value)
}

func appendBytesField(dst []byte, number int, value []byte) []byte {
	dst = appendVarint(dst, uint64(number<<3|2))
	dst = appendVarint(dst, uint64(len(value)))
	return append(dst, value...)
}

func appendVarint(dst []byte, value uint64) []byte {
	for value >= 0x80 {
		dst = append(dst, byte(value)|0x80)
		value >>= 7
	}
	return append(dst, byte(value))
}
