package purity

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/signature"
	"github.com/stretchr/testify/require"
)

func TestSignatureFingerprintDetailsNeverExposeInternalDedupHash(t *testing.T) {
	details := signatureFingerprintDetails([]signature.Fingerprint{
		{
			DecodedLengthBucket: "256-511",
			TopLevelFields:      []int{2, 3},
			EnvelopeFields:      []int{1, 2, 3, 4, 5},
			MetadataFields:      []int{1, 2, 3, 5, 6, 7, 8},
			DedupHash:           "private-dedup-hash",
		},
	}, "claude-opus-4-8")

	raw, err := json.Marshal(details)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "private-dedup-hash")
	require.NotContains(t, string(raw), "dedup_hash")
	require.Contains(t, string(raw), "decoded_length_bucket")
}
