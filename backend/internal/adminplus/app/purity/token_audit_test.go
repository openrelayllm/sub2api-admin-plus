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

func TestTokenAuditPayloadsSplitOpenAICacheAndContextReplayProbes(t *testing.T) {
	auditNonce := "audit-test-nonce"
	roundOnePrompt := openAITokenAuditPrompt(1, auditNonce)
	roundElevenPrompt := openAITokenAuditPrompt(11, auditNonce)
	require.Contains(t, roundOnePrompt, "stable-cache-prefix")
	require.Contains(t, roundOnePrompt, auditNonce)
	require.NotContains(t, roundOnePrompt, "previous_response_id")
	require.Contains(t, roundElevenPrompt, "stable-cache-prefix")

	promptCacheKey := openAITokenAuditPromptCacheKey("gpt-5.4", auditNonce)
	var cachePayload map[string]any
	require.NoError(t, json.Unmarshal(responsesAuditProbePayload("gpt-5.4", 2, auditNonce, "resp_audit_1", promptCacheKey), &cachePayload))
	require.Equal(t, "gpt-5.4", cachePayload["model"])
	require.NotContains(t, cachePayload, "store")
	require.NotContains(t, cachePayload, "previous_response_id")
	require.Equal(t, promptCacheKey, cachePayload["prompt_cache_key"])
	require.NotContains(t, cachePayload, "tool_choice")
	require.NotContains(t, cachePayload, "parallel_tool_calls")
	require.NotContains(t, cachePayload, "tools")
	require.Contains(t, cachePayload["instructions"], "stable-cache-prefix")
	require.NotContains(t, cachePayload["instructions"], "round-cache-02")
	input, ok := cachePayload["input"].([]any)
	require.True(t, ok)
	require.Len(t, input, 1)
	inputMessage, ok := input[0].(map[string]any)
	require.True(t, ok)
	content, ok := inputMessage["content"].([]any)
	require.True(t, ok)
	inputBlock, ok := content[0].(map[string]any)
	require.True(t, ok)
	require.Contains(t, inputBlock["text"], "long prefix is intentionally stable")
	require.NotContains(t, inputBlock["text"], "round-cache-02")

	var statefulPayload map[string]any
	require.NoError(t, json.Unmarshal(responsesAuditProbePayload("gpt-5.4", 10, auditNonce, "resp_audit_9", promptCacheKey), &statefulPayload))
	require.NotContains(t, statefulPayload, "store")
	require.NotContains(t, statefulPayload, "previous_response_id")
	require.Equal(t, promptCacheKey, statefulPayload["prompt_cache_key"])
	require.NotContains(t, statefulPayload["instructions"], "stable-cache-prefix")
	contextInput, ok := statefulPayload["input"].([]any)
	require.True(t, ok)
	require.Len(t, contextInput, 3)
	firstInput, ok := contextInput[0].(map[string]any)
	require.True(t, ok)
	secondInput, ok := contextInput[1].(map[string]any)
	require.True(t, ok)
	thirdInput, ok := contextInput[2].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "user", firstInput["role"])
	require.Equal(t, "assistant", secondInput["role"])
	require.Equal(t, "user", thirdInput["role"])

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
	require.Len(t, systemBlocks, 3)
	systemCacheControlled := 0
	for i, rawBlock := range systemBlocks {
		block, _ := rawBlock.(map[string]any)
		if _, ok := block["cache_control"].(map[string]any); ok {
			systemCacheControlled++
			require.GreaterOrEqual(t, i, 1)
		}
	}
	require.Equal(t, 2, systemCacheControlled)
	cliSystemBlock, ok := systemBlocks[1].(map[string]any)
	require.True(t, ok)
	require.Equal(t, claudeCodeSystemPrompt, cliSystemBlock["text"])
	cachedSystemBlock, ok := systemBlocks[2].(map[string]any)
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
	sample := claudeTokenAuditSampleFromProbe(2, httpProbe{StatusCode: http.StatusOK, Body: body, LatencyMS: 100}, pricing, 2)
	require.Equal(t, CheckStatusPass, sample.Status)
	require.Equal(t, claudeTokenAuditModeHistoryReplay, sample.RequestMode)
	require.True(t, sample.StateLinked)
	require.True(t, sample.CacheCreationFieldPresent)
	require.True(t, sample.CachedTokensFieldPresent)
	require.Equal(t, 2, sample.SystemCacheControlBlocks)
	require.Equal(t, 1, sample.MessageCacheControlBlocks)
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
	require.NotContains(t, report.Anomalies, "context_replay_incomplete")
	require.True(t, report.ContextReplayOK)
	require.Zero(t, report.ContextReplayLinksExpected)
	require.Equal(t, CheckStatusPass, report.Samples[0].Status)
	require.Empty(t, report.Samples[0].PromptCacheKey)
	require.False(t, report.Samples[0].Store)
	require.Equal(t, "minimal_retry", report.Samples[0].RequestMode)
	require.True(t, report.Samples[0].Retried)
}
func TestOpenAITokenAuditTreatsZeroCachedTokensFromOfficialShapeAsObservation(t *testing.T) {
	report := &TokenAuditReport{
		Status:      CheckStatusWarn,
		Summary:     "Token 用量审计样本不足。",
		PriceSource: "synthetic",
		Samples:     make([]TokenAuditSample, 0, tokenAuditSamples),
	}
	for i := 1; i <= tokenAuditSamples; i++ {
		mode := openAITokenAuditRequestMode(i)
		sample := TokenAuditSample{
			Index:                    i,
			Round:                    i,
			Status:                   CheckStatusPass,
			InputTokens:              1600,
			OutputTokens:             24,
			TotalTokens:              1624,
			OfficialBaselineUSD:      0.00436,
			ActualCostUSD:            0.00436,
			BaselineCostUSD:          0.00436,
			CostUSD:                  0.00436,
			Multiplier:               1,
			Ratio:                    1,
			RequestMode:              mode,
			PromptCacheKey:           "cache-key",
			CachedTokensFieldPresent: true,
		}
		if mode == openAITokenAuditModeContextReplay && i > openAITokenAuditCacheProbeRounds+1 {
			sample.StateLinked = true
		}
		report.Samples = append(report.Samples, sample)
	}

	finalizeOpenAITokenAudit(report)

	require.Equal(t, CheckStatusPass, report.Status)
	require.True(t, report.CachedTokensFieldObserved)
	require.Equal(t, openAITokenAuditCacheProbeRounds, report.CacheProbeRounds)
	require.Zero(t, report.CacheProbeHits)
	require.NotContains(t, report.Anomalies, "cached_tokens_missing")
	require.Contains(t, report.Summary, "本次未观察到 OpenAI 自动缓存命中")
}

func TestOpenAITokenAuditUsesUncachedBaselineAndInfersCacheCreation(t *testing.T) {
	pricing := openAIModelPricingFor("gpt-5.4")
	body, err := json.Marshal(map[string]any{
		"id":     "resp_cached",
		"object": "response",
		"output": []map[string]any{
			{"type": "message", "role": "assistant", "content": []map[string]any{{"type": "output_text", "text": "ok"}}},
		},
		"usage": map[string]any{
			"input_tokens":  1000,
			"output_tokens": 100,
			"total_tokens":  1100,
			"input_tokens_details": map[string]any{
				"cached_tokens": 900,
			},
		},
	})
	require.NoError(t, err)

	sample := tokenAuditSampleFromProbe(2, httpProbe{StatusCode: http.StatusOK, Body: body, LatencyMS: 100}, pricing, "", "cache-key", false, openAITokenAuditModeCacheProbe)
	require.Equal(t, CheckStatusPass, sample.Status)
	require.Greater(t, sample.OfficialBaselineUSD, sample.ActualCostUSD)
	require.Less(t, sample.Multiplier, float64(1))

	report := &TokenAuditReport{
		Samples: []TokenAuditSample{
			{Index: 1, Round: 1, Status: CheckStatusPass, RequestMode: openAITokenAuditModeCacheProbe, OfficialBaselineUSD: 0.01, ActualCostUSD: 0.01},
			sample,
		},
	}
	inferOpenAICacheCreationFromReads(report)
	require.Equal(t, int64(900), report.Samples[0].CacheCreationTokens)
	require.Equal(t, int64(900), report.Samples[0].CacheCreationInputTokens)
	require.Equal(t, int64(900), report.CacheCreationTokens)
}

func TestOpenAITokenAuditWarnsOnlyWhenCachedTokensFieldMissing(t *testing.T) {
	report := &TokenAuditReport{
		Status:      CheckStatusWarn,
		Summary:     "Token 用量审计样本不足。",
		PriceSource: "synthetic",
		Samples:     make([]TokenAuditSample, 0, tokenAuditSamples),
	}
	for i := 1; i <= tokenAuditSamples; i++ {
		mode := openAITokenAuditRequestMode(i)
		sample := TokenAuditSample{
			Index:               i,
			Round:               i,
			Status:              CheckStatusPass,
			InputTokens:         1600,
			OutputTokens:        24,
			TotalTokens:         1624,
			OfficialBaselineUSD: 0.00436,
			ActualCostUSD:       0.00436,
			BaselineCostUSD:     0.00436,
			CostUSD:             0.00436,
			Multiplier:          1,
			Ratio:               1,
			RequestMode:         mode,
			PromptCacheKey:      "cache-key",
		}
		if mode == openAITokenAuditModeContextReplay && i > openAITokenAuditCacheProbeRounds+1 {
			sample.StateLinked = true
		}
		report.Samples = append(report.Samples, sample)
	}

	finalizeOpenAITokenAudit(report)

	require.Equal(t, CheckStatusWarn, report.Status)
	require.False(t, report.CachedTokensFieldObserved)
	require.Contains(t, report.Anomalies, "cached_tokens_missing")
	require.Contains(t, report.Summary, "cached_tokens 字段")
}
func TestOpenAITokenAuditTimeoutDoesNotMinimalRetry(t *testing.T) {
	originalRoundTimeout := tokenAuditRoundTimeout
	tokenAuditRoundTimeout = 20 * time.Millisecond
	t.Cleanup(func() { tokenAuditRoundTimeout = originalRoundTimeout })

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/responses", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		attempts++
		if _, ok := body["prompt_cache_key"].(string); ok {
			<-r.Context().Done()
			return
		}
		writeJSON(t, w, map[string]any{
			"id":     "resp_timeout_fallback",
			"object": "response",
			"output": []map[string]any{
				{"type": "message", "role": "assistant", "content": []map[string]any{{"type": "output_text", "text": "ok"}}},
			},
			"usage": map[string]any{
				"input_tokens":  210,
				"output_tokens": 20,
				"total_tokens":  230,
			},
		})
	}))
	defer server.Close()

	service := NewService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	result := service.probeResponsesAudit(ctx, server.Client(), server.URL, "sk-test", "gpt-5.4", 1, "nonce", "", "cache-key")
	sample := tokenAuditSampleFromProbe(1, result.probe, openAIModelPricingFor("gpt-5.4"), result.previousResponseID, result.promptCacheKey, result.store, result.requestMode)
	sample.Retried = result.retried

	require.Equal(t, 1, attempts)
	require.Equal(t, CheckStatusFail, sample.Status)
	require.Equal(t, openAITokenAuditModeCacheProbe, sample.RequestMode)
	require.False(t, sample.Retried)
	require.False(t, sample.Store)
	require.Equal(t, "cache-key", sample.PromptCacheKey)
	require.Contains(t, sample.ErrorClass, "deadline")
}
func TestOpenAITokenAuditFallbackDoesNotRetryTimeout(t *testing.T) {
	require.False(t, shouldRetryOpenAITokenAuditMinimal(httpProbe{
		ErrorClass:   "response_header_timeout",
		ErrorMessage: `Post "https://example.test/v1/responses": net/http: timeout awaiting response headers`,
	}))
}
func TestErrorClassForRequestErrorDetectsResponseHeaderTimeout(t *testing.T) {
	err := &timeoutNetError{message: `Post "https://example.test/v1/responses": net/http: timeout awaiting response headers`}
	require.Equal(t, "response_header_timeout", errorClassForRequestError(context.Background(), err))
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
	}, pricing, "resp_2", "cache_key", true, openAITokenAuditModeStateful)

	require.Equal(t, CheckStatusFail, sample.Status)
	require.Equal(t, http.StatusBadRequest, sample.StatusCode)
	require.Equal(t, "request_error", sample.ErrorClass)
	require.Equal(t, "unsupported parameter: prompt_cache_key", sample.ErrorMessage)
	require.Equal(t, "resp_2", sample.PreviousResponseID)
	require.Equal(t, "cache_key", sample.PromptCacheKey)
}

type timeoutNetError struct {
	message string
}

func (e *timeoutNetError) Error() string {
	return e.message
}
func (e *timeoutNetError) Timeout() bool {
	return true
}
func (e *timeoutNetError) Temporary() bool {
	return true
}
func writeOpenAITokenAuditTestResponse(t *testing.T, w http.ResponseWriter, body map[string]any, responseIndex *int) {
	t.Helper()
	require.NotNil(t, responseIndex)
	*responseIndex++
	index := *responseIndex
	require.NotEmpty(t, body["prompt_cache_key"])
	require.NotContains(t, body, "tool_choice")
	require.NotContains(t, body, "parallel_tool_calls")
	require.NotContains(t, body, "tools")

	if index <= openAITokenAuditCacheProbeRounds {
		require.NotContains(t, body, "store")
		require.NotContains(t, body, "previous_response_id")
	} else {
		require.NotContains(t, body, "store")
		require.NotContains(t, body, "previous_response_id")
		input, ok := body["input"].([]any)
		require.True(t, ok)
		require.Len(t, input, (index-openAITokenAuditCacheProbeRounds)*2-1)
	}

	cachedTokens := 0
	if index > 1 && index <= openAITokenAuditCacheProbeRounds {
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
