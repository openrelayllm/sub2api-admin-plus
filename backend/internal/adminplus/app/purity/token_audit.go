package purity

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/tidwall/gjson"
)

type tokenAuditProbeResult struct {
	probe              httpProbe
	previousResponseID string
	promptCacheKey     string
	store              bool
}

func (s *Service) runTokenAudit(ctx context.Context, client *http.Client, baseURL string, apiKey string, model string, emitSample func(TokenAuditSample)) *TokenAuditReport {
	pricing := openAIModelPricingFor(model)
	auditNonce := newAuditNonce(model, s.currentTime())
	promptCacheKey := openAITokenAuditPromptCacheKey(model, auditNonce)
	report := &TokenAuditReport{
		Status:      CheckStatusWarn,
		Summary:     "Token 用量审计样本不足。",
		PriceSource: pricing.Source,
		Samples:     make([]TokenAuditSample, 0, tokenAuditSamples),
	}
	auditCtx, cancelAudit := context.WithTimeout(ctx, tokenAuditTimeout)
	defer cancelAudit()
	var totalOfficial float64
	var totalActual float64
	var totalUncached float64
	previousResponseID := ""
	for i := 1; i <= tokenAuditSamples; i++ {
		select {
		case <-auditCtx.Done():
			report.Summary = "Token 用量审计超时，已保留完成样本。"
			finalizeOpenAITokenAudit(report)
			return report
		default:
		}
		roundCtx, cancelRound := context.WithTimeout(auditCtx, tokenAuditRoundTimeout)
		probeResult := s.probeResponsesAudit(roundCtx, client, baseURL, apiKey, model, i, auditNonce, previousResponseID, promptCacheKey)
		cancelRound()
		sample := tokenAuditSampleFromProbe(i, probeResult.probe, pricing, probeResult.previousResponseID, probeResult.promptCacheKey, probeResult.store)
		report.Samples = append(report.Samples, sample)
		if sample.PromptCacheKey != "" {
			report.PromptCacheKey = sample.PromptCacheKey
		}
		if sample.Store {
			report.StoreEnabled = true
		}
		if emitSample != nil {
			emitSample(sample)
		}
		if sample.Status != CheckStatusPass {
			continue
		}
		report.InputTokens += sample.InputTokens
		report.OutputTokens += sample.OutputTokens
		report.CacheCreationTokens += sample.CacheCreationTokens
		report.CachedTokens += sample.CachedTokens
		totalOfficial += sample.OfficialBaselineUSD
		totalActual += sample.ActualCostUSD
		totalUncached += sample.UncachedBaselineUSD
		if sample.StateLinked {
			report.StatefulRounds++
		}
		if sample.ResponseID != "" {
			previousResponseID = sample.ResponseID
		}
	}
	report.OfficialBaselineUSD = roundMoney(totalOfficial)
	report.ActualCostUSD = roundMoney(totalActual)
	report.UncachedBaselineUSD = roundMoney(totalUncached)
	finalizeOpenAITokenAudit(report)
	return report
}

func (s *Service) probeResponsesAudit(ctx context.Context, client *http.Client, baseURL string, apiKey string, model string, round int, auditNonce string, previousResponseID string, promptCacheKey string) tokenAuditProbeResult {
	body := responsesAuditProbePayload(model, round, auditNonce, previousResponseID, promptCacheKey)
	probe := s.doJSON(ctx, client, http.MethodPost, buildOpenAIEndpointURL(baseURL, "/v1/responses"), apiKey, body, "application/json")
	if shouldRetryOpenAITokenAuditMinimal(probe) {
		fallbackProbe := s.doJSON(ctx, client, http.MethodPost, buildOpenAIEndpointURL(baseURL, "/v1/responses"), apiKey, responsesAuditMinimalProbePayload(model, round, auditNonce), "application/json")
		return tokenAuditProbeResult{probe: fallbackProbe}
	}
	return tokenAuditProbeResult{
		probe:              probe,
		previousResponseID: strings.TrimSpace(previousResponseID),
		promptCacheKey:     strings.TrimSpace(promptCacheKey),
		store:              true,
	}
}

func responsesAuditProbePayload(model string, round int, auditNonce string, previousResponseID string, promptCacheKey string) []byte {
	return responsesAuditProbePayloadWithOptions(model, round, auditNonce, previousResponseID, promptCacheKey, true)
}

func responsesAuditMinimalProbePayload(model string, round int, auditNonce string) []byte {
	return responsesAuditProbePayloadWithOptions(model, round, auditNonce, "", "", false)
}

func responsesAuditProbePayloadWithOptions(model string, round int, auditNonce string, previousResponseID string, promptCacheKey string, stateful bool) []byte {
	if strings.TrimSpace(model) == "" {
		model = openai.DefaultTestModel
	}
	bodyMap := map[string]any{
		"model":        model,
		"instructions": openAITokenAuditInstructions(auditNonce),
		"input": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": openAITokenAuditRoundInput(round, auditNonce)},
				},
			},
		},
		"max_output_tokens": tokenAuditOutputBudget(round),
		"stream":            false,
	}
	if stateful {
		bodyMap["store"] = true
		if strings.TrimSpace(promptCacheKey) != "" {
			bodyMap["prompt_cache_key"] = strings.TrimSpace(promptCacheKey)
		}
		if strings.TrimSpace(previousResponseID) != "" {
			bodyMap["previous_response_id"] = strings.TrimSpace(previousResponseID)
		}
	}
	body, _ := json.Marshal(bodyMap)
	return body
}

func shouldRetryOpenAITokenAuditMinimal(probe httpProbe) bool {
	return probe.StatusCode == http.StatusBadRequest || probe.StatusCode == http.StatusUnprocessableEntity
}

type openAIModelPricing struct {
	InputPerToken     float64
	OutputPerToken    float64
	CacheReadPerToken float64
	Source            string
}

func openAIModelPricingFor(model string) openAIModelPricing {
	key := strings.ToLower(strings.TrimSpace(model))
	table := openAIModelPricingTable()
	if pricing, ok := table[key]; ok {
		return pricing
	}
	var best openAIModelPricing
	bestLen := 0
	for prefix, pricing := range table {
		if strings.HasPrefix(key, prefix) && len(prefix) > bestLen {
			best = pricing
			bestLen = len(prefix)
		}
	}
	if bestLen > 0 {
		return best
	}
	return openAIModelPricing{
		InputPerToken:     2.5e-6,
		OutputPerToken:    15e-6,
		CacheReadPerToken: 0.25e-6,
		Source:            "Official OpenAI API pricing, Standard tier direct API, default gpt-5.5 baseline, verified 2026-06-27",
	}
}

func openAIModelPricingTable() map[string]openAIModelPricing {
	return map[string]openAIModelPricing{
		"gpt-5.5": {
			InputPerToken:     5e-6,
			OutputPerToken:    30e-6,
			CacheReadPerToken: 0.5e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5.5, verified 2026-06-27",
		},
		"gpt-5.5-pro": {
			InputPerToken:     30e-6,
			OutputPerToken:    180e-6,
			CacheReadPerToken: 30e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5.5-pro, cached input not separately listed, verified 2026-06-27",
		},
		"gpt-5.4-pro": {
			InputPerToken:     30e-6,
			OutputPerToken:    180e-6,
			CacheReadPerToken: 30e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5.4-pro, cached input not separately listed, verified 2026-06-27",
		},
		"gpt-5.4-mini": {
			InputPerToken:     0.75e-6,
			OutputPerToken:    4.5e-6,
			CacheReadPerToken: 0.075e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5.4-mini, verified 2026-06-27",
		},
		"gpt-5.4-nano": {
			InputPerToken:     0.2e-6,
			OutputPerToken:    1.25e-6,
			CacheReadPerToken: 0.02e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5.4-nano, verified 2026-06-27",
		},
		"gpt-5.4": {
			InputPerToken:     2.5e-6,
			OutputPerToken:    15e-6,
			CacheReadPerToken: 0.25e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5.4, verified 2026-06-27",
		},
		"gpt-5.2": {
			InputPerToken:     1.75e-6,
			OutputPerToken:    14e-6,
			CacheReadPerToken: 0.175e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5.2, verified 2026-06-27",
		},
		"gpt-5.2-pro": {
			InputPerToken:     21e-6,
			OutputPerToken:    168e-6,
			CacheReadPerToken: 21e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5.2-pro, cached input not separately listed, verified 2026-06-27",
		},
		"gpt-5.1": {
			InputPerToken:     1.25e-6,
			OutputPerToken:    10e-6,
			CacheReadPerToken: 0.125e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5.1, verified 2026-06-27",
		},
		"gpt-5-mini": {
			InputPerToken:     0.25e-6,
			OutputPerToken:    2e-6,
			CacheReadPerToken: 0.025e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5-mini, verified 2026-06-27",
		},
		"gpt-5-nano": {
			InputPerToken:     0.05e-6,
			OutputPerToken:    0.4e-6,
			CacheReadPerToken: 0.005e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5-nano, verified 2026-06-27",
		},
		"gpt-5-pro": {
			InputPerToken:     15e-6,
			OutputPerToken:    120e-6,
			CacheReadPerToken: 15e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5-pro, cached input not separately listed, verified 2026-06-27",
		},
		"gpt-5": {
			InputPerToken:     1.25e-6,
			OutputPerToken:    10e-6,
			CacheReadPerToken: 0.125e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5, verified 2026-06-27",
		},
		"chat-latest": {
			InputPerToken:     5e-6,
			OutputPerToken:    30e-6,
			CacheReadPerToken: 0.5e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, chat-latest, verified 2026-06-27",
		},
		"gpt-5.3-chat-latest": {
			InputPerToken:     1.75e-6,
			OutputPerToken:    14e-6,
			CacheReadPerToken: 0.175e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5.3-chat-latest, verified 2026-06-27",
		},
		"gpt-5.2-chat-latest": {
			InputPerToken:     1.75e-6,
			OutputPerToken:    14e-6,
			CacheReadPerToken: 0.175e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5.2-chat-latest, verified 2026-06-27",
		},
		"gpt-5.1-chat-latest": {
			InputPerToken:     1.25e-6,
			OutputPerToken:    10e-6,
			CacheReadPerToken: 0.125e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5.1-chat-latest, verified 2026-06-27",
		},
		"gpt-5-chat-latest": {
			InputPerToken:     1.25e-6,
			OutputPerToken:    10e-6,
			CacheReadPerToken: 0.125e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5-chat-latest, verified 2026-06-27",
		},
		"gpt-5.3-codex": {
			InputPerToken:     1.75e-6,
			OutputPerToken:    14e-6,
			CacheReadPerToken: 0.175e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5.3-codex, verified 2026-06-27",
		},
		"gpt-5.2-codex": {
			InputPerToken:     1.75e-6,
			OutputPerToken:    14e-6,
			CacheReadPerToken: 0.175e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5.2-codex, verified 2026-06-27",
		},
		"gpt-5.1-codex-max": {
			InputPerToken:     1.25e-6,
			OutputPerToken:    10e-6,
			CacheReadPerToken: 0.125e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5.1-codex-max, verified 2026-06-27",
		},
		"gpt-5.1-codex": {
			InputPerToken:     1.25e-6,
			OutputPerToken:    10e-6,
			CacheReadPerToken: 0.125e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5.1-codex, verified 2026-06-27",
		},
		"gpt-5-codex": {
			InputPerToken:     1.25e-6,
			OutputPerToken:    10e-6,
			CacheReadPerToken: 0.125e-6,
			Source:            "Official OpenAI API pricing, Standard tier direct API, gpt-5-codex, verified 2026-06-27",
		},
	}
}

func tokenAuditSampleFromProbe(index int, probe httpProbe, pricing openAIModelPricing, previousResponseID string, promptCacheKey string, store bool) TokenAuditSample {
	sample := TokenAuditSample{
		Index:              index,
		Round:              index,
		LatencyMS:          probe.LatencyMS,
		Status:             CheckStatusFail,
		StatusCode:         probe.StatusCode,
		ErrorClass:         strings.TrimSpace(probe.ErrorClass),
		ErrorMessage:       strings.TrimSpace(probe.ErrorMessage),
		PreviousResponseID: strings.TrimSpace(previousResponseID),
		PromptCacheKey:     strings.TrimSpace(promptCacheKey),
		Store:              store,
	}
	if probe.StatusCode < 200 || probe.StatusCode >= 300 {
		applyTokenAuditProbeFailure(&sample, probe, "Responses 用量审计请求未成功")
		return sample
	}
	usage := parseResponsesUsage(probe.Body)
	if usage == nil {
		applyTokenAuditProbeFailure(&sample, probe, "Responses 响应未返回 usage 字段")
		return sample
	}
	sample.InputTokens = usage.InputTokens
	sample.OutputTokens = usage.OutputTokens
	sample.CachedTokens = usage.CachedTokens
	sample.CacheCreationTokens = usage.CacheCreationTokens
	sample.CacheReadInputTokens = usage.CachedTokens
	sample.CacheCreationInputTokens = usage.CacheCreationTokens
	sample.UncachedInputTokens = maxInt64(0, usage.InputTokens-usage.CachedTokens)
	sample.ReasoningTokens = usage.ReasoningTokens
	sample.TotalTokens = usage.TotalTokens
	sample.ResponseID = strings.TrimSpace(gjson.GetBytes(probe.Body, "id").String())
	sample.StateLinked = sample.PreviousResponseID != "" && sample.ResponseID != ""
	sample.UncachedBaselineUSD = roundMoney(tokenCost(usage, pricing, false))
	sample.OfficialBaselineUSD = roundMoney(tokenCost(usage, pricing, true))
	sample.BaselineCostUSD = sample.OfficialBaselineUSD
	sample.ActualCostUSD = sample.OfficialBaselineUSD
	sample.CostUSD = sample.ActualCostUSD
	if sample.UncachedBaselineUSD > sample.ActualCostUSD {
		sample.CacheDiscountUSD = roundMoney(sample.UncachedBaselineUSD - sample.ActualCostUSD)
	}
	if sample.OfficialBaselineUSD > 0 {
		sample.Multiplier = roundRatio(sample.ActualCostUSD / sample.OfficialBaselineUSD)
		sample.Ratio = sample.Multiplier
	}
	sample.Status = CheckStatusPass
	applyTokenAuditSampleDerivedFields(&sample)
	return sample
}

func applyTokenAuditProbeFailure(sample *TokenAuditSample, probe httpProbe, fallbackMessage string) {
	if sample == nil {
		return
	}
	if sample.StatusCode == 0 {
		sample.StatusCode = probe.StatusCode
	}
	if sample.ErrorClass == "" {
		if probe.StatusCode >= 200 && probe.StatusCode < 300 {
			sample.ErrorClass = "usage_missing"
		} else {
			sample.ErrorClass = errorClassForStatus(probe.StatusCode)
		}
	}
	if sample.ErrorMessage == "" {
		sample.ErrorMessage = strings.TrimSpace(firstNonEmptyString(upstreamErrorMessage(probe.Body), fallbackMessage))
	}
	sample.ErrorMessage = sanitizeMessage(sample.ErrorMessage, "")
}

func tokenCost(usage *TokenUsage, pricing openAIModelPricing, cacheAware bool) float64 {
	if usage == nil {
		return 0
	}
	inputTokens := usage.InputTokens
	cachedTokens := usage.CachedTokens
	if cachedTokens < 0 {
		cachedTokens = 0
	}
	if cachedTokens > inputTokens {
		cachedTokens = inputTokens
	}
	billedInput := inputTokens
	cost := float64(billedInput) * pricing.InputPerToken
	if cacheAware && cachedTokens > 0 {
		uncached := inputTokens - cachedTokens
		cost = float64(uncached)*pricing.InputPerToken + float64(cachedTokens)*pricing.CacheReadPerToken
	}
	cost += float64(usage.OutputTokens) * pricing.OutputPerToken
	return cost
}

func finalizeTokenAudit(report *TokenAuditReport) {
	if report == nil {
		return
	}
	report.SampleCount = len(report.Samples)
	if report.OfficialBaselineUSD == 0 || report.ActualCostUSD == 0 {
		for _, sample := range report.Samples {
			if sample.Status != CheckStatusPass {
				continue
			}
			report.OfficialBaselineUSD += sample.OfficialBaselineUSD
			report.ActualCostUSD += sample.ActualCostUSD
		}
		report.OfficialBaselineUSD = roundMoney(report.OfficialBaselineUSD)
		report.ActualCostUSD = roundMoney(report.ActualCostUSD)
	}
	if report.OfficialBaselineUSD > 0 {
		report.Multiplier = roundRatio(report.ActualCostUSD / report.OfficialBaselineUSD)
		report.OverallRatio = report.Multiplier
	}
	report.BaselineTotalCostUSD = report.OfficialBaselineUSD
	report.TotalCostUSD = report.ActualCostUSD
	report.BaselineTotalCost = report.BaselineTotalCostUSD
	report.TotalCost = report.TotalCostUSD
	report.OverallRatioCompat = report.Multiplier
	for i := range report.Samples {
		applyTokenAuditSampleDerivedFields(&report.Samples[i])
	}
	report.Rows = append([]TokenAuditSample(nil), report.Samples...)
	cacheDenominator := report.InputTokens + report.CacheCreationTokens + report.CachedTokens
	if cacheDenominator > 0 {
		report.CacheHitRate = roundRatio(float64(report.CachedTokens) / float64(cacheDenominator))
	}
	report.CacheHitRatePercent = math.Round(report.CacheHitRate * 100)
	passed := 0
	for _, sample := range report.Samples {
		if sample.Status == CheckStatusPass {
			passed++
		}
	}
	ratioLooksNormal := report.Multiplier >= 0.5 && report.Multiplier <= 1.5
	cacheDiscountLooksNormal := report.CachedTokens > 0 && report.Multiplier >= 0.05 && report.Multiplier <= 1.5
	switch {
	case passed == tokenAuditSamples && (ratioLooksNormal || cacheDiscountLooksNormal):
		report.Status = CheckStatusPass
		report.Summary = "用量正常"
	case passed > 0:
		report.Status = CheckStatusWarn
		report.Summary = tokenAuditSummaryWithFailureHint("用量样本不完整或倍率波动", report.Samples)
		report.Anomalies = append(report.Anomalies, "sample_or_ratio_anomaly")
	default:
		report.Status = CheckStatusFail
		report.Summary = tokenAuditSummaryWithFailureHint("未获取到可审计 usage", report.Samples)
		report.Anomalies = append(report.Anomalies, "usage_missing")
	}
}

func tokenAuditSummaryWithFailureHint(summary string, samples []TokenAuditSample) string {
	for _, sample := range samples {
		if sample.Status == CheckStatusPass {
			continue
		}
		reason := strings.TrimSpace(firstNonEmptyString(sample.ErrorMessage, sample.ErrorClass))
		if reason == "" {
			continue
		}
		if len(reason) > 180 {
			reason = reason[:180]
		}
		if sample.StatusCode > 0 {
			return fmt.Sprintf("%s：第 %d 轮 HTTP %d %s", summary, sample.Round, sample.StatusCode, reason)
		}
		return fmt.Sprintf("%s：第 %d 轮 %s", summary, sample.Round, reason)
	}
	return summary
}

func finalizeOpenAITokenAudit(report *TokenAuditReport) {
	finalizeTokenAudit(report)
	if report == nil || report.SampleCount == 0 || report.Status == CheckStatusFail {
		return
	}
	expectedStatefulRounds := maxInt(0, minInt(tokenAuditSamples, report.SampleCount)-1)
	report.PreviousChainOK = expectedStatefulRounds == 0 || report.StatefulRounds >= expectedStatefulRounds
	if !report.PreviousChainOK {
		report.Status = CheckStatusWarn
		report.Summary = "Responses 状态链路不完整"
		report.Anomalies = appendUniqueString(report.Anomalies, "previous_response_chain_incomplete")
	}
	if report.CachedTokens == 0 {
		report.Status = CheckStatusWarn
		if report.PreviousChainOK {
			report.Summary = "状态链路正常，但未观察到 cached_tokens"
		}
		report.Anomalies = appendUniqueString(report.Anomalies, "cached_tokens_missing")
	}
	if report.UncachedBaselineUSD > 0 && report.ActualCostUSD > 0 {
		report.OverallRatio = roundRatio(report.ActualCostUSD / report.UncachedBaselineUSD)
		report.OverallRatioCompat = report.OverallRatio
	}
}

func applyTokenAuditSampleDerivedFields(sample *TokenAuditSample) {
	if sample == nil {
		return
	}
	if sample.BaselineCostUSD > 0 {
		sample.CostDeltaPct = roundDeltaPercent(sample.CostUSD, sample.BaselineCostUSD)
	}
	if sample.BaselineInputTokens > 0 {
		sample.InputDeltaPct = roundDeltaPercent(float64(sample.InputTokens), float64(sample.BaselineInputTokens))
	}
	if sample.BaselineOutputTokens > 0 {
		sample.OutputDeltaPct = roundDeltaPercent(float64(sample.OutputTokens), float64(sample.BaselineOutputTokens))
	}
	if sample.BaselineCacheCreation > 0 {
		sample.CacheCreationDeltaPct = roundDeltaPercent(float64(sample.CacheCreationInputTokens), float64(sample.BaselineCacheCreation))
	}
	if sample.BaselineCacheRead > 0 {
		sample.CacheReadDeltaPct = roundDeltaPercent(float64(sample.CacheReadInputTokens), float64(sample.BaselineCacheRead))
	}
}

func roundDeltaPercent(actual float64, baseline float64) float64 {
	if baseline <= 0 || math.IsNaN(actual) || math.IsNaN(baseline) || math.IsInf(actual, 0) || math.IsInf(baseline, 0) {
		return 0
	}
	return math.Round((actual - baseline) * 100 / baseline)
}

func openAITokenAuditPrompt(round int, auditNonce string) string {
	return strings.Join([]string{
		openAITokenAuditInstructions(auditNonce),
		auditCumulativeRoundText(round),
		tokenAuditRoundInstruction(round),
	}, "\n\n")
}

func openAITokenAuditInstructions(auditNonce string) string {
	return strings.Join([]string{
		"proxyai.best OpenAI token audit. Keep responses concise and do not call tools unless explicitly requested. audit_nonce=" + auditNonce,
		auditStableCacheText(auditNonce),
	}, "\n\n")
}

func openAITokenAuditRoundInput(round int, auditNonce string) string {
	return strings.Join([]string{
		fmt.Sprintf("Responses stateful audit round %02d. Use the prior response context when previous_response_id is present. audit_nonce=%s", clampAuditRound(round), auditNonce),
		auditRoundCacheText(round),
		tokenAuditRoundInstruction(round),
	}, "\n\n")
}

func openAITokenAuditPromptCacheKey(model string, auditNonce string) string {
	raw := strings.Join([]string{"proxyai.best", "openai-token-audit", strings.TrimSpace(model), auditNonce}, "\x00")
	hash := sha256Hex(raw)
	if len(hash) > 24 {
		hash = hash[:24]
	}
	return "proxyai_best_" + hash
}

func auditStableCacheText(auditNonce string) string {
	return "stable-cache-prefix " + auditNonce + " " + strings.Repeat("cache-anchor proxyai-best-purity "+auditNonce+" ", 620)
}

func auditCumulativeRoundText(round int) string {
	if round < 1 {
		round = 1
	}
	if round > tokenAuditSamples {
		round = tokenAuditSamples
	}
	parts := make([]string, 0, round)
	for i := 1; i <= round; i++ {
		parts = append(parts, auditRoundCacheText(i))
	}
	return strings.Join(parts, "\n")
}

func auditRoundCacheText(round int) string {
	repeats := []int{590, 27, 10, 58, 26, 47, 40, 173, 65, 81, 28}
	round = clampAuditRound(round)
	return fmt.Sprintf("round-cache-%02d ", round) + strings.Repeat(fmt.Sprintf("segment-%02d proxyai-best-cache ", round), repeats[round-1])
}

func tokenAuditRoundInstruction(round int) string {
	targets := []int{64, 48, 72, 56, 80, 64, 88, 56, 72, 48, 64}
	round = clampAuditRound(round)
	return fmt.Sprintf("Round %02d: write a compact comma-separated sequence of exactly %d lowercase letter x characters and nothing else.", round, targets[round-1])
}

func tokenAuditOutputBudget(round int) int {
	targets := []int{96, 80, 104, 88, 112, 96, 128, 88, 104, 80, 96}
	round = clampAuditRound(round)
	return targets[round-1]
}

func newAuditNonce(model string, at time.Time) string {
	raw := strings.Join([]string{"proxyai.best", model, at.UTC().Format(time.RFC3339Nano)}, "\x00")
	hash := sha256Hex(raw)
	if len(hash) > 12 {
		return hash[:12]
	}
	return hash
}
