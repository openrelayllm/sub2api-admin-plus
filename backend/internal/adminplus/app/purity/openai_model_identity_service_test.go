package purity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServiceRunPublicCheck_OpenAIModelHeaderOverridesSpoofedBodyModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer sk-header", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/v1/models":
			writeJSON(t, w, map[string]any{
				"object": "list",
				"data": []map[string]any{
					{"id": "gpt-5.5", "object": "model"},
				},
			})
		case "/v1/responses":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			if stream, _ := body["stream"].(bool); stream {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = fmt.Fprintln(w, `data: {"type":"response.output_text.delta","delta":"ok"}`)
				_, _ = fmt.Fprintln(w)
				_, _ = fmt.Fprintln(w, `data: {"type":"response.completed","response":{"status":"completed"}}`)
				_, _ = fmt.Fprintln(w)
				_, _ = fmt.Fprintln(w, "data: [DONE]")
				return
			}
			if openAIStoreIncludeProbeRequest(body) {
				writeOpenAITextResponse(t, w, "resp_store_include", "gpt-5.5", "ok")
				return
			}
			w.Header().Set("OpenAI-Model", "gpt-5.4-mini")
			writeJSON(t, w, map[string]any{
				"id":     "resp_header_spoof",
				"object": "response",
				"model":  "gpt-5.5",
				"output": []map[string]any{
					{"type": "function_call", "name": "probe_ping", "arguments": `{"ok":true}`},
				},
				"usage": map[string]any{
					"input_tokens":  8,
					"output_tokens": 3,
					"total_tokens":  11,
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
	service.now = func() time.Time { return time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC) }

	report, err := service.RunDeveloperAPICheck(context.Background(), PublicCheckInput{
		Provider:       ProviderOpenAI,
		APIBaseURL:     server.URL,
		APIKey:         "sk-header",
		ModelID:        "gpt-5.5",
		ClientIP:       "203.0.113.20",
		SkipTokenAudit: true,
	})
	require.NoError(t, err)
	require.Equal(t, "gpt-5.4-mini", report.ResponseModel)
	require.Equal(t, "openai-model", report.ResponseModelSource)
	require.Equal(t, CheckStatusFail, findValidation(t, report, "model_identity").Status)
	require.Equal(t, modelIdentityReasonVersionDowngrade, report.ModelIdentity.Reason)
	require.Equal(t, "gpt-5.5", report.ModelIdentity.Evidence["response_body_model"])
	require.Equal(t, "gpt-5.4-mini", report.ModelIdentity.Evidence["openai_model_header"])
	require.Equal(t, "openai-model", report.ModelIdentity.Evidence["response_model_source"])
	require.Equal(t, VerdictOpenAICompatible, report.Verdict)
	require.LessOrEqual(t, report.OfficialScore, 50)
}
