package purity

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
	require.Equal(t, VerdictOfficialGemini, report.Verdict)
	require.Equal(t, "gemini-compatible", report.NonStreamChannel)
	require.Equal(t, "gemini-compatible", report.StreamChannel)
	require.Equal(t, "gemini-3-pro-preview", report.ResponseModel)
	require.Equal(t, int64(11), report.Metrics.Usage.TotalTokens)
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
			writeJSON(t, w, map[string]any{
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
			})
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
	tools, ok := body["tools"].([]any)
	if !ok || len(tools) == 0 {
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
