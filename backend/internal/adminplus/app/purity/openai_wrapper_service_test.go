package purity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServiceRunPublicCheck_OpenAIWrapperObfuscationSignalsAreNotOfficial(t *testing.T) {
	auditResponseIndex := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer sk-wrapper", r.Header.Get("Authorization"))
		w.Header().Set("X-CPA-SUPPORT-PLUGIN", "true")
		w.Header().Set("X-New-Api-Version", "9.99.0")
		w.Header().Set("X-Relay-Provider", "openai-compatible-kimi")
		switch r.URL.Path {
		case "/v1/models":
			writeJSON(t, w, map[string]any{
				"object": "list",
				"data": []map[string]any{
					{"id": "gpt-5.4", "object": "model"},
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
			if _, ok := body["prompt_cache_key"].(string); ok {
				writeOpenAITokenAuditTestResponse(t, w, body, &auditResponseIndex)
				return
			}
			if openAIStoreIncludeProbeRequest(body) {
				writeOpenAITextResponse(t, w, "resp_store_include", "gpt-5.4", "ok")
				return
			}
			writeJSON(t, w, map[string]any{
				"id":     "resp_1",
				"object": "response",
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

	report, err := service.RunPublicCheck(context.Background(), PublicCheckInput{
		Provider:   ProviderOpenAI,
		APIBaseURL: server.URL,
		APIKey:     "sk-wrapper",
		ModelID:    "gpt-5.4",
		ClientIP:   "203.0.113.10",
	})
	require.NoError(t, err)
	require.Equal(t, VerdictOpenAICompatible, report.Verdict)
	require.NotEqual(t, VerdictOfficialOpenAI, report.Verdict)
	require.Contains(t, report.WrapperSignals, "cliproxyapi")
	require.Contains(t, report.WrapperSignals, "new-api")
	require.Contains(t, report.WrapperSignals, "openai-compatible")
	require.Contains(t, report.WrapperSignals, "kimi")
	require.Equal(t, CheckStatusFail, findCheck(t, report, "wrapper_fingerprint").Status)
	require.LessOrEqual(t, report.Score, 65)
	require.Contains(t, report.Summary, "包装/中转")
}

func TestServiceRunPublicCheck_DetectsCLIProxyAPIHeaderFingerprint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			writeJSON(t, w, map[string]any{"message": "CLI Proxy API Server"})
			return
		}
		require.Equal(t, "Bearer sk-cliproxy", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("X-CPA-SUPPORT-PLUGIN", "true")
			writeJSON(t, w, map[string]any{
				"object": "list",
				"data": []map[string]any{
					{"id": "gpt-5.4", "object": "model"},
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
			writeJSON(t, w, map[string]any{
				"id":     "resp_1",
				"object": "response",
				"model":  "gpt-5.4",
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

	report, err := service.RunPublicCheck(context.Background(), PublicCheckInput{
		Provider:       ProviderOpenAI,
		APIBaseURL:     server.URL,
		APIKey:         "sk-cliproxy",
		ModelID:        "gpt-5.4",
		ClientIP:       "203.0.113.10",
		SkipTokenAudit: true,
	})
	require.NoError(t, err)
	require.Equal(t, VerdictOfficialOpenAI, report.Verdict)
	require.Contains(t, report.WrapperSignals, "cliproxyapi")
	require.Equal(t, CheckStatusPass, findCheck(t, report, "wrapper_fingerprint").Status)
	require.GreaterOrEqual(t, report.Score, 85)
	require.Contains(t, report.Summary, "透明中转/兼容网关")
}
