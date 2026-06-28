package purity

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
	require.True(t, report.TokenAudit.HistoryReplayOK)
	require.Equal(t, tokenAuditSamples-1, report.TokenAudit.HistoryReplayRounds)
	require.Equal(t, tokenAuditSamples-1, report.TokenAudit.HistoryReplayLinks)
	require.Equal(t, tokenAuditSamples-1, report.TokenAudit.HistoryReplayLinksExpected)
	require.True(t, report.TokenAudit.CacheCreationFieldObserved)
	require.True(t, report.TokenAudit.CacheReadFieldObserved)
	require.Equal(t, claudeTokenAuditModeHistoryReplay, report.TokenAudit.Samples[1].RequestMode)
	require.Contains(t, eventTypes(events), PublicCheckEventProgress)
	require.Contains(t, eventTypes(events), PublicCheckEventTokenAuditSample)
	require.Contains(t, progressStepNames(events), "signature")
	require.Equal(t, PublicCheckEventReport, events[len(events)-1].Type)
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
func newClaudeCompatibleServer(t *testing.T, expectedKey string) *httptest.Server {
	t.Helper()
	auditRound := 0
	sessionID := ""
	deviceID := ""
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/usage" {
			require.Equal(t, expectedKey, r.Header.Get("x-api-key"))
			http.NotFound(w, r)
			return
		}
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
		messages, _ := body["messages"].([]any)
		require.Len(t, messages, auditRound*2-1)
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
		if r.URL.Path == "/v1/usage" {
			require.Equal(t, expectedKey, r.Header.Get("x-api-key"))
			http.NotFound(w, r)
			return
		}
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
