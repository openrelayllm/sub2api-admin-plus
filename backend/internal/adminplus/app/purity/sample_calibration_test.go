package purity

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type calibrationSamples struct {
	WrapperCases       []wrapperCalibrationCase       `json:"wrapper_cases"`
	ModelIdentityCases []modelIdentityCalibrationCase `json:"model_identity_cases"`
	TokenAuditCases    []tokenAuditCalibrationCase    `json:"token_audit_cases"`
}

type wrapperCalibrationCase struct {
	SampleID         string              `json:"sample_id"`
	SampleKind       string              `json:"sample_kind"`
	SourceAuthorized *bool               `json:"source_authorized,omitempty"`
	SampledAt        string              `json:"sampled_at,omitempty"`
	Name             string              `json:"name"`
	Host             string              `json:"host"`
	Provider         string              `json:"provider"`
	ModelID          string              `json:"model_id"`
	ExpectedModel    string              `json:"expected_model"`
	ResponseModel    string              `json:"response_model"`
	Headers          []map[string]string `json:"headers"`
	Values           []string            `json:"values"`
	WantSignals      []string            `json:"want_signals"`
	DenySignals      []string            `json:"deny_signals"`
	WantObfuscation  bool                `json:"want_obfuscation"`
	WantScoreCap     int                 `json:"want_score_cap"`
}

type modelIdentityCalibrationCase struct {
	SampleID            string   `json:"sample_id"`
	SampleKind          string   `json:"sample_kind"`
	SourceAuthorized    *bool    `json:"source_authorized,omitempty"`
	SampledAt           string   `json:"sampled_at,omitempty"`
	Name                string   `json:"name"`
	Provider            string   `json:"provider"`
	ModelID             string   `json:"model_id"`
	ExpectedModel       string   `json:"expected_model"`
	ResponseModel       string   `json:"response_model"`
	WrapperSignals      []string `json:"wrapper_signals"`
	WantStatus          string   `json:"want_status"`
	WantReason          string   `json:"want_reason"`
	WantSuspectedVendor string   `json:"want_suspected_vendor"`
}

type tokenAuditCalibrationCase struct {
	SampleID            string                        `json:"sample_id"`
	SampleKind          string                        `json:"sample_kind"`
	SourceAuthorized    *bool                         `json:"source_authorized,omitempty"`
	SampledAt           string                        `json:"sampled_at,omitempty"`
	Name                string                        `json:"name"`
	Provider            string                        `json:"provider"`
	ModelID             string                        `json:"model_id"`
	Samples             []tokenAuditCalibrationSample `json:"samples"`
	WantStatus          string                        `json:"want_status"`
	WantSummaryContains string                        `json:"want_summary_contains"`
	WantAnomalies       []string                      `json:"want_anomalies"`
	WantUsableSamples   int                           `json:"want_usable_samples"`
	WantMissingSamples  int                           `json:"want_missing_samples"`
}

type tokenAuditCalibrationSample struct {
	Round                    int     `json:"round"`
	Status                   string  `json:"status"`
	StatusCode               int     `json:"status_code,omitempty"`
	ErrorClass               string  `json:"error_class,omitempty"`
	ErrorMessage             string  `json:"error_message,omitempty"`
	InputTokens              int64   `json:"input_tokens,omitempty"`
	OutputTokens             int64   `json:"output_tokens,omitempty"`
	TotalTokens              int64   `json:"total_tokens,omitempty"`
	OfficialBaselineUSD      float64 `json:"official_baseline_usd,omitempty"`
	ActualCostUSD            float64 `json:"actual_cost_usd,omitempty"`
	CachedTokens             int64   `json:"cached_tokens,omitempty"`
	CachedTokensFieldPresent bool    `json:"cached_tokens_present,omitempty"`
	PromptCacheKey           string  `json:"prompt_cache_key,omitempty"`
	Store                    bool    `json:"store,omitempty"`
	ResponseID               string  `json:"response_id,omitempty"`
	PreviousResponseID       string  `json:"previous_response_id,omitempty"`
	StateLinked              bool    `json:"state_linked,omitempty"`
	RequestMode              string  `json:"request_mode,omitempty"`
}

func TestCalibrationSamples_ContractAndRedaction(t *testing.T) {
	raw := loadCalibrationSamplesRaw(t)
	for _, pattern := range []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]{8,}`),
		regexp.MustCompile(`\bsk-(?:ant-)?[A-Za-z0-9_-]{12,}`),
		regexp.MustCompile(`(?i)\b(cookie|set-cookie)\s*:`),
		regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`),
	} {
		require.NotRegexp(t, pattern, string(raw))
	}

	samples := decodeCalibrationSamples(t, raw)
	seenSampleIDs := map[string]string{}
	for _, tc := range samples.WrapperCases {
		t.Run("wrapper/"+tc.Name, func(t *testing.T) {
			require.NotEmpty(t, tc.Name)
			validateSampleMetadata(t, tc.SampleID, tc.SampleKind, tc.SourceAuthorized, tc.SampledAt)
			requireUniqueSampleID(t, seenSampleIDs, tc.SampleID, "wrapper/"+tc.Name)
			for _, headerSet := range tc.Headers {
				requireNoAuthHeaders(t, headerSet)
			}
		})
	}
	for _, tc := range samples.ModelIdentityCases {
		t.Run("model_identity/"+tc.Name, func(t *testing.T) {
			require.NotEmpty(t, tc.Name)
			validateSampleMetadata(t, tc.SampleID, tc.SampleKind, tc.SourceAuthorized, tc.SampledAt)
			requireUniqueSampleID(t, seenSampleIDs, tc.SampleID, "model_identity/"+tc.Name)
		})
	}
	for _, tc := range samples.TokenAuditCases {
		t.Run("token_audit/"+tc.Name, func(t *testing.T) {
			require.NotEmpty(t, tc.Name)
			validateSampleMetadata(t, tc.SampleID, tc.SampleKind, tc.SourceAuthorized, tc.SampledAt)
			requireUniqueSampleID(t, seenSampleIDs, tc.SampleID, "token_audit/"+tc.Name)
			require.NotEmpty(t, tc.Samples)
			for _, sample := range tc.Samples {
				require.NotContains(t, sample.ErrorMessage, "sk-")
				require.NotContains(t, strings.ToLower(sample.ErrorMessage), "bearer ")
				if sample.Status != CheckStatusPass {
					require.True(t,
						sample.StatusCode > 0 || sample.ErrorClass != "" || sample.ErrorMessage != "",
						"failed token audit sample must expose a displayable failure reason",
					)
				}
			}
		})
	}
}

func TestCalibrationSamples_WrapperFingerprints(t *testing.T) {
	samples := loadCalibrationSamples(t)
	require.NotEmpty(t, samples.WrapperCases)

	for _, tc := range samples.WrapperCases {
		t.Run(tc.Name, func(t *testing.T) {
			report := &PublicReport{
				APIBaseHost:   tc.Host,
				Provider:      firstNonEmptyString(tc.Provider, ProviderOpenAI),
				ModelID:       tc.ModelID,
				ExpectedModel: tc.ExpectedModel,
				ResponseModel: tc.ResponseModel,
			}

			signals := wrapperFingerprintSignalsForReportWithValues(report, tc.Values, tc.Headers...)
			for _, signal := range tc.WantSignals {
				require.Contains(t, signals, signal)
			}
			for _, signal := range tc.DenySignals {
				require.NotContains(t, signals, signal)
			}

			report.WrapperSignals = signals
			require.Equalf(t, tc.WantObfuscation, hasWrapperObfuscationFingerprint(report), "signals=%v", signals)
			if tc.WantScoreCap > 0 {
				require.Equal(t, tc.WantScoreCap, wrapperPurityScoreCap(report))
			}
		})
	}
}

func TestCalibrationSamples_ModelIdentity(t *testing.T) {
	samples := loadCalibrationSamples(t)
	require.NotEmpty(t, samples.ModelIdentityCases)

	for _, tc := range samples.ModelIdentityCases {
		t.Run(tc.Name, func(t *testing.T) {
			report := &PublicReport{
				Provider:       tc.Provider,
				ModelID:        tc.ModelID,
				ExpectedModel:  tc.ExpectedModel,
				ResponseModel:  tc.ResponseModel,
				WrapperSignals: append([]string(nil), tc.WrapperSignals...),
			}

			check := buildModelIdentityCheck(report)
			require.Equal(t, tc.WantStatus, check.Status)
			require.NotNil(t, report.ModelIdentity)
			require.Equal(t, tc.WantReason, report.ModelIdentity.Reason)
			if tc.WantSuspectedVendor != "" {
				require.Equal(t, tc.WantSuspectedVendor, report.ModelIdentity.Evidence["suspected_upstream_vendor"])
			}
		})
	}
}

func TestCalibrationSamples_TokenAudit(t *testing.T) {
	samples := loadCalibrationSamples(t)
	require.NotEmpty(t, samples.TokenAuditCases)

	for _, tc := range samples.TokenAuditCases {
		t.Run(tc.Name, func(t *testing.T) {
			report := &TokenAuditReport{
				Status:      CheckStatusWarn,
				Summary:     "Token 用量审计样本不足。",
				PriceSource: "synthetic calibration sample",
				Samples:     make([]TokenAuditSample, 0, len(tc.Samples)),
			}
			for _, raw := range tc.Samples {
				sample := TokenAuditSample{
					Index:                    raw.Round,
					Round:                    raw.Round,
					Status:                   firstNonEmptyString(raw.Status, CheckStatusFail),
					StatusCode:               raw.StatusCode,
					ErrorClass:               raw.ErrorClass,
					ErrorMessage:             raw.ErrorMessage,
					InputTokens:              raw.InputTokens,
					OutputTokens:             raw.OutputTokens,
					TotalTokens:              raw.TotalTokens,
					CachedTokens:             raw.CachedTokens,
					CacheReadInputTokens:     raw.CachedTokens,
					CachedTokensFieldPresent: raw.CachedTokensFieldPresent,
					OfficialBaselineUSD:      raw.OfficialBaselineUSD,
					ActualCostUSD:            raw.ActualCostUSD,
					BaselineCostUSD:          raw.OfficialBaselineUSD,
					CostUSD:                  raw.ActualCostUSD,
					PromptCacheKey:           raw.PromptCacheKey,
					Store:                    raw.Store,
					ResponseID:               raw.ResponseID,
					PreviousResponseID:       raw.PreviousResponseID,
					StateLinked:              raw.StateLinked,
					RequestMode:              raw.RequestMode,
				}
				if tc.Provider == ProviderOpenAI && sample.RequestMode == "" {
					sample.RequestMode = openAITokenAuditRequestMode(raw.Round)
				}
				if sample.Status == CheckStatusPass && sample.OfficialBaselineUSD > 0 {
					sample.Multiplier = roundRatio(sample.ActualCostUSD / sample.OfficialBaselineUSD)
					sample.Ratio = sample.Multiplier
				}
				report.Samples = append(report.Samples, sample)
			}

			if tc.Provider == ProviderOpenAI {
				finalizeOpenAITokenAudit(report)
			} else {
				finalizeTokenAudit(report)
			}

			require.Equal(t, tc.WantStatus, report.Status)
			require.Contains(t, report.Summary, tc.WantSummaryContains)
			for _, anomaly := range tc.WantAnomalies {
				require.Contains(t, report.Anomalies, anomaly)
			}
			require.Equal(t, tc.WantUsableSamples, countUsableTokenAuditSamples(report.Samples))
			require.Equal(t, tc.WantMissingSamples, len(report.Samples)-countUsableTokenAuditSamples(report.Samples))
		})
	}
}

func loadCalibrationSamples(t *testing.T) calibrationSamples {
	t.Helper()
	return decodeCalibrationSamples(t, loadCalibrationSamplesRaw(t))
}

func loadCalibrationSamplesRaw(t *testing.T) []byte {
	t.Helper()

	data, err := os.ReadFile("testdata/calibration_samples.json")
	require.NoError(t, err)
	return data
}

func countUsableTokenAuditSamples(samples []TokenAuditSample) int {
	count := 0
	for _, sample := range samples {
		if sample.TotalTokens > 0 ||
			sample.InputTokens > 0 ||
			sample.OutputTokens > 0 ||
			sample.CacheCreationTokens > 0 ||
			sample.CachedTokens > 0 ||
			sample.ActualCostUSD > 0 ||
			sample.CostUSD > 0 ||
			sample.OfficialBaselineUSD > 0 ||
			sample.BaselineCostUSD > 0 {
			count++
		}
	}
	return count
}

func decodeCalibrationSamples(t *testing.T, data []byte) calibrationSamples {
	t.Helper()
	var samples calibrationSamples
	require.NoError(t, json.Unmarshal(data, &samples))
	return samples
}

func validateSampleMetadata(t *testing.T, sampleID string, sampleKind string, sourceAuthorized *bool, sampledAt string) {
	t.Helper()

	require.NotEmpty(t, sampleID)
	require.Regexp(t, regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`), sampleID)
	require.Contains(t, []string{"synthetic", "authorized_redacted"}, sampleKind)

	if sourceAuthorized != nil {
		require.True(t, *sourceAuthorized, "real samples must be authorized before entering calibration data")
	}
	if sampleKind == "authorized_redacted" {
		require.NotNil(t, sourceAuthorized)
		require.NotEmpty(t, sampledAt)
		_, err := time.Parse(time.RFC3339, sampledAt)
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(sampleID, "authorized-"), "authorized samples must use an authorized-* sample id")
	} else {
		require.Nil(t, sourceAuthorized)
		require.Empty(t, sampledAt)
	}
}

func requireUniqueSampleID(t *testing.T, seen map[string]string, sampleID string, owner string) {
	t.Helper()

	if previousOwner, ok := seen[sampleID]; ok {
		require.Failf(t, "duplicate calibration sample id", "%s duplicates %s", owner, previousOwner)
	}
	seen[sampleID] = owner
}

func requireNoAuthHeaders(t *testing.T, headers map[string]string) {
	t.Helper()

	for key := range headers {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "authorization", "x-api-key", "x-goog-api-key", "cookie", "set-cookie":
			require.Failf(t, "auth header leaked into calibration sample", "header %q must be redacted", key)
		}
	}
}
