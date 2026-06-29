package purity

import (
	"context"
	"encoding/json"
	"fmt"
	coreservice "github.com/Wei-Shaw/sub2api/internal/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServiceRunPublicCheck_GeminiNative(t *testing.T) {
	server := newGeminiCompatibleServer(t, "AIza-test", "gemini-3-pro-preview")
	defer server.Close()

	service := NewService(nil)
	service.httpClient = server.Client()
	service.allowPrivateHosts = true
	service.limiter = nil
	service.now = func() time.Time { return time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC) }

	report, err := service.RunPublicCheck(context.Background(), PublicCheckInput{
		Provider:   ProviderGemini,
		APIBaseURL: server.URL,
		APIKey:     "AIza-test",
		ModelID:    "gemini-3-pro-preview",
		ClientIP:   "203.0.113.20",
	})
	require.NoError(t, err)
	require.Equal(t, ProviderGemini, report.Provider)
	require.Equal(t, RunStatusDone, report.Status)
	require.NotEqual(t, VerdictInvalidOrUnavailable, report.Verdict)
	require.Equal(t, "gemini-compatible", report.NonStreamChannel)
	require.Equal(t, "gemini-compatible", report.StreamChannel)
	require.Equal(t, "gemini-3-pro-preview", report.ResponseModel)
	require.Equal(t, int64(9), report.Metrics.Usage.TotalTokens)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "models_schema").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "responses_schema").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "tool_call").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "streaming").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "usage").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "multimodal").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "token_audit").Status)
	require.NotNil(t, report.TokenAudit)
	require.Equal(t, tokenAuditSamples, report.TokenAudit.SampleCount)
	require.Equal(t, tokenAuditSamples-1, report.TokenAudit.HistoryReplayRounds)
	require.Equal(t, tokenAuditSamples-1, report.TokenAudit.HistoryReplayLinks)
	require.True(t, report.TokenAudit.HistoryReplayOK)
	require.True(t, report.TokenAudit.ContextReplayOK)
	require.True(t, report.TokenAudit.CachedTokensFieldObserved)
	require.Greater(t, report.TokenAudit.CachedTokens, int64(0))
	require.Equal(t, CheckStatusPass, findValidation(t, report, "model_identity").Status)
	require.Equal(t, CheckStatusPass, findValidation(t, report, "token_audit").Status)
	require.Equal(t, 100, report.CompatibilityScore)
	require.Equal(t, 100, report.OfficialScore)
	require.Empty(t, report.WrapperSignals)
}

func TestServiceRunPublicCheck_GeminiClientPathPassesWhenEnhancedProbesAreLimited(t *testing.T) {
	server := newGeminiLimitedCompatibleServer(t, "AIza-test", "gemini-3.5-flash")
	defer server.Close()

	service := NewService(nil)
	service.httpClient = server.Client()
	service.allowPrivateHosts = true
	service.limiter = nil
	service.now = func() time.Time { return time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC) }

	report, err := service.RunPublicCheck(context.Background(), PublicCheckInput{
		Provider:   ProviderGemini,
		APIBaseURL: server.URL,
		APIKey:     "AIza-test",
		ModelID:    "gemini-3.5-flash",
		ClientIP:   "203.0.113.20",
	})
	require.NoError(t, err)
	require.Equal(t, RunStatusDone, report.Status)
	require.NotEqual(t, VerdictInvalidOrUnavailable, report.Verdict)
	require.Equal(t, "gemini-3.5-flash", report.ResponseModel)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "responses_schema").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "usage").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "streaming").Status)
	require.Equal(t, CheckStatusFail, findCheck(t, report, "tool_call").Status)
	require.Equal(t, CheckStatusWarn, findCheck(t, report, "multimodal").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "token_audit").Status)
	require.NotNil(t, report.TokenAudit)
	require.Equal(t, tokenAuditSamples, report.TokenAudit.SampleCount)
	require.GreaterOrEqual(t, report.CompatibilityScore, 50)
	require.Empty(t, report.Error)
}

func TestServiceRunPublicCheck_GeminiModelFallbackPreservesRequestedModel(t *testing.T) {
	server := newGeminiFallbackServer(t, "AIza-test", "gemini-3.5-flash")
	defer server.Close()

	service := NewService(nil)
	service.httpClient = server.Client()
	service.allowPrivateHosts = true
	service.limiter = nil
	service.now = func() time.Time { return time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC) }

	report, err := service.RunPublicCheck(context.Background(), PublicCheckInput{
		Provider:   ProviderGemini,
		APIBaseURL: server.URL,
		APIKey:     "AIza-test",
		ModelID:    "gemini-3-pro-preview",
		ClientIP:   "203.0.113.20",
	})
	require.NoError(t, err)
	require.Equal(t, "gemini-3-pro-preview", report.ModelID)
	require.Equal(t, "gemini-3.5-flash", report.ResponseModel)
	require.Equal(t, "body.modelVersion", report.ResponseModelSource)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "responses_schema").Status)
	require.Equal(t, "gemini-3-pro-preview", findCheck(t, report, "responses_schema").Details["requested_model"])
	require.Equal(t, "gemini-3.5-flash", findCheck(t, report, "responses_schema").Details["probe_model"])
	require.Equal(t, CheckStatusPass, findValidation(t, report, "model_identity").Status)
	require.Equal(t, modelIdentityReasonProbeFallback, report.ModelIdentity.Reason)
	require.Equal(t, CheckStatusPass, findValidation(t, report, "schema_integrity").Status)
	require.GreaterOrEqual(t, report.Scores["tag_check"], 9)
	require.Equal(t, 20, report.Scores["structure"])
}

func TestGeminiTokenAuditDoesNotTreatZeroCachedContentAsCacheReadHit(t *testing.T) {
	body := []byte(`{
		"candidates": [{"content": {"parts": [{"text": "ok"}]}}],
		"usageMetadata": {
			"promptTokenCount": 90,
			"candidatesTokenCount": 12,
			"cachedContentTokenCount": 0,
			"totalTokenCount": 102
		}
	}`)

	sample := geminiTokenAuditSampleFromProbe(1, httpProbe{StatusCode: http.StatusOK, Body: body, LatencyMS: 100}, geminiModelPricingFor("gemini-3.5-flash"), 0, geminiTokenAuditModeHistoryReplay)

	require.Equal(t, CheckStatusPass, sample.Status)
	require.Equal(t, int64(0), sample.CachedTokens)
	require.Equal(t, int64(0), sample.CacheReadInputTokens)
	require.True(t, sample.CachedTokensFieldPresent)
	require.Equal(t, int64(90), sample.UncachedInputTokens)
}

func TestGeminiTokenAuditReadsCacheTokensDetails(t *testing.T) {
	body := []byte(`{
		"candidates": [{"content": {"parts": [{"text": "ok"}]}}],
		"usageMetadata": {
			"promptTokenCount": 120,
			"candidatesTokenCount": 12,
			"cacheTokensDetails": [
				{"modality": "TEXT", "tokenCount": 35},
				{"modality": "IMAGE", "token_count": 7}
			],
			"totalTokenCount": 132
		}
	}`)

	sample := geminiTokenAuditSampleFromProbe(2, httpProbe{StatusCode: http.StatusOK, Body: body, LatencyMS: 100}, geminiModelPricingFor("gemini-3.5-flash"), 2, geminiTokenAuditModeHistoryReplay)

	require.Equal(t, CheckStatusPass, sample.Status)
	require.Equal(t, int64(42), sample.CachedTokens)
	require.Equal(t, int64(42), sample.CacheReadInputTokens)
	require.True(t, sample.CachedTokensFieldPresent)
	require.Equal(t, int64(78), sample.UncachedInputTokens)
}

func TestServiceRunAccountCheckStream_GeminiUsesAccountBillingMultiplier(t *testing.T) {
	server := newGeminiCompatibleServer(t, "AIza-account", "gemini-3-pro-preview")
	defer server.Close()
	rateMultiplier := 0.11

	service := NewServiceWithAccountResolver(nil, accountResolverStub{
		account: &coreservice.Account{
			ID:             86,
			Platform:       coreservice.PlatformGemini,
			Type:           coreservice.AccountTypeAPIKey,
			RateMultiplier: &rateMultiplier,
			Credentials: map[string]any{
				"api_key":  "AIza-account",
				"base_url": server.URL,
			},
		},
	})
	service.httpClient = server.Client()
	service.allowPrivateHosts = true
	service.limiter = nil

	report, err := service.RunAccountCheckStream(context.Background(), AccountCheckInput{
		AccountID: 86,
		ModelID:   "gemini-3-pro-preview",
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, report.TokenAudit)
	require.NotNil(t, report.TokenAudit.BillingMultiplier)
	require.InDelta(t, 0.11, *report.TokenAudit.BillingMultiplier, 1e-12)
	require.NotNil(t, report.TokenAudit.BillingMultiplierCompat)
	require.InDelta(t, 0.11, *report.TokenAudit.BillingMultiplierCompat, 1e-12)
	require.Equal(t, "account_config", report.TokenAudit.BillingMultiplierSource)
	require.Equal(t, "account_config", report.TokenAudit.BillingMultiplierSourceCompat)
}

func TestShouldFallbackGeminiModelForNewAPIUnavailableChannel(t *testing.T) {
	probe := httpProbe{
		StatusCode: http.StatusServiceUnavailable,
		Body: []byte(`{
			"error": {
				"message": "No available channel for model gemini-3-pro-preview under group gemini (distributor)"
			}
		}`),
	}

	require.True(t, shouldFallbackGeminiModel(probe))
}

func newGeminiCompatibleServer(t *testing.T, expectedKey string, model string) *httptest.Server {
	t.Helper()
	modelPath := "/v1beta/models/" + model
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, expectedKey, r.Header.Get("x-goog-api-key"))
		switch r.URL.Path {
		case "/v1/usage":
			http.NotFound(w, r)
		case "/v1beta/models":
			writeJSON(t, w, map[string]any{
				"models": []map[string]any{
					{
						"name":                       "models/" + model,
						"supportedGenerationMethods": []string{"generateContent", "streamGenerateContent"},
					},
				},
			})
		case modelPath + ":generateContent":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			if bodyHasGeminiInlineData(body) {
				writeJSON(t, w, geminiTextResponse(model, "ok"))
				return
			}
			if bodyIsGeminiAuditRequest(body) {
				contents, _ := body["contents"].([]any)
				require.True(t, len(contents)%2 == 1)
				round := (len(contents) + 1) / 2
				writeJSON(t, w, geminiAuditTextResponse(model, round))
				return
			}
			if bodyHasGeminiToolDeclaration(body) {
				writeJSON(t, w, geminiToolResponse(model))
				return
			}
			writeJSON(t, w, geminiTextResponse(model, "Hello! How can I help you today?"))
		case modelPath + ":streamGenerateContent":
			require.Equal(t, "sse", r.URL.Query().Get("alt"))
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprintln(w, `data: {"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":4,"candidatesTokenCount":1,"totalTokenCount":5}}`)
			_, _ = fmt.Fprintln(w)
		default:
			http.NotFound(w, r)
		}
	}))
}

func newGeminiLimitedCompatibleServer(t *testing.T, expectedKey string, model string) *httptest.Server {
	t.Helper()
	modelPath := "/v1beta/models/" + model
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, expectedKey, r.Header.Get("x-goog-api-key"))
		switch r.URL.Path {
		case "/v1/usage":
			http.NotFound(w, r)
		case "/v1beta/models":
			writeJSON(t, w, map[string]any{
				"models": []map[string]any{
					{
						"name":                       "models/" + model,
						"supportedGenerationMethods": []string{"generateContent", "streamGenerateContent"},
					},
				},
			})
		case modelPath + ":generateContent":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			if bodyHasGeminiInlineData(body) {
				writeJSONWithStatus(t, w, http.StatusServiceUnavailable, map[string]any{
					"error": map[string]any{
						"code":    "model_not_found",
						"message": "No available channel for inlineData under group gemini",
						"type":    "new_api_error",
					},
				})
				return
			}
			if bodyHasGeminiToolDeclaration(body) {
				writeJSONWithStatus(t, w, http.StatusBadGateway, map[string]any{
					"error": map[string]any{
						"message": "toolConfig is not supported by this compatible gateway",
					},
				})
				return
			}
			writeJSON(t, w, geminiTextResponse(model, "Hello! How can I help you today?"))
		case modelPath + ":streamGenerateContent":
			require.Equal(t, "sse", r.URL.Query().Get("alt"))
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprintln(w, `data: {"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":4,"candidatesTokenCount":1,"totalTokenCount":5}}`)
			_, _ = fmt.Fprintln(w)
		default:
			http.NotFound(w, r)
		}
	}))
}

func newGeminiFallbackServer(t *testing.T, expectedKey string, fallbackModel string) *httptest.Server {
	t.Helper()
	modelPath := "/v1beta/models/" + fallbackModel
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, expectedKey, r.Header.Get("x-goog-api-key"))
		switch r.URL.Path {
		case "/v1beta/models":
			writeJSON(t, w, map[string]any{
				"models": []map[string]any{
					{
						"name":                       "models/" + fallbackModel,
						"supportedGenerationMethods": []string{"generateContent", "streamGenerateContent"},
					},
				},
			})
		case "/v1/usage":
			http.NotFound(w, r)
		case modelPath + ":generateContent":
			writeJSON(t, w, geminiTextResponse(fallbackModel, "ok"))
		case modelPath + ":streamGenerateContent":
			require.Equal(t, "sse", r.URL.Query().Get("alt"))
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprintln(w, `data: {"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":4,"candidatesTokenCount":1,"totalTokenCount":5}}`)
			_, _ = fmt.Fprintln(w)
		default:
			http.NotFound(w, r)
		}
	}))
}

func geminiToolResponse(model string) map[string]any {
	return map[string]any{
		"modelVersion": model,
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"role": "model",
					"parts": []map[string]any{
						{"functionCall": map[string]any{
							"name": "probe_ping",
							"args": map[string]any{"ok": true},
						}},
					},
				},
				"finishReason": "STOP",
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":     9,
			"candidatesTokenCount": 2,
			"totalTokenCount":      11,
		},
	}
}

func geminiTextResponse(model string, text string) map[string]any {
	return map[string]any{
		"modelVersion": model,
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"role": "model",
					"parts": []map[string]any{
						{"text": text},
					},
				},
				"finishReason": "STOP",
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":     8,
			"candidatesTokenCount": 1,
			"totalTokenCount":      9,
		},
	}
}
func geminiAuditTextResponse(model string, round int) map[string]any {
	inputTokens := int64(80 + round*10)
	outputTokens := int64(12 + round)
	cachedTokens := int64(0)
	if round > 1 {
		cachedTokens = int64(20 + round)
	}
	return map[string]any{
		"modelVersion": model,
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"role": "model",
					"parts": []map[string]any{
						{"text": geminiAuditAssistantMemory(round, "test")},
					},
				},
				"finishReason": "STOP",
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":        inputTokens,
			"candidatesTokenCount":    outputTokens,
			"cachedContentTokenCount": cachedTokens,
			"thoughtsTokenCount":      1,
			"totalTokenCount":         inputTokens + outputTokens + 1,
		},
	}
}
func bodyHasGeminiInlineData(body map[string]any) bool {
	contents, _ := body["contents"].([]any)
	for _, rawContent := range contents {
		content, _ := rawContent.(map[string]any)
		parts, _ := content["parts"].([]any)
		for _, rawPart := range parts {
			part, _ := rawPart.(map[string]any)
			if _, ok := part["inlineData"]; ok {
				return true
			}
		}
	}
	return false
}
func bodyIsGeminiAuditRequest(body map[string]any) bool {
	systemInstruction, ok := body["systemInstruction"].(map[string]any)
	if !ok {
		return false
	}
	parts, _ := systemInstruction["parts"].([]any)
	for _, rawPart := range parts {
		part, _ := rawPart.(map[string]any)
		text, _ := part["text"].(string)
		if strings.Contains(text, "Gemini CLI token audit") {
			return true
		}
	}
	return false
}

func bodyHasGeminiToolDeclaration(body map[string]any) bool {
	tools, ok := body["tools"].([]any)
	return ok && len(tools) > 0
}
