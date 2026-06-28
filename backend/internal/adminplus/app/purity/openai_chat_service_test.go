package purity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServiceRunPublicCheck_ChatCompletionsFallbackProducesUsageAuditRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer sk-chat-only", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/v1/models":
			writeJSON(t, w, map[string]any{
				"object": "list",
				"data":   []map[string]any{{"id": "gpt-5.4", "object": "model"}},
			})
		case "/v1/responses":
			w.WriteHeader(http.StatusNotFound)
			writeJSON(t, w, map[string]any{"error": map[string]any{"message": "responses unsupported"}})
		case "/v1/chat/completions":
			writeJSON(t, w, map[string]any{
				"id":     "chatcmpl-test",
				"object": "chat.completion",
				"model":  "gpt-5.4",
				"choices": []map[string]any{
					{"index": 0, "message": map[string]any{"role": "assistant", "content": "ok"}, "finish_reason": "stop"},
				},
				"usage": map[string]any{
					"prompt_tokens":     11,
					"completion_tokens": 2,
					"total_tokens":      13,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := NewService(nil)
	service.httpClient = server.Client()
	service.allowPrivateHosts = true
	service.limiter = nil
	service.now = func() time.Time { return time.Date(2026, 6, 28, 15, 10, 0, 0, time.UTC) }

	report, err := service.RunPublicCheck(context.Background(), PublicCheckInput{
		Provider:   ProviderOpenAI,
		APIBaseURL: server.URL,
		APIKey:     "sk-chat-only",
		ModelID:    "gpt-5.4",
		ClientIP:   "203.0.113.10",
	})

	require.NoError(t, err)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "chat_completions").Status)
	require.Equal(t, CheckStatusWarn, findCheck(t, report, "token_audit").Status)
	require.NotNil(t, report.TokenAudit)
	require.Equal(t, chatTokenAuditSamples, report.TokenAudit.SampleCount)
	require.Len(t, report.TokenAudit.Rows, chatTokenAuditSamples)
	require.Contains(t, report.TokenAudit.Anomalies, "chat_completions_audit_fallback")
	require.Equal(t, int64(33), report.TokenAudit.InputTokens)
	require.Equal(t, int64(6), report.TokenAudit.OutputTokens)
}
