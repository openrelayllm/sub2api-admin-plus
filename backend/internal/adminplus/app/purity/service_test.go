package purity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coreservice "github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestNormalizeProvider_ProtocolAliases(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty_defaults_openai", in: "", want: ProviderOpenAI},
		{name: "openai", in: "openai", want: ProviderOpenAI},
		{name: "openai_compatible_dash", in: "OpenAI-Compatible", want: ProviderOpenAI},
		{name: "openai_compatible_space", in: "openai compatible", want: ProviderOpenAI},
		{name: "anthropic", in: "anthropic", want: ProviderAnthropic},
		{name: "claude", in: "Claude", want: ProviderAnthropic},
		{name: "claude_compatible_dash", in: "claude-compatible", want: ProviderAnthropic},
		{name: "anthropic_compatible_space", in: "Anthropic Compatible", want: ProviderAnthropic},
		{name: "gemini", in: "gemini", want: ProviderGemini},
		{name: "gemini_compatible_dash", in: "Gemini-Compatible", want: ProviderGemini},
		{name: "google_ai_studio_space", in: "Google AI Studio", want: ProviderGemini},
		{name: "qwen_remains_unsupported", in: "qwen", want: "qwen"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, normalizeProvider(tc.in))
		})
	}
}

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
	require.Equal(t, CheckStatusWarn, findCheck(t, report, "token_audit").Status)
	require.Nil(t, report.TokenAudit)
	require.Equal(t, CheckStatusPass, findValidation(t, report, "model_identity").Status)
	require.Equal(t, CheckStatusWarn, findValidation(t, report, "token_audit").Status)
	require.Equal(t, 100, report.CompatibilityScore)
	require.Equal(t, 100, report.OfficialScore)
	require.Empty(t, report.WrapperSignals)
}

func TestOfficialPurityScore_TokenAuditWarnUsesReservedWeight(t *testing.T) {
	report := &PublicReport{
		Provider: ProviderOpenAI,
		Status:   RunStatusDone,
		Checks: []CheckResult{
			passCheck("base_url", "API Base 域名", 20, "ok", nil),
			CheckResult{ID: "token_audit", Name: "Token 用量审计", Status: CheckStatusWarn, Score: 8, MaxScore: 15, Message: "Responses 状态链路不完整"},
		},
		Validations: []ValidationResult{
			{ID: "llm_fingerprint", Status: CheckStatusPass},
			{ID: "schema_integrity", Status: CheckStatusPass},
			{ID: "behavior", Status: CheckStatusPass},
			{ID: "signature", Status: CheckStatusPass},
			{ID: "multimodal", Status: CheckStatusPass},
			{ID: "token_audit", Status: CheckStatusWarn},
		},
		TokenAudit: &TokenAuditReport{Status: CheckStatusWarn, Summary: "Responses 状态链路不完整", SampleCount: tokenAuditSamples},
	}

	finalizeReport(report)

	require.Equal(t, 95, report.OfficialScore)
	require.Equal(t, 95, report.Score)
}

func TestOfficialPurityScore_SkippedTokenAuditDoesNotLowerScore(t *testing.T) {
	report := &PublicReport{
		Provider: ProviderGemini,
		Status:   RunStatusDone,
		Checks: []CheckResult{
			passCheck("base_url", "API Base 域名", 20, "ok", nil),
			CheckResult{ID: "token_audit", Name: "Token 用量审计", Status: CheckStatusWarn, Score: 0, MaxScore: 0, Message: "skipped", Details: map[string]any{"skipped": true}},
		},
		Validations: []ValidationResult{
			{ID: "llm_fingerprint", Status: CheckStatusPass},
			{ID: "schema_integrity", Status: CheckStatusPass},
			{ID: "behavior", Status: CheckStatusPass},
			{ID: "signature", Status: CheckStatusPass},
			{ID: "multimodal", Status: CheckStatusPass},
			{ID: "token_audit", Status: CheckStatusWarn},
		},
	}

	finalizeReport(report)

	require.Equal(t, 100, report.OfficialScore)
	require.Equal(t, 100, report.Score)
}

func TestServiceRunPublicCheck_OpenAIPureBehindCustomBaseURL(t *testing.T) {
	auditResponseIndex := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
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
	service.now = func() time.Time { return time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC) }

	report, err := service.RunPublicCheck(context.Background(), PublicCheckInput{
		Provider:   ProviderOpenAI,
		APIBaseURL: server.URL,
		APIKey:     "sk-test",
		ModelID:    "gpt-5.4",
		ClientIP:   "203.0.113.10",
	})
	require.NoError(t, err)
	require.Equal(t, AccessModeWeb, report.AccessMode)
	require.Equal(t, AccessModeWeb, report.AccessModeCompat)
	require.Equal(t, BillingModeCaptchaRateLimit, report.BillingMode)
	require.Equal(t, BillingModeCaptchaRateLimit, report.BillingModeCompat)
	require.Equal(t, VerdictOfficialOpenAI, report.Verdict)
	require.NotEmpty(t, report.ReportID)
	require.Equal(t, 100, report.Score)
	require.Equal(t, 100, report.CompatibilityScore)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "tool_call").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "streaming").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "multimodal").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "token_audit").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "wrapper_fingerprint").Status)
	require.Len(t, report.Validations, 8)
	require.Equal(t, "llm_fingerprint", report.Validations[0].ID)
	require.Equal(t, CheckStatusPass, findValidation(t, report, "behavior").Status)
	require.Equal(t, CheckStatusWarn, findValidation(t, report, "model_identity").Status)
	require.Equal(t, "programmatic_probe", findValidation(t, report, "behavior").Details["detector"])
	require.NotNil(t, report.TokenAudit)
	require.Equal(t, tokenAuditSamples, report.TokenAudit.SampleCount)
	require.Len(t, report.TokenAudit.Samples, tokenAuditSamples)
	require.Len(t, report.TokenAudit.Rows, tokenAuditSamples)
	require.Equal(t, report.TokenAudit.ActualCostUSD, report.TokenAudit.TotalCostUSD)
	require.Equal(t, float64(1), report.TokenAudit.Multiplier)
	require.True(t, report.TokenAudit.PreviousChainOK)
	require.Equal(t, tokenAuditSamples-1, report.TokenAudit.StatefulRounds)
	require.NotZero(t, report.TokenAudit.CachedTokens)
	require.True(t, strings.HasPrefix(report.TokenAudit.PromptCacheKey, "proxyai_best_"))
	require.Empty(t, report.TokenAudit.Samples[0].PreviousResponseID)
	require.Equal(t, "resp_audit_1", report.TokenAudit.Samples[1].PreviousResponseID)

	developerReport, err := service.RunDeveloperAPICheck(context.Background(), PublicCheckInput{
		Provider:       ProviderOpenAI,
		APIBaseURL:     server.URL,
		APIKey:         "sk-test",
		ModelID:        "gpt-5.4",
		ClientIP:       "203.0.113.10",
		SkipTokenAudit: true,
	})
	require.NoError(t, err)
	require.Equal(t, AccessModeDeveloperAPI, developerReport.AccessMode)
	require.Equal(t, AccessModeDeveloperAPI, developerReport.AccessModeCompat)
	require.Equal(t, BillingModeAPIKeyMetered, developerReport.BillingMode)
	require.Equal(t, BillingModeAPIKeyMetered, developerReport.BillingModeCompat)
}

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
	require.Equal(t, CheckStatusWarn, findCheck(t, report, "wrapper_fingerprint").Status)
	require.GreaterOrEqual(t, report.Score, 85)
	require.Contains(t, report.Summary, "透明中转/兼容网关")
}

func TestServiceRunPublicCheck_NonFatalProbeErrorStaysInCheckDetails(t *testing.T) {
	auditResponseIndex := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			if payloadHasInputImage(body) {
				w.WriteHeader(http.StatusInternalServerError)
				writeJSON(t, w, map[string]any{"error": map[string]any{"message": "multimodal upstream timeout"}})
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
	service.now = func() time.Time { return time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC) }

	report, err := service.RunPublicCheck(context.Background(), PublicCheckInput{
		Provider:   ProviderOpenAI,
		APIBaseURL: server.URL,
		APIKey:     "sk-test",
		ModelID:    "gpt-5.4",
		ClientIP:   "203.0.113.10",
	})
	require.NoError(t, err)
	require.Equal(t, RunStatusDone, report.Status)
	require.Empty(t, report.Error)
	require.Empty(t, report.Metrics.ErrorClass)
	require.Empty(t, report.Metrics.ErrorMessage)
	multimodalCheck := findCheck(t, report, "multimodal")
	require.Equal(t, CheckStatusFail, multimodalCheck.Status)
	require.Equal(t, "upstream_5xx", multimodalCheck.Details["error_class"])
	require.Equal(t, "multimodal upstream timeout", multimodalCheck.Details["error_message"])
}

func TestTokenAuditPayloadsUseCumulativeCacheShape(t *testing.T) {
	auditNonce := "audit-test-nonce"
	roundOnePrompt := openAITokenAuditPrompt(1, auditNonce)
	roundElevenPrompt := openAITokenAuditPrompt(11, auditNonce)
	require.Contains(t, roundOnePrompt, "stable-cache-prefix")
	require.Contains(t, roundOnePrompt, auditNonce)
	require.Contains(t, roundOnePrompt, "round-cache-01")
	require.NotContains(t, roundOnePrompt, "round-cache-02")
	require.Contains(t, roundElevenPrompt, "round-cache-11")
	require.Greater(t, len(roundElevenPrompt), len(roundOnePrompt))

	promptCacheKey := openAITokenAuditPromptCacheKey("gpt-5.4", auditNonce)
	var openAIPayload map[string]any
	require.NoError(t, json.Unmarshal(responsesAuditProbePayload("gpt-5.4", 2, auditNonce, "resp_audit_1", promptCacheKey), &openAIPayload))
	require.Equal(t, "gpt-5.4", openAIPayload["model"])
	require.Equal(t, true, openAIPayload["store"])
	require.Equal(t, "resp_audit_1", openAIPayload["previous_response_id"])
	require.Equal(t, promptCacheKey, openAIPayload["prompt_cache_key"])
	require.NotContains(t, openAIPayload, "tool_choice")
	require.NotContains(t, openAIPayload, "parallel_tool_calls")
	require.NotContains(t, openAIPayload, "tools")
	require.Contains(t, openAIPayload["instructions"], "stable-cache-prefix")
	require.NotContains(t, openAIPayload["instructions"], "round-cache-02")
	input, ok := openAIPayload["input"].([]any)
	require.True(t, ok)
	require.Len(t, input, 1)
	inputMessage, ok := input[0].(map[string]any)
	require.True(t, ok)
	content, ok := inputMessage["content"].([]any)
	require.True(t, ok)
	inputBlock, ok := content[0].(map[string]any)
	require.True(t, ok)
	require.Contains(t, inputBlock["text"], "round-cache-02")

	var minimalPayload map[string]any
	require.NoError(t, json.Unmarshal(responsesAuditMinimalProbePayload("gpt-5.4", 2, auditNonce), &minimalPayload))
	require.NotContains(t, minimalPayload, "store")
	require.NotContains(t, minimalPayload, "previous_response_id")
	require.NotContains(t, minimalPayload, "prompt_cache_key")
	require.NotContains(t, minimalPayload, "tools")

	probeCtx := newClaudeProbeContext()
	history := []claudeAuditTurn{
		{Round: 1, UserText: claudeAuditUserText(1, auditNonce), AssistantText: strings.Repeat("x ", 20)},
		{Round: 2, UserText: claudeAuditUserText(2, auditNonce), AssistantText: strings.Repeat("y ", 10)},
	}
	var claudePayload map[string]any
	require.NoError(t, json.Unmarshal(claudeAuditProbePayload(defaultClaudeModel, 3, auditNonce, probeCtx, history), &claudePayload))
	systemBlocks, ok := claudePayload["system"].([]any)
	require.True(t, ok)
	require.Len(t, systemBlocks, 2)
	systemCacheControlled := 0
	for i, rawBlock := range systemBlocks {
		block, _ := rawBlock.(map[string]any)
		if _, ok := block["cache_control"].(map[string]any); ok {
			systemCacheControlled++
			require.Equal(t, 1, i)
		}
	}
	require.Equal(t, 1, systemCacheControlled)
	cachedSystemBlock, ok := systemBlocks[1].(map[string]any)
	require.True(t, ok)
	require.Contains(t, cachedSystemBlock["text"], "stable-cache-prefix")
	require.NotContains(t, cachedSystemBlock["text"], "round-cache-01")
	require.NotContains(t, cachedSystemBlock["text"], "round-cache-03")

	messages, ok := claudePayload["messages"].([]any)
	require.True(t, ok)
	require.Len(t, messages, 5)
	messageLevelCacheControlled := 0
	for i, rawMessage := range messages {
		message, ok := rawMessage.(map[string]any)
		require.True(t, ok)
		content, ok := message["content"].([]any)
		require.True(t, ok)
		require.NotEmpty(t, content)
		block, ok := content[len(content)-1].(map[string]any)
		require.True(t, ok)
		if _, ok := block["cache_control"].(map[string]any); ok {
			messageLevelCacheControlled++
			require.Equal(t, len(messages)-1, i)
		}
	}
	require.Equal(t, 1, messageLevelCacheControlled)
	message, ok := messages[0].(map[string]any)
	require.True(t, ok)
	content, ok = message["content"].([]any)
	require.True(t, ok)
	require.Len(t, content, 1)
	instructionBlock, ok := content[0].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, instructionBlock, "cache_control")
	require.Contains(t, instructionBlock["text"], "round-cache-01")
	currentMessage, ok := messages[4].(map[string]any)
	require.True(t, ok)
	currentContent, ok := currentMessage["content"].([]any)
	require.True(t, ok)
	currentBlock, ok := currentContent[0].(map[string]any)
	require.True(t, ok)
	require.Contains(t, currentBlock["text"], "round-cache-03")
	require.Contains(t, currentBlock["text"], "Purity token audit response target.")

	metadata, ok := claudePayload["metadata"].(map[string]any)
	require.True(t, ok)
	userID, ok := metadata["user_id"].(string)
	require.True(t, ok)
	var parsedUserID map[string]string
	require.NoError(t, json.Unmarshal([]byte(userID), &parsedUserID))
	require.Len(t, parsedUserID["device_id"], 64)
	require.Equal(t, probeCtx.sessionID, parsedUserID["session_id"])
	require.Equal(t, "", parsedUserID["account_uuid"])
}

func TestOfficialPricingBaselines(t *testing.T) {
	openai := openAIModelPricingFor("gpt-5.4")
	require.Equal(t, 2.5e-6, openai.InputPerToken)
	require.Equal(t, 0.25e-6, openai.CacheReadPerToken)
	require.Equal(t, 15e-6, openai.OutputPerToken)
	require.Contains(t, openai.Source, "Official OpenAI API pricing")

	openAIMiniAlias := openAIModelPricingFor("gpt-5.4-mini-preview")
	require.Equal(t, 0.75e-6, openAIMiniAlias.InputPerToken)
	require.Equal(t, 0.075e-6, openAIMiniAlias.CacheReadPerToken)
	require.Equal(t, 4.5e-6, openAIMiniAlias.OutputPerToken)
	require.Contains(t, openAIMiniAlias.Source, "gpt-5.4-mini")

	openAIPro := openAIModelPricingFor("gpt-5.4-pro")
	require.Equal(t, 30e-6, openAIPro.InputPerToken)
	require.Equal(t, 30e-6, openAIPro.CacheReadPerToken)
	require.Equal(t, 180e-6, openAIPro.OutputPerToken)
	require.Contains(t, openAIPro.Source, "cached input not separately listed")

	openAICodexAlias := openAIModelPricingFor("gpt-5.3-codex-spark")
	require.Equal(t, 1.75e-6, openAICodexAlias.InputPerToken)
	require.Equal(t, 0.175e-6, openAICodexAlias.CacheReadPerToken)
	require.Equal(t, 14e-6, openAICodexAlias.OutputPerToken)
	require.Contains(t, openAICodexAlias.Source, "gpt-5.3-codex")

	openAIChatLatest := openAIModelPricingFor("chat-latest")
	require.Equal(t, 5e-6, openAIChatLatest.InputPerToken)
	require.Equal(t, 0.5e-6, openAIChatLatest.CacheReadPerToken)
	require.Equal(t, 30e-6, openAIChatLatest.OutputPerToken)

	openAILegacyCurrent := openAIModelPricingFor("gpt-5.2")
	require.Equal(t, 1.75e-6, openAILegacyCurrent.InputPerToken)
	require.Contains(t, openAILegacyCurrent.Source, "Official OpenAI API pricing")
	require.NotContains(t, openAILegacyCurrent.Source, "fallback")

	opus48 := claudeModelPricingFor("claude-opus-4-8")
	require.Equal(t, 5e-6, opus48.InputPerToken)
	require.Equal(t, 6.25e-6, opus48.CacheWritePerToken)
	require.Equal(t, 0.5e-6, opus48.CacheReadPerToken)
	require.Equal(t, 25e-6, opus48.OutputPerToken)
	require.Contains(t, opus48.Source, "Official Anthropic Claude pricing")
	require.Contains(t, opus48.Source, "5m cache writes")

	legacyOpus := claudeModelPricingFor("claude-opus-4-1")
	require.Equal(t, 15e-6, legacyOpus.InputPerToken)
	require.Equal(t, 75e-6, legacyOpus.OutputPerToken)
}

func TestClaudeTokenAuditUsesCacheAwareCCTestBaseline(t *testing.T) {
	pricing := claudeModelPricingFor("claude-opus-4-8")
	var totalBaseline float64
	for i := 1; i <= tokenAuditSamples; i++ {
		sample := TokenAuditSample{Index: i}
		applyClaudeTokenAuditBaseline(&sample, pricing)
		require.Greater(t, sample.OfficialBaselineUSD, float64(0))
		totalBaseline += sample.OfficialBaselineUSD
	}
	require.InDelta(t, 0.4628, roundMoney(totalBaseline), 0.0001)

	body, err := json.Marshal(map[string]any{
		"type":  "message",
		"model": "claude-opus-4-8",
		"content": []map[string]any{
			{"type": "text", "text": "ok"},
		},
		"usage": map[string]any{
			"input_tokens":                2,
			"output_tokens":               115,
			"cache_creation_input_tokens": 380,
			"cache_read_input_tokens":     23911,
		},
	})
	require.NoError(t, err)
	sample := claudeTokenAuditSampleFromProbe(2, httpProbe{StatusCode: http.StatusOK, Body: body, LatencyMS: 100}, pricing)
	require.Equal(t, CheckStatusPass, sample.Status)
	require.Equal(t, int64(1), sample.BaselineInputTokens)
	require.Equal(t, int64(162), sample.BaselineOutputTokens)
	require.Equal(t, int64(422), sample.BaselineCacheCreation)
	require.Equal(t, int64(23911), sample.BaselineCacheRead)
	require.InDelta(t, 0.92, sample.Multiplier, 0.01)
	require.InDelta(t, 0.92, sample.Ratio, 0.01)

	report := &TokenAuditReport{Samples: []TokenAuditSample{
		{Index: 1, Status: CheckStatusPass, InputTokens: 2, CacheCreationTokens: 100, CachedTokens: 0},
		{Index: 2, Status: CheckStatusPass, InputTokens: 2, CacheCreationTokens: 1, CachedTokens: 99},
		{Index: 3, Status: CheckStatusPass, InputTokens: 2, CacheCreationTokens: 1, CachedTokens: 99},
	}}
	applyClaudeWarmCacheHitRate(report)
	require.Equal(t, 0.97, report.CacheHitRate)
	require.Equal(t, float64(97), report.CacheHitRatePercent)
}

func TestWrapperFingerprintSignals_CoversCommonProxyProviders(t *testing.T) {
	signals := wrapperFingerprintSignals("https://api.proxyai.best", map[string]string{
		"x-cpa-support-plugin": "true",
		"x-relay-provider":     "antigravity",
		"x-provider":           "openai-compatible-kimi",
		"x-upstream-provider":  "xai-grok",
		"x-model-provider":     "gemini-aistudio-codex",
	})
	require.Contains(t, signals, "sub2api")
	require.Contains(t, signals, "cliproxyapi")
	require.Contains(t, signals, "antigravity")
	require.Contains(t, signals, "openai-compatible")
	require.Contains(t, signals, "kimi")
	require.Contains(t, signals, "xai")
	require.Contains(t, signals, "gemini")
	require.Contains(t, signals, "aistudio")
	require.Contains(t, signals, "codex")
}

func TestWrapperFingerprintSignals_CoversSub2APIBlackBoxFingerprints(t *testing.T) {
	report := &PublicReport{
		APIBaseHost:   "relay.example.com",
		Provider:      ProviderOpenAI,
		ModelID:       "gpt-5.5",
		ExpectedModel: "gpt-5.5",
		ResponseModel: "gpt-5.5",
	}
	signals := wrapperFingerprintSignalsForReportWithValues(report, []string{
		`{"error":"API_KEY_REQUIRED","message":"missing Authorization, x-api-key, or x-goog-api-key","routes":["/v1/messages","/backend-api/codex/responses","/antigravity/v1beta/models"]}`,
		`{"data":[{"id":"codex-auto-review"},{"id":"gpt-5.3-codex-spark"},{"id":"models/gemini-3.1-pro-preview-customtools"},{"id":"claude-fable-5"}]}`,
	}, map[string]string{
		"x-client-request-id": "req_123",
	})
	require.Contains(t, signals, "sub2api")
	require.NotContains(t, signals, "sub2api-model-mapping")
	require.NotContains(t, signals, "sub2api-protocol-bridge")
}

func TestWrapperFingerprintSignals_DoesNotFlagSub2APIWeakHeaderAlone(t *testing.T) {
	signals := wrapperFingerprintSignals("relay.example.com", map[string]string{
		"x-client-request-id": "req_123",
	})
	require.NotContains(t, signals, "sub2api")
}

func TestWrapperFingerprintSignals_CoversSub2APIModelMappingLeak(t *testing.T) {
	report := &PublicReport{
		APIBaseHost:   "relay.example.com",
		Provider:      ProviderAnthropic,
		ModelID:       "claude-opus-4-8",
		ExpectedModel: "claude-opus-4-8",
		ResponseModel: "claude-opus-4-8",
	}
	signals := wrapperFingerprintSignalsForReportWithValues(report, []string{
		`{"requested_model":"claude-opus-4-8","upstream_model":"gpt-5.4","model_mapping_chain":["account.model_mapping"]}`,
	}, nil)
	require.Contains(t, signals, "sub2api-model-mapping")

	report.WrapperSignals = signals
	require.True(t, hasWrapperObfuscationFingerprint(report))
	require.Equal(t, 55, wrapperPurityScoreCap(report))
}

func TestWrapperFingerprintSignals_CoversNewAPIHeaderFingerprint(t *testing.T) {
	signals := wrapperFingerprintSignals("relay.example.com", map[string]string{
		"x-new-api-version": "0.9.99",
	})
	require.Contains(t, signals, "new-api")
	require.NotContains(t, signals, "new-api-model-mapping")
}

func TestWrapperFingerprintSignals_CoversCLIProxyAPICodexAndSignatureBridge(t *testing.T) {
	report := &PublicReport{
		APIBaseHost:   "proxy.local",
		Provider:      ProviderOpenAI,
		ModelID:       "gpt-5.4",
		ExpectedModel: "gpt-5.4",
		ResponseModel: "gpt-5.4",
	}
	signals := wrapperFingerprintSignalsForReportWithValues(report, []string{
		`{"message":"CLI Proxy API Server","endpoints":["POST /v1/responses/compact","GET /v1beta/models"]}`,
		`{"oauth-model-alias":[{"name":"glm-5.2","alias":"claude-sonnet-latest","force-mapping":true}]}`,
		`data: {"type":"content_block_delta","delta":{"type":"signature_delta","signature":"claude#abc"}}`,
		`{"native_finish_reason":"MAX_TOKENS","choices":[{"delta":{"reasoning_content":"hidden"}}]}`,
	}, map[string]string{
		"access-control-expose-headers": "X-CPA-VERSION, X-CPA-COMMIT, X-CPA-SUPPORT-PLUGIN",
		"x-codex-turn-metadata":         `{"prompt_cache_key":"cache-1","turn_id":"turn-1"}`,
		"x-codex-window-id":             "cache-1:0",
		"openai-beta":                   "responses_websockets=2026-02-06",
		"originator":                    "codex-tui",
	})
	require.Contains(t, signals, "cliproxyapi")
	require.Contains(t, signals, "cliproxyapi-codex-direct")
	require.Contains(t, signals, "codex")
	require.Contains(t, signals, "cliproxyapi-codex-identity")
	require.Contains(t, signals, "cliproxyapi-model-mapping")
	require.Contains(t, signals, "cliproxyapi-signature-bridge")
}

func TestWrapperPurityScoreCap_DistinguishesTransparentRelayAndObfuscation(t *testing.T) {
	transparent := &PublicReport{
		Provider:       ProviderOpenAI,
		WrapperSignals: []string{"cliproxyapi", "new-api", "sub2api"},
	}
	require.False(t, hasWrapperObfuscationFingerprint(transparent))
	require.Equal(t, CheckStatusWarn, buildWrapperFingerprintCheck(transparent).Status)
	require.Equal(t, 100, wrapperPurityScoreCap(transparent))
	require.Contains(t, summaryForReport(transparent), "透明中转/兼容网关")
	require.Contains(t, summaryForReport(transparent), "未显示模型或协议混淆")

	obfuscated := &PublicReport{
		Provider:       ProviderOpenAI,
		WrapperSignals: []string{"cliproxyapi", "cliproxyapi-model-mapping"},
	}
	require.True(t, hasWrapperObfuscationFingerprint(obfuscated))
	require.Equal(t, CheckStatusFail, buildWrapperFingerprintCheck(obfuscated).Status)
	require.Equal(t, 55, wrapperPurityScoreCap(obfuscated))
	require.Contains(t, summaryForReport(obfuscated), "模型或协议混淆风险")
}

func TestWrapperFingerprintSignals_CoversNewAPIErrorBodyFingerprints(t *testing.T) {
	report := &PublicReport{
		APIBaseHost:   "relay.example.com",
		Provider:      ProviderOpenAI,
		ModelID:       "gpt-5.5",
		ExpectedModel: "gpt-5.5",
		ResponseModel: "gpt-5.5",
	}
	// 错误体独有 error.type 命名空间 + request id 后缀 + 分组（distributor）文案
	signals := wrapperFingerprintSignalsForReportWithValues(report, []string{
		`{"error":{"message":"分组 default 下模型 gpt-5.5 无可用渠道（distributor） (request id: 2026xxxx)","type":"new_api_error","code":"model_not_found"}}`,
	}, nil)
	require.Contains(t, signals, "new-api")
	require.NotContains(t, signals, "new-api-model-mapping")
}

func TestWrapperFingerprintSignals_CoversNewAPIFixedConstantHeaders(t *testing.T) {
	// auth-version / specific_channel_version 固定常量值
	signals := wrapperFingerprintSignals("relay.example.com", map[string]string{
		"auth-version": "864b7076dbcd0a3c01b5520316720ebf",
	})
	require.Contains(t, signals, "new-api")
}

func TestWrapperFingerprintSignals_CoversCLIProxyAPIUnauthFingerprints(t *testing.T) {
	report := &PublicReport{
		APIBaseHost:   "proxy.example.com",
		Provider:      ProviderOpenAI,
		ModelID:       "gpt-5.4",
		ExpectedModel: "gpt-5.4",
		ResponseModel: "gpt-5.4",
	}
	// 无鉴权可观测：OAuth 回调固定 HTML + 管理面文案 + 全局 CORS 新头名
	signals := wrapperFingerprintSignalsForReportWithValues(report, []string{
		`<html><head><title>Authentication successful</title></head><body><h1>Authentication successful!</h1><script>setTimeout(function(){window.close();},5000)</script></body></html>`,
		`{"error":"IP banned due to too many failed attempts. Try again in 5m"}`,
	}, map[string]string{
		"access-control-expose-headers": "X-CPA-VERSION, X-SERVER-VERSION, X-SERVER-BUILD-DATE",
	})
	require.Contains(t, signals, "cliproxyapi")
}

func TestWrapperFingerprintSignals_CoversSub2APIStrongProbes(t *testing.T) {
	report := &PublicReport{
		APIBaseHost:   "relay.example.com",
		Provider:      ProviderAnthropic,
		ModelID:       "claude-opus-4-8",
		ExpectedModel: "claude-opus-4-8",
		ResponseModel: "claude-opus-4-8",
	}
	// 预热拦截 mock id（强独立信号）
	mock := wrapperFingerprintSignalsForReportWithValues(report, []string{
		`event: message_start\ndata: {"type":"message_start","message":{"id":"msg_mock_warmup","usage":{"input_tokens":10}}}`,
	}, nil)
	require.Contains(t, mock, "sub2api")

	// 协议门控 404 文案（强独立信号）
	gating := wrapperFingerprintSignalsForReportWithValues(report, []string{
		`{"error":{"type":"not_found_error","message":"Token counting is not supported for this platform"}}`,
	}, nil)
	require.Contains(t, gating, "sub2api")
}

func TestWrapperFingerprintSignals_DoesNotFlagFableModelOrConfigLeakAlone(t *testing.T) {
	// claude-fable-5 现为真实官方模型，单独出现不得判 sub2api
	fable := wrapperFingerprintSignals("api.anthropic.com", map[string]string{
		"content-type": "application/json",
	})
	require.NotContains(t, fable, "sub2api")
	report := &PublicReport{APIBaseHost: "api.anthropic.com", Provider: ProviderAnthropic}
	fableBody := wrapperFingerprintSignalsForReportWithValues(report, []string{
		`{"data":[{"id":"claude-fable-5","type":"model","display_name":"Claude Fable 5"}]}`,
	}, nil)
	require.NotContains(t, fableBody, "sub2api")

	// turnstile/needs_setup 单条弱信号不得单独触发
	weak := wrapperFingerprintSignals("relay.example.com", map[string]string{
		"x-client-request-id": "req_1",
	})
	require.NotContains(t, weak, "sub2api")
}

func TestWrapperFingerprintSignals_CoversAntigravityAndGeminiRoutes(t *testing.T) {
	// Antigravity 专用路由 + 客户端 UA（不依赖 host 含 antigravity）
	report := &PublicReport{APIBaseHost: "relay.example.com", Provider: ProviderAnthropic}
	ag := wrapperFingerprintSignalsForReportWithValues(report, []string{
		`{"routes":["/antigravity/v1/messages","/antigravity/v1beta/models"]}`,
	}, map[string]string{
		"x-goog-api-client": "antigravity/cli 1.0 darwin/arm64",
	})
	require.Contains(t, ag, "antigravity")

	// Gemini 原生协议标记 + Vertex Google 客户端头
	gem := wrapperFingerprintSignals("relay.example.com", map[string]string{
		"x-goog-api-client": "google-api-nodejs-client/10.3.0",
	})
	require.Contains(t, gem, "vertex")
	gemBody := wrapperFingerprintSignalsForReportWithValues(
		&PublicReport{APIBaseHost: "relay.example.com", Provider: ProviderOpenAI},
		[]string{`{"models":[{"name":"models/gemini-3-flash","supportedGenerationMethods":["generateContent"]}]}`},
		nil,
	)
	require.Contains(t, gemBody, "gemini")
	nativeGeminiBody := wrapperFingerprintSignalsForReportWithValues(
		&PublicReport{APIBaseHost: "relay.example.com", Provider: ProviderGemini},
		[]string{`{"models":[{"name":"models/gemini-3-flash","supportedGenerationMethods":["generateContent"]}]}`},
		nil,
	)
	require.NotContains(t, nativeGeminiBody, "gemini")
}

func TestWrapperFingerprintSignals_DoesNotFlagOfficialCodexModelNameAlone(t *testing.T) {
	report := &PublicReport{
		APIBaseHost:   "api.openai.com",
		Provider:      ProviderOpenAI,
		ModelID:       "gpt-5-codex",
		ExpectedModel: "gpt-5-codex",
		ResponseModel: "gpt-5-codex",
	}
	signals := wrapperFingerprintSignalsForReport(report)
	require.NotContains(t, signals, "codex")
}

func TestWrapperFingerprintSignals_CoversMainstreamRelayBrands(t *testing.T) {
	signals := wrapperFingerprintSignals("api.pptoken.org", map[string]string{
		"x-litellm-version": "1.70.0",
		"server":            "LiteLLM",
		"x-provider":        "api2d-aiproxy-openmodel-pateway-suixiang-ohmygpt",
	})
	require.Contains(t, signals, "litellm")
	require.Contains(t, signals, "api2d")
	require.Contains(t, signals, "aiproxy")
	require.Contains(t, signals, "openmodel")
	require.Contains(t, signals, "pptoken")
	require.Contains(t, signals, "pateway")
	require.Contains(t, signals, "suixiang")
	require.Contains(t, signals, "ohmygpt")
}

func TestWrapperFingerprintSignals_CoversChineseOpenModelChannels(t *testing.T) {
	report := &PublicReport{
		APIBaseHost:   "ark.cn-beijing.volces.com",
		ModelID:       "qwen3.7-max",
		ExpectedModel: "glm-5.2",
		ResponseModel: "doubao-seed-2-1-pro-260628",
	}
	signals := wrapperFingerprintSignalsForReport(report, map[string]string{
		"x-provider":          "MiniMax-M2.7-highspeed",
		"x-upstream-provider": "hy3-preview",
		"x-model-provider":    "kimi-k2.7-code-highspeed mimo-v2.5-pro",
	})
	require.Contains(t, signals, "qwen")
	require.Contains(t, signals, "glm")
	require.Contains(t, signals, "doubao")
	require.Contains(t, signals, "minimax")
	require.Contains(t, signals, "hunyuan")
	require.Contains(t, signals, "kimi")
	require.Contains(t, signals, "mimo")
}

func TestWrapperFingerprintSignals_CoversDeepSeekChannel(t *testing.T) {
	signals := wrapperFingerprintSignals("api.deepseek.com", map[string]string{
		"x-upstream-provider": "deepseek-v4-pro",
	})
	require.Contains(t, signals, "deepseek")
}

func TestWrapperFingerprintSignals_CoversCloudProviderChannels(t *testing.T) {
	signals := wrapperFingerprintSignals("us-central1-aiplatform.googleapis.com", map[string]string{
		"x-goog-request-id": "goog-req",
		"x-amzn-requestid":  "aws-req",
	})
	require.Contains(t, signals, "vertex")
	require.Contains(t, signals, "bedrock")
}

func TestModelIdentityDetectsVersionAndTierDowngrade(t *testing.T) {
	report := &PublicReport{
		Provider:      ProviderOpenAI,
		ModelID:       "gpt5.5",
		ExpectedModel: "gpt5.5",
		ResponseModel: "gpt-5.4-mini",
	}
	check := buildModelIdentityCheck(report)
	require.Equal(t, CheckStatusFail, check.Status)
	require.NotNil(t, report.ModelIdentity)
	require.Equal(t, modelIdentityReasonVersionDowngrade, report.ModelIdentity.Reason)
	require.Equal(t, "openai", report.ModelIdentity.RequestedVendor)
	require.Equal(t, "openai", report.ModelIdentity.ResponseVendor)
	require.Contains(t, check.Message, "降级")

	claudeReport := &PublicReport{
		Provider:      ProviderAnthropic,
		ModelID:       "claude-opus-4-8",
		ExpectedModel: "claude-opus-4-8",
		ResponseModel: "claude-opus-4-6",
	}
	claudeCheck := buildModelIdentityCheck(claudeReport)
	require.Equal(t, CheckStatusFail, claudeCheck.Status)
	require.Equal(t, modelIdentityReasonVersionDowngrade, claudeReport.ModelIdentity.Reason)
}

func TestModelIdentityDetectsCrossVendorAndWrapperAlias(t *testing.T) {
	report := &PublicReport{
		Provider:      ProviderAnthropic,
		ModelID:       "claude-sonnet-latest",
		ExpectedModel: "claude-sonnet-latest",
		ResponseModel: "glm-5.2",
	}
	check := buildModelIdentityCheck(report)
	require.Equal(t, CheckStatusFail, check.Status)
	require.Equal(t, modelIdentityReasonCrossVendorAlias, report.ModelIdentity.Reason)
	require.Equal(t, "anthropic", report.ModelIdentity.RequestedVendor)
	require.Equal(t, "glm", report.ModelIdentity.ResponseVendor)

	forceMappedReport := &PublicReport{
		Provider:       ProviderAnthropic,
		ModelID:        "claude-opus-4-8",
		ExpectedModel:  "claude-opus-4-8",
		ResponseModel:  "claude-opus-4-8",
		WrapperSignals: []string{"antigravity"},
	}
	forceMappedCheck := buildModelIdentityCheck(forceMappedReport)
	require.Equal(t, CheckStatusFail, forceMappedCheck.Status)
	require.Equal(t, modelIdentityReasonWrapperVendorSignalMismatch, forceMappedReport.ModelIdentity.Reason)
	require.Equal(t, "google", forceMappedReport.ModelIdentity.Evidence["suspected_upstream_vendor"])
}

func TestModelIdentityDetectsProtocolVendorMismatchForceMappingAlias(t *testing.T) {
	report := &PublicReport{
		Provider:      ProviderOpenAI,
		ModelID:       "claude-opus-4.66",
		ExpectedModel: "claude-opus-4.66",
		ResponseModel: "claude-opus-4.66",
	}
	check := buildModelIdentityCheck(report)
	require.Equal(t, CheckStatusFail, check.Status)
	require.Equal(t, modelIdentityReasonProtocolVendorMismatch, report.ModelIdentity.Reason)
	require.Equal(t, "anthropic", report.ModelIdentity.RequestedVendor)
	require.Equal(t, "openai", report.ModelIdentity.Evidence["protocol_expected_vendor"])

	claudeReport := &PublicReport{
		Provider:      ProviderAnthropic,
		ModelID:       "gpt-5.5",
		ExpectedModel: "gpt-5.5",
		ResponseModel: "gpt-5.5",
	}
	claudeCheck := buildModelIdentityCheck(claudeReport)
	require.Equal(t, CheckStatusFail, claudeCheck.Status)
	require.Equal(t, modelIdentityReasonProtocolVendorMismatch, claudeReport.ModelIdentity.Reason)
	require.Equal(t, "openai", claudeReport.ModelIdentity.RequestedVendor)
	require.Equal(t, "anthropic", claudeReport.ModelIdentity.Evidence["protocol_expected_vendor"])
}

func TestModelIdentityDetectsUnexpectedReasoningTokens(t *testing.T) {
	report := &PublicReport{
		Provider:      ProviderOpenAI,
		ModelID:       "gpt-5.4",
		ExpectedModel: "gpt-5.4",
		ResponseModel: "gpt-5.4",
		Metrics: PublicCheckMetrics{
			Usage: &TokenUsage{InputTokens: 10, OutputTokens: 3, TotalTokens: 13, ReasoningTokens: 2},
		},
	}
	check := buildModelIdentityCheck(report)
	require.Equal(t, CheckStatusFail, check.Status)
	require.Equal(t, modelIdentityReasonReasoningTokensMismatch, report.ModelIdentity.Reason)
	require.Equal(t, int64(2), report.ModelIdentity.Evidence["reasoning_tokens"])

	codexReport := &PublicReport{
		Provider:      ProviderOpenAI,
		ModelID:       "gpt-5-codex",
		ExpectedModel: "gpt-5-codex",
		ResponseModel: "gpt-5-codex",
		Metrics: PublicCheckMetrics{
			Usage: &TokenUsage{InputTokens: 10, OutputTokens: 3, TotalTokens: 13, ReasoningTokens: 2},
		},
	}
	codexCheck := buildModelIdentityCheck(codexReport)
	require.Equal(t, CheckStatusPass, codexCheck.Status)
}

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

func TestServiceRunPublicCheck_OpenAIResponsesStoreIncludeAccepted(t *testing.T) {
	server := newOpenAIStoreIncludeServer(t, http.StatusOK, "", nil)
	defer server.Close()

	service := NewService(nil)
	service.httpClient = server.Client()
	service.allowPrivateHosts = true
	service.limiter = nil

	report, err := service.RunDeveloperAPICheck(context.Background(), PublicCheckInput{
		Provider:       ProviderOpenAI,
		APIBaseURL:     server.URL,
		APIKey:         "sk-store-include",
		ModelID:        "gpt-5.4",
		ClientIP:       "203.0.113.20",
		SkipTokenAudit: true,
	})
	require.NoError(t, err)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "responses_store_include").Status)
	require.Equal(t, CheckStatusPass, findValidation(t, report, "signature").Status)
	require.Equal(t, VerdictOfficialOpenAI, report.Verdict)
	require.Equal(t, 100, report.OfficialScore)
}

func TestServiceRunPublicCheck_OpenAIResponsesStoreIncludeRejected(t *testing.T) {
	server := newOpenAIStoreIncludeServer(t, http.StatusBadRequest, "unsupported include: reasoning.encrypted_content", nil)
	defer server.Close()

	service := NewService(nil)
	service.httpClient = server.Client()
	service.allowPrivateHosts = true
	service.limiter = nil

	report, err := service.RunDeveloperAPICheck(context.Background(), PublicCheckInput{
		Provider:       ProviderOpenAI,
		APIBaseURL:     server.URL,
		APIKey:         "sk-store-include",
		ModelID:        "gpt-5.4",
		ClientIP:       "203.0.113.20",
		SkipTokenAudit: true,
	})
	require.NoError(t, err)
	require.Equal(t, CheckStatusFail, findCheck(t, report, "responses_store_include").Status)
	require.Equal(t, CheckStatusFail, findValidation(t, report, "signature").Status)
	require.Equal(t, VerdictOpenAICompatible, report.Verdict)
	require.Equal(t, 70, report.OfficialScore)
	require.Equal(t, 86, report.CompatibilityScore)
}

func TestServiceRunPublicCheck_OpenAIResponsesStoreIncludeBalanceWarn(t *testing.T) {
	server := newOpenAIStoreIncludeServer(t, http.StatusForbidden, "Insufficient account balance", nil)
	defer server.Close()

	service := NewService(nil)
	service.httpClient = server.Client()
	service.allowPrivateHosts = true
	service.limiter = nil

	report, err := service.RunDeveloperAPICheck(context.Background(), PublicCheckInput{
		Provider:       ProviderOpenAI,
		APIBaseURL:     server.URL,
		APIKey:         "sk-store-include",
		ModelID:        "gpt-5.4",
		ClientIP:       "203.0.113.20",
		SkipTokenAudit: true,
	})
	require.NoError(t, err)
	require.Equal(t, CheckStatusWarn, findCheck(t, report, "responses_store_include").Status)
	require.Equal(t, CheckStatusWarn, findValidation(t, report, "signature").Status)
	require.Equal(t, VerdictOfficialOpenAI, report.Verdict)
	require.Contains(t, findCheck(t, report, "responses_store_include").Message, "不能据此判断非官方")
}

func TestPublicReportCCTestCompatFields(t *testing.T) {
	report := &PublicReport{
		Provider:         ProviderOpenAI,
		ReportID:         "report-compat",
		APIBaseHost:      "api.proxyai.best",
		ModelID:          "gpt-5.4",
		ExpectedModel:    "gpt-5.4",
		ResponseModel:    "gpt-5.4",
		Status:           RunStatusRunning,
		Step:             6,
		StepName:         "token_audit",
		Progress:         0.86,
		Verdict:          VerdictOpenAICompatible,
		StreamChannel:    "openai_compatible",
		NonStreamChannel: "openai_compatible",
		HasVertex:        true,
		IsKiro:           true,
		WrapperSignals:   []string{"vertex", "kiro"},
		Checks: []CheckResult{
			passCheck("base_url", "API Base 域名", 20, "ok", nil),
			passCheck("responses_schema", "Responses 非流式结构", 20, "ok", nil),
			passCheck("tool_call", "强制工具调用", 20, "ok", nil),
			passCheck("usage", "Usage 计量", 10, "ok", nil),
			passCheck("streaming", "Responses 流式事件", 15, "ok", nil),
			{ID: "responses_store_include", Name: "Responses store/include", Status: CheckStatusPass, Score: 0, MaxScore: 0, Message: "ok"},
			passCheck("multimodal", "多模态输入", 10, "ok", nil),
			passCheck("token_audit", "Token 用量审计", 15, "ok", nil),
		},
		Metrics: PublicCheckMetrics{
			LatencyMS:            321,
			StreamFirstTokenMS:   45,
			StreamTotalLatencyMS: 180,
			ModelsLatencyMS:      12,
			ResponsesLatencyMS:   98,
			MultimodalLatencyMS:  110,
			TokensPerSecond:      18.75,
			Usage:                &TokenUsage{InputTokens: 1200, OutputTokens: 88, TotalTokens: 1288, CachedTokens: 1024},
		},
		CheckedAt: time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC),
	}
	finalizeReport(report)

	raw, err := json.Marshal(report)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	require.Equal(t, "token_audit", payload["stepName"])
	require.Equal(t, float64(100), payload["total"])
	require.Equal(t, "official", payload["verdictKey"])
	require.Equal(t, "gpt-5.4", payload["expectedModel"])
	require.Equal(t, "gpt-5.4", payload["responseModel"])
	require.Equal(t, "openai_compatible", payload["streamChannel"])
	require.Equal(t, "openai_compatible", payload["nonStreamChannel"])
	require.Equal(t, true, payload["hasVertex"])
	require.Equal(t, true, payload["isKiro"])
	require.Equal(t, []any{"vertex", "kiro"}, payload["wrapperSignals"])

	metrics, ok := payload["metrics"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(321), metrics["latencyMs"])
	require.Equal(t, float64(45), metrics["ttfbMs"])
	require.Equal(t, 18.75, metrics["tokensPerSec"])
	require.Equal(t, float64(1200), metrics["inputTokens"])
	require.Equal(t, float64(88), metrics["outputTokens"])

	var emitted PublicCheckEvent
	emitPublicCheckEvent(func(event PublicCheckEvent) {
		emitted = event
	}, PublicCheckEvent{
		Type:     PublicCheckEventProgress,
		ReportID: report.ReportID,
		StepName: report.StepName,
		Metrics:  &report.Metrics,
		Report:   report,
	})
	eventRaw, err := json.Marshal(emitted)
	require.NoError(t, err)
	var eventPayload map[string]any
	require.NoError(t, json.Unmarshal(eventRaw, &eventPayload))
	require.Equal(t, "token_audit", eventPayload["stepName"])
	eventMetrics, ok := eventPayload["metrics"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(321), eventMetrics["latencyMs"])
	require.Equal(t, float64(45), eventMetrics["ttfbMs"])
	require.Equal(t, 18.75, eventMetrics["tokensPerSec"])
}

func TestServiceRunPublicCheckStream_EmitsProgressEvents(t *testing.T) {
	auditResponseIndex := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
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

	var events []PublicCheckEvent
	report, err := service.RunPublicCheckStream(context.Background(), PublicCheckInput{
		Provider:   ProviderOpenAI,
		APIBaseURL: server.URL,
		APIKey:     "sk-test",
		ModelID:    "gpt-5.4",
		ClientIP:   "203.0.113.10",
	}, func(event PublicCheckEvent) {
		events = append(events, event)
	})
	require.NoError(t, err)
	require.Equal(t, report.ReportID, events[0].ReportID)
	require.Equal(t, PublicCheckEventStarted, events[0].Type)
	require.Contains(t, eventTypes(events), PublicCheckEventCheck)
	require.Contains(t, eventTypes(events), PublicCheckEventValidation)
	require.Contains(t, eventTypes(events), PublicCheckEventTokenAuditSample)
	require.Equal(t, PublicCheckEventReport, events[len(events)-1].Type)
	require.NotNil(t, events[len(events)-1].Report)
	require.Len(t, events[len(events)-1].Report.Validations, 8)
	require.Equal(t, []string{
		"base_url",
		"models_schema",
		"responses_schema",
		"tool_call",
		"usage",
		"streaming",
		"responses_store_include",
		"multimodal",
		"token_audit",
		"model_identity",
		"wrapper_fingerprint",
	}, checkEventIDs(events))
	require.Equal(t, []string{
		"llm_fingerprint",
		"schema_integrity",
		"behavior",
		"signature",
		"multimodal",
		"token_audit",
		"model_identity",
		"wrapper_fingerprint",
	}, validationEventIDs(events))
	require.Contains(t, eventTypes(events), PublicCheckEventProgress)
	require.Contains(t, progressStepNames(events), "signature")
	for _, event := range events {
		if event.Type != PublicCheckEventValidation {
			continue
		}
		require.NotNil(t, event.Validation)
		require.Contains(t, []string{"programmatic_probe", "openai_base_url_and_models_probe", "channel_signal_detectors"}, event.Validation.Details["detector"])
		require.NotEmpty(t, event.Validation.RelatedCheckIDs)
	}
}

func TestServiceRunPublicCheckStream_ClaudePureBehindCustomBaseURL(t *testing.T) {
	server := newClaudeCompatibleServer(t, "sk-claude-test")
	defer server.Close()

	service := NewService(nil)
	service.allowPrivateHosts = true
	service.limiter = nil

	var events []PublicCheckEvent
	report, err := service.RunPublicCheckStream(context.Background(), PublicCheckInput{
		Provider:   ProviderAnthropic,
		APIBaseURL: server.URL,
		APIKey:     "sk-claude-test",
		ModelID:    "claude-sonnet-4-6",
		ClientIP:   "203.0.113.10",
	}, func(event PublicCheckEvent) {
		events = append(events, event)
	})
	require.NoError(t, err)
	require.Equal(t, AccessModeWeb, report.AccessMode)
	require.Equal(t, BillingModeCaptchaRateLimit, report.BillingMode)
	require.Equal(t, ProviderAnthropic, report.Provider)
	require.Equal(t, VerdictOfficialClaude, report.Verdict)
	require.Equal(t, 100, report.CompatibilityScore)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "claude_messages_schema").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "claude_tool_use").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "claude_streaming").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "claude_thinking_signature").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "claude_multimodal").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "token_audit").Status)
	require.Equal(t, CheckStatusPass, findValidation(t, report, "behavior").Status)
	require.Equal(t, CheckStatusPass, findValidation(t, report, "signature").Status)
	require.NotNil(t, report.TokenAudit)
	require.Equal(t, tokenAuditSamples, report.TokenAudit.SampleCount)
	require.Len(t, report.TokenAudit.Rows, tokenAuditSamples)
	require.Equal(t, report.TokenAudit.ActualCostUSD, report.TokenAudit.TotalCostUSD)
	require.Greater(t, report.TokenAudit.Samples[0].CacheCreationInputTokens, int64(0))
	require.Greater(t, report.TokenAudit.Samples[1].CacheReadInputTokens, int64(0))
	require.Contains(t, eventTypes(events), PublicCheckEventProgress)
	require.Contains(t, eventTypes(events), PublicCheckEventTokenAuditSample)
	require.Contains(t, progressStepNames(events), "signature")
	require.Equal(t, PublicCheckEventReport, events[len(events)-1].Type)
}

func TestServiceRunAccountCheckStream_InfersClaudeProvider(t *testing.T) {
	server := newClaudeCompatibleServer(t, "sk-claude-account")
	defer server.Close()

	service := NewServiceWithAccountResolver(nil, accountResolverStub{
		account: &coreservice.Account{
			ID:       84,
			Platform: coreservice.PlatformAnthropic,
			Type:     coreservice.AccountTypeAPIKey,
			Credentials: map[string]any{
				"api_key":  "sk-claude-account",
				"base_url": server.URL,
			},
		},
	})
	service.limiter = nil

	report, err := service.RunAccountCheckStream(context.Background(), AccountCheckInput{
		AccountID: 84,
		ModelID:   "claude-sonnet-4-6",
	}, nil)
	require.NoError(t, err)
	require.Equal(t, AccessModeAccount, report.AccessMode)
	require.Equal(t, BillingModeAccountInternal, report.BillingMode)
	require.Equal(t, ProviderAnthropic, report.Provider)
	require.Equal(t, VerdictOfficialClaude, report.Verdict)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "claude_tool_use").Status)
}

func TestServiceRunAccountCheckStream_InfersGeminiProvider(t *testing.T) {
	server := newGeminiCompatibleServer(t, "AIza-account", "gemini-3-pro-preview")
	defer server.Close()

	service := NewServiceWithAccountResolver(nil, accountResolverStub{
		account: &coreservice.Account{
			ID:       85,
			Platform: coreservice.PlatformGemini,
			Type:     coreservice.AccountTypeAPIKey,
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
		AccountID: 85,
		ModelID:   "gemini-3-pro-preview",
	}, nil)
	require.NoError(t, err)
	require.Equal(t, AccessModeAccount, report.AccessMode)
	require.Equal(t, BillingModeAccountInternal, report.BillingMode)
	require.Equal(t, ProviderGemini, report.Provider)
	require.Equal(t, VerdictOfficialGemini, report.Verdict)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "tool_call").Status)
	require.Equal(t, CheckStatusPass, findCheck(t, report, "multimodal").Status)
	require.Equal(t, CheckStatusWarn, findCheck(t, report, "token_audit").Status)
}

func TestServiceRunPublicCheck_ClaudeWrapperSignatureAndUsageAnomaliesAreNotOfficial(t *testing.T) {
	server := newClaudeWrapperServer(t, "sk-wrapper-test", map[string]string{
		"X-Kiro-Upstream":  "kiro",
		"X-Relay-Provider": "antigravity",
	})
	defer server.Close()

	service := NewService(nil)
	service.allowPrivateHosts = true
	service.limiter = nil

	report, err := service.RunPublicCheck(context.Background(), PublicCheckInput{
		Provider:   ProviderAnthropic,
		APIBaseURL: server.URL,
		APIKey:     "sk-wrapper-test",
		ModelID:    "claude-opus-4-8",
		ClientIP:   "203.0.113.10",
	})
	require.NoError(t, err)
	require.True(t, report.IsKiro)
	require.Contains(t, report.WrapperSignals, "kiro")
	require.Contains(t, report.WrapperSignals, "antigravity")
	require.Contains(t, report.WrapperSignalsCompat, "kiro")
	require.Equal(t, CheckStatusFail, findCheck(t, report, "wrapper_fingerprint").Status)
	require.Equal(t, VerdictPartialCompatible, report.Verdict)
	require.NotEqual(t, VerdictOfficialClaude, report.Verdict)
	require.LessOrEqual(t, report.Score, 45)
	require.Equal(t, CheckStatusFail, findCheck(t, report, "claude_thinking_signature").Status)
	require.Equal(t, CheckStatusFail, findCheck(t, report, "claude_thinking_budget").Status)
	require.Equal(t, CheckStatusFail, findCheck(t, report, "claude_cache_control_overflow").Status)
	require.Equal(t, CheckStatusFail, findValidation(t, report, "signature").Status)
	require.NotNil(t, report.TokenAudit)
	require.Equal(t, CheckStatusWarn, report.TokenAudit.Status)
	require.Contains(t, report.TokenAudit.Anomalies, "claude_cache_accounting_missing")
	require.Contains(t, report.TokenAudit.Anomalies, "cost_multiplier_anomaly")
	require.Contains(t, report.Summary, "包装/中转")
	require.Contains(t, report.Summary, "antigravity")
	require.Contains(t, report.Summary, "兼容受限")
}

func TestServiceRunAccountCheckStream_LoadsAccountCredentialAndEmitsProgress(t *testing.T) {
	auditResponseIndex := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer sk-account", r.Header.Get("Authorization"))
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

	service := NewServiceWithAccountResolver(nil, accountResolverStub{
		account: &coreservice.Account{
			ID:       42,
			Platform: coreservice.PlatformOpenAI,
			Type:     coreservice.AccountTypeAPIKey,
			Credentials: map[string]any{
				"api_key":  "sk-account",
				"base_url": server.URL,
			},
		},
	})
	service.httpClient = server.Client()
	service.limiter = nil

	var events []PublicCheckEvent
	report, err := service.RunAccountCheckStream(context.Background(), AccountCheckInput{
		AccountID: 42,
		Provider:  ProviderOpenAI,
		ModelID:   "gpt-5.4",
	}, func(event PublicCheckEvent) {
		events = append(events, event)
	})
	require.NoError(t, err)
	require.Equal(t, AccessModeAccount, report.AccessMode)
	require.Equal(t, AccessModeAccount, report.AccessModeCompat)
	require.Equal(t, BillingModeAccountInternal, report.BillingMode)
	require.Equal(t, BillingModeAccountInternal, report.BillingModeCompat)
	require.Equal(t, VerdictOfficialOpenAI, report.Verdict)
	require.Contains(t, eventTypes(events), PublicCheckEventValidation)
	require.Contains(t, eventTypes(events), PublicCheckEventTokenAuditSample)
	require.Equal(t, PublicCheckEventReport, events[len(events)-1].Type)
	require.Equal(t, CheckStatusPass, findValidation(t, report, "behavior").Status)
	require.Equal(t, tokenAuditSamples, report.TokenAudit.SampleCount)
}

func TestServiceRunPublicCheck_FailsUnexpectedToolCall(t *testing.T) {
	auditResponseIndex := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
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
			writeJSON(t, w, map[string]any{
				"id":     "resp_1",
				"object": "response",
				"output": []map[string]any{
					{"type": "function_call", "name": "wrong_tool", "arguments": `{"ok":true}`},
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
		APIKey:     "sk-test",
		ModelID:    "gpt-5.4",
		ClientIP:   "203.0.113.10",
	})
	require.NoError(t, err)
	require.Equal(t, CheckStatusFail, findCheck(t, report, "tool_call").Status)
	require.Equal(t, CheckStatusFail, findValidation(t, report, "behavior").Status)
}

func TestServiceRunPublicCheck_BlocksPrivateBaseURL(t *testing.T) {
	service := NewService(nil)
	service.limiter = nil

	_, err := service.RunPublicCheck(context.Background(), PublicCheckInput{
		Provider:   ProviderOpenAI,
		APIBaseURL: "http://127.0.0.1:8080",
		APIKey:     "sk-test",
		ModelID:    "gpt-5.4",
		ClientIP:   "203.0.113.10",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "PURITY_BASE_URL_INVALID")
}

func TestServiceRunPublicCheck_RedactsAPIKeyFromReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(t, w, map[string]any{
			"error": map[string]any{"message": "bad key sk-secret-value"},
		})
	}))
	defer server.Close()

	service := NewService(nil)
	service.httpClient = server.Client()
	service.allowPrivateHosts = true
	service.limiter = nil

	report, err := service.RunPublicCheck(context.Background(), PublicCheckInput{
		Provider:   ProviderOpenAI,
		APIBaseURL: server.URL,
		APIKey:     "sk-secret-value",
		ModelID:    "gpt-5.4",
		ClientIP:   "203.0.113.10",
	})
	require.NoError(t, err)
	require.Equal(t, RunStatusError, report.Status)
	require.Contains(t, report.Error, "[redacted]")
	raw, err := json.Marshal(report)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "sk-secret-value")
	require.Contains(t, string(raw), "[redacted]")
}

func TestServiceRunPublicCheck_InsufficientBalanceIsFatalAccountState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		writeJSON(t, w, map[string]any{
			"error": map[string]any{"message": "Insufficient account balance"},
		})
	}))
	defer server.Close()

	service := NewService(nil)
	service.httpClient = server.Client()
	service.allowPrivateHosts = true
	service.limiter = nil

	report, err := service.RunPublicCheck(context.Background(), PublicCheckInput{
		Provider:   ProviderOpenAI,
		APIBaseURL: server.URL,
		APIKey:     "sk-test",
		ModelID:    "gpt-5.4",
		ClientIP:   "203.0.113.10",
	})
	require.NoError(t, err)
	require.Equal(t, RunStatusError, report.Status)
	require.Equal(t, VerdictInvalidOrUnavailable, report.Verdict)
	require.Equal(t, 0, report.Score)
	require.Equal(t, 0, report.CompatibilityScore)
	require.Equal(t, 0, report.OfficialScore)
	require.Equal(t, errorClassAccountBalanceInsufficient, report.Metrics.ErrorClass)
	require.Contains(t, report.Error, "账号余额不足")
	require.Contains(t, report.Summary, "账号余额不足")
	require.Equal(t, CheckStatusFail, findCheck(t, report, "models_schema").Status)
	require.Contains(t, findCheck(t, report, "models_schema").Message, "账号余额不足")
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(payload))
}

func payloadHasInputImage(payload map[string]any) bool {
	body, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	return strings.Contains(string(body), `"type":"input_image"`)
}

func openAIStoreIncludeProbeRequest(payload map[string]any) bool {
	if store, ok := payload["store"].(bool); !ok || store {
		return false
	}
	if stream, ok := payload["stream"].(bool); !ok || stream {
		return false
	}
	reasoning, _ := payload["reasoning"].(map[string]any)
	if reasoning == nil {
		return false
	}
	if effort, _ := reasoning["effort"].(string); effort != "minimal" {
		return false
	}
	includes, ok := payload["include"].([]any)
	if !ok {
		return false
	}
	for _, item := range includes {
		if value, _ := item.(string); value == "reasoning.encrypted_content" {
			return true
		}
	}
	return false
}

func writeOpenAITextResponse(t *testing.T, w http.ResponseWriter, id string, model string, text string) {
	t.Helper()
	writeJSON(t, w, map[string]any{
		"id":     id,
		"object": "response",
		"model":  model,
		"output": []map[string]any{
			{"type": "message", "role": "assistant", "content": []map[string]any{{"type": "output_text", "text": text}}},
		},
		"usage": map[string]any{
			"input_tokens":  8,
			"output_tokens": 2,
			"total_tokens":  10,
		},
	})
}

func findCheck(t *testing.T, report *PublicReport, id string) CheckResult {
	t.Helper()
	for _, check := range report.Checks {
		if check.ID == id {
			return check
		}
	}
	t.Fatalf("check %s not found", id)
	return CheckResult{}
}

func findValidation(t *testing.T, report *PublicReport, id string) ValidationResult {
	t.Helper()
	for _, validation := range report.Validations {
		if validation.ID == id {
			return validation
		}
	}
	t.Fatalf("validation %s not found", id)
	return ValidationResult{}
}

func newOpenAIStoreIncludeServer(t *testing.T, storeIncludeStatus int, storeIncludeMessage string, responseHeaders map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer sk-store-include", r.Header.Get("Authorization"))
		for key, value := range responseHeaders {
			w.Header().Set(key, value)
		}
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
			if openAIStoreIncludeProbeRequest(body) {
				if storeIncludeStatus >= 200 && storeIncludeStatus < 300 {
					writeOpenAITextResponse(t, w, "resp_store_include", "gpt-5.4", "ok")
					return
				}
				w.WriteHeader(storeIncludeStatus)
				writeJSON(t, w, map[string]any{"error": map[string]any{"message": storeIncludeMessage}})
				return
			}
			if payloadHasInputImage(body) {
				writeOpenAITextResponse(t, w, "resp_multimodal", "gpt-5.4", "ok")
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
}

func eventTypes(events []PublicCheckEvent) []string {
	out := make([]string, 0, len(events))
	for _, event := range events {
		out = append(out, event.Type)
	}
	return out
}

func validationEventIDs(events []PublicCheckEvent) []string {
	out := make([]string, 0, len(events))
	for _, event := range events {
		if event.Type == PublicCheckEventValidation && event.Validation != nil {
			out = append(out, event.Validation.ID)
		}
	}
	return out
}

func checkEventIDs(events []PublicCheckEvent) []string {
	out := make([]string, 0, len(events))
	for _, event := range events {
		if event.Type == PublicCheckEventCheck && event.Check != nil {
			out = append(out, event.Check.ID)
		}
	}
	return out
}

func progressStepNames(events []PublicCheckEvent) []string {
	out := make([]string, 0, len(events))
	for _, event := range events {
		if event.Type == PublicCheckEventProgress {
			out = append(out, event.StepName)
		}
	}
	return out
}

func newClaudeCompatibleServer(t *testing.T, expectedKey string) *httptest.Server {
	t.Helper()
	auditRound := 0
	sessionID := ""
	deviceID := ""
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/messages", r.URL.Path)
		require.Equal(t, expectedKey, r.Header.Get("x-api-key"))
		require.Equal(t, anthropicVersion, r.Header.Get("anthropic-version"))
		require.Equal(t, "cli", r.Header.Get("x-app"))
		require.Equal(t, claudeCodeProbeUserAgent, r.Header.Get("User-Agent"))
		require.Equal(t, claudeAPIKeyBetaHeader, r.Header.Get("anthropic-beta"))
		require.Equal(t, "js", r.Header.Get("X-Stainless-Lang"))
		require.Equal(t, "0.94.0", r.Header.Get("X-Stainless-Package-Version"))
		require.Equal(t, "Linux", r.Header.Get("X-Stainless-OS"))
		require.Equal(t, "arm64", r.Header.Get("X-Stainless-Arch"))
		require.Equal(t, "node", r.Header.Get("X-Stainless-Runtime"))
		require.Equal(t, "v24.3.0", r.Header.Get("X-Stainless-Runtime-Version"))
		require.Equal(t, "0", r.Header.Get("X-Stainless-Retry-Count"))
		require.Equal(t, "600", r.Header.Get("X-Stainless-Timeout"))
		require.Equal(t, "true", r.Header.Get("Anthropic-Dangerous-Direct-Browser-Access"))
		require.NotEmpty(t, r.Header.Get("x-client-request-id"))
		requestSessionID := r.Header.Get("X-Claude-Code-Session-Id")
		require.NotEmpty(t, requestSessionID)
		if sessionID == "" {
			sessionID = requestSessionID
		}
		require.Equal(t, sessionID, requestSessionID)

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		metadata, ok := body["metadata"].(map[string]any)
		require.True(t, ok)
		userID, ok := metadata["user_id"].(string)
		require.True(t, ok)
		var parsedUserID map[string]string
		require.NoError(t, json.Unmarshal([]byte(userID), &parsedUserID))
		require.Len(t, parsedUserID["device_id"], 64)
		if deviceID == "" {
			deviceID = parsedUserID["device_id"]
		}
		require.Equal(t, deviceID, parsedUserID["device_id"])
		require.Equal(t, "", parsedUserID["account_uuid"])
		require.Equal(t, sessionID, parsedUserID["session_id"])

		model, _ := body["model"].(string)
		if model == "" {
			model = defaultClaudeModel
		}
		if stream, _ := body["stream"].(bool); stream {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprintf(w, "data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_stream\",\"type\":\"message\",\"role\":\"assistant\",\"model\":%q,\"content\":[]}}\n\n", model)
			_, _ = fmt.Fprintln(w, `data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"ok"}}`)
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, `data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":1}}`)
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, `data: {"type":"message_stop"}`)
			return
		}

		if tools, ok := body["tools"].([]any); ok && len(tools) > 0 {
			writeClaudeMessage(t, w, model, []map[string]any{
				{"type": "tool_use", "id": "toolu_1", "name": "probe_ping", "input": map[string]any{"ok": true}},
			}, map[string]any{
				"input_tokens":  18,
				"output_tokens": 4,
			})
			return
		}
		if claudeBodyHasThinkingBudgetViolation(body) {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(t, w, map[string]any{
				"error": map[string]any{"message": "thinking.budget_tokens must be less than max_tokens"},
			})
			return
		}
		if claudeBodyContainsThinking(body) {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(t, w, map[string]any{
				"error": map[string]any{"message": "Invalid `signature` in `thinking` block"},
			})
			return
		}
		if claudeBodyHasCacheControlOverflow(body) {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(t, w, map[string]any{
				"error": map[string]any{"message": "Too many cache_control blocks in system prompt"},
			})
			return
		}
		if claudeBodyContainsImage(body) {
			writeClaudeMessage(t, w, model, []map[string]any{
				{"type": "text", "text": "ok"},
			}, map[string]any{
				"input_tokens":  20,
				"output_tokens": 2,
			})
			return
		}

		auditRound++
		baselineUsage := claudeTokenAuditBaselineUsage(auditRound)
		require.NotNil(t, baselineUsage)
		writeClaudeMessage(t, w, model, []map[string]any{
			{"type": "text", "text": "ok"},
		}, map[string]any{
			"input_tokens":                baselineUsage.InputTokens,
			"output_tokens":               baselineUsage.OutputTokens,
			"cache_creation_input_tokens": baselineUsage.CacheCreationTokens,
			"cache_read_input_tokens":     baselineUsage.CachedTokens,
		})
	}))
}

func newClaudeWrapperServer(t *testing.T, expectedKey string, responseHeaders map[string]string) *httptest.Server {
	t.Helper()
	auditRound := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/messages", r.URL.Path)
		require.Equal(t, expectedKey, r.Header.Get("x-api-key"))
		for key, value := range responseHeaders {
			w.Header().Set(key, value)
		}

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		model, _ := body["model"].(string)
		if model == "" {
			model = defaultClaudeModel
		}
		if stream, _ := body["stream"].(bool); stream {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprintf(w, "data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_stream\",\"type\":\"message\",\"role\":\"assistant\",\"model\":%q,\"content\":[]}}\n\n", model)
			_, _ = fmt.Fprintln(w, `data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"ok"}}`)
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, `data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":1}}`)
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, `data: {"type":"message_stop"}`)
			return
		}

		if tools, ok := body["tools"].([]any); ok && len(tools) > 0 {
			writeClaudeMessage(t, w, model, []map[string]any{
				{"type": "tool_use", "id": "toolu_1", "name": "probe_ping", "input": map[string]any{"ok": true}},
			}, map[string]any{
				"input_tokens":  41178,
				"output_tokens": 5,
			})
			return
		}
		if claudeBodyContainsThinking(body) {
			writeClaudeMessage(t, w, model, []map[string]any{
				{"type": "text", "text": "ok"},
			}, map[string]any{
				"input_tokens":  24,
				"output_tokens": 1,
			})
			return
		}
		if claudeBodyContainsImage(body) {
			writeClaudeMessage(t, w, model, []map[string]any{
				{"type": "text", "text": "ok"},
			}, map[string]any{
				"input_tokens":  20,
				"output_tokens": 2,
			})
			return
		}

		auditRound++
		inputTokens := int64(30000 + auditRound*300)
		outputTokens := int64(100 + auditRound*20)
		writeClaudeMessage(t, w, model, []map[string]any{
			{"type": "text", "text": "ok"},
		}, map[string]any{
			"input_tokens":                inputTokens,
			"output_tokens":               outputTokens,
			"cache_creation_input_tokens": 0,
			"cache_read_input_tokens":     0,
		})
	}))
}

func writeClaudeMessage(t *testing.T, w http.ResponseWriter, model string, content []map[string]any, usage map[string]any) {
	t.Helper()
	writeJSON(t, w, map[string]any{
		"id":            "msg_1",
		"type":          "message",
		"role":          "assistant",
		"model":         model,
		"content":       content,
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"usage":         usage,
	})
}

func claudeBodyContainsImage(body map[string]any) bool {
	messages, _ := body["messages"].([]any)
	for _, rawMessage := range messages {
		message, _ := rawMessage.(map[string]any)
		content, _ := message["content"].([]any)
		for _, rawBlock := range content {
			block, _ := rawBlock.(map[string]any)
			if block["type"] == "image" {
				return true
			}
		}
	}
	return false
}

func claudeBodyContainsThinking(body map[string]any) bool {
	if _, ok := body["thinking"].(map[string]any); ok {
		return true
	}
	messages, _ := body["messages"].([]any)
	for _, rawMessage := range messages {
		message, _ := rawMessage.(map[string]any)
		content, _ := message["content"].([]any)
		for _, rawBlock := range content {
			block, _ := rawBlock.(map[string]any)
			if block["type"] == "thinking" {
				return true
			}
		}
	}
	return false
}

func claudeBodyHasThinkingBudgetViolation(body map[string]any) bool {
	thinking, _ := body["thinking"].(map[string]any)
	if thinking == nil {
		return false
	}
	maxTokens, ok := numericJSONValue(body["max_tokens"])
	if !ok || maxTokens <= 0 {
		return false
	}
	budgetTokens, ok := numericJSONValue(thinking["budget_tokens"])
	return ok && budgetTokens >= maxTokens
}

func claudeBodyHasCacheControlOverflow(body map[string]any) bool {
	blocks, _ := body["system"].([]any)
	count := 0
	for _, rawBlock := range blocks {
		block, _ := rawBlock.(map[string]any)
		if _, ok := block["cache_control"].(map[string]any); ok {
			count++
		}
	}
	return count >= 5
}

func numericJSONValue(value any) (int64, bool) {
	switch v := value.(type) {
	case float64:
		return int64(v), true
	case int:
		return int64(v), true
	case int64:
		return v, true
	default:
		return 0, false
	}
}

func TestOpenAITokenAuditFallbackPreservesUsageWhenStatefulParamsUnsupported(t *testing.T) {
	primaryAttempts := 0
	fallbackAttempts := 0
	responseIndex := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/responses", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		if _, ok := body["prompt_cache_key"].(string); ok {
			primaryAttempts++
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(t, w, map[string]any{"error": map[string]any{"message": "unsupported parameter: prompt_cache_key"}})
			return
		}
		fallbackAttempts++
		responseIndex++
		inputTokens := 200 + responseIndex
		outputTokens := 20 + responseIndex
		writeJSON(t, w, map[string]any{
			"id":     fmt.Sprintf("resp_fallback_%d", responseIndex),
			"object": "response",
			"output": []map[string]any{
				{"type": "message", "role": "assistant", "content": []map[string]any{{"type": "output_text", "text": "ok"}}},
			},
			"usage": map[string]any{
				"input_tokens":  inputTokens,
				"output_tokens": outputTokens,
				"total_tokens":  inputTokens + outputTokens,
			},
		})
	}))
	defer server.Close()

	service := NewService(nil)
	report := service.runTokenAudit(context.Background(), server.Client(), server.URL, "sk-test", "gpt-5.4", nil)

	require.Equal(t, tokenAuditSamples, primaryAttempts)
	require.Equal(t, tokenAuditSamples, fallbackAttempts)
	require.Equal(t, tokenAuditSamples, report.SampleCount)
	require.Greater(t, report.ActualCostUSD, float64(0))
	require.Equal(t, CheckStatusWarn, report.Status)
	require.Contains(t, report.Anomalies, "previous_response_chain_incomplete")
	require.Equal(t, CheckStatusPass, report.Samples[0].Status)
	require.Empty(t, report.Samples[0].PromptCacheKey)
	require.False(t, report.Samples[0].Store)
}

func TestTokenAuditSampleCapturesProbeFailureReason(t *testing.T) {
	body, err := json.Marshal(map[string]any{"error": map[string]any{"message": "unsupported parameter: prompt_cache_key"}})
	require.NoError(t, err)
	pricing := openAIModelPricingFor("gpt-5.4")

	sample := tokenAuditSampleFromProbe(3, httpProbe{
		StatusCode:   http.StatusBadRequest,
		Body:         body,
		LatencyMS:    12,
		ErrorClass:   "request_error",
		ErrorMessage: "unsupported parameter: prompt_cache_key",
	}, pricing, "resp_2", "cache_key", true)

	require.Equal(t, CheckStatusFail, sample.Status)
	require.Equal(t, http.StatusBadRequest, sample.StatusCode)
	require.Equal(t, "request_error", sample.ErrorClass)
	require.Equal(t, "unsupported parameter: prompt_cache_key", sample.ErrorMessage)
	require.Equal(t, "resp_2", sample.PreviousResponseID)
	require.Equal(t, "cache_key", sample.PromptCacheKey)
}

func writeOpenAITokenAuditTestResponse(t *testing.T, w http.ResponseWriter, body map[string]any, responseIndex *int) {
	t.Helper()
	require.NotNil(t, responseIndex)
	*responseIndex++
	index := *responseIndex
	require.Equal(t, true, body["store"])
	require.NotEmpty(t, body["prompt_cache_key"])
	require.NotContains(t, body, "tool_choice")
	require.NotContains(t, body, "parallel_tool_calls")
	require.NotContains(t, body, "tools")
	if index == 1 {
		require.NotContains(t, body, "previous_response_id")
	} else {
		require.Equal(t, fmt.Sprintf("resp_audit_%d", index-1), body["previous_response_id"])
	}

	cachedTokens := 0
	if index > 1 {
		cachedTokens = 640
	}
	inputTokens := 1600 + index*13
	outputTokens := 24 + index
	writeJSON(t, w, map[string]any{
		"id":     fmt.Sprintf("resp_audit_%d", index),
		"object": "response",
		"output": []map[string]any{
			{"type": "message", "role": "assistant", "content": []map[string]any{{"type": "output_text", "text": "ok"}}},
		},
		"usage": map[string]any{
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
			"total_tokens":  inputTokens + outputTokens,
			"input_tokens_details": map[string]any{
				"cached_tokens": cachedTokens,
			},
		},
	})
}

func newGeminiCompatibleServer(t *testing.T, expectedKey string, model string) *httptest.Server {
	t.Helper()
	modelPath := "/v1beta/models/" + model
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, expectedKey, r.Header.Get("x-goog-api-key"))
		switch r.URL.Path {
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

type accountResolverStub struct {
	account *coreservice.Account
	err     error
}

func (s accountResolverStub) GetByID(context.Context, int64) (*coreservice.Account, error) {
	return s.account, s.err
}
