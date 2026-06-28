package purity

import (
	"context"
	"github.com/tidwall/gjson"
	"math"
	"net/http"
	"strings"
)

const claudeTokenAuditModeHistoryReplay = "history_replay"

type claudeModelPricing struct {
	InputPerToken      float64
	OutputPerToken     float64
	CacheWritePerToken float64
	CacheReadPerToken  float64
	Source             string
}

func claudeModelPricingFor(model string) claudeModelPricing {
	key := strings.ToLower(strings.TrimSpace(model))
	if strings.Contains(key, "opus-4-8") || strings.Contains(key, "opus-4-7") || strings.Contains(key, "opus-4-6") || strings.Contains(key, "opus-4-5") {
		return claudeModelPricing{InputPerToken: 5e-6, OutputPerToken: 25e-6, CacheWritePerToken: 6.25e-6, CacheReadPerToken: 0.5e-6, Source: "Official Anthropic Claude pricing, Opus 4.x current generation, 5m cache writes and cache hits, verified 2026-06-27"}
	}
	if strings.Contains(key, "opus") {
		return claudeModelPricing{InputPerToken: 15e-6, OutputPerToken: 75e-6, CacheWritePerToken: 18.75e-6, CacheReadPerToken: 1.5e-6, Source: "Official Anthropic Claude pricing, legacy Opus class, 5m cache writes and cache hits, verified 2026-06-27"}
	}
	if strings.Contains(key, "haiku-4-5") || strings.Contains(key, "haiku-4.5") {
		return claudeModelPricing{InputPerToken: 1e-6, OutputPerToken: 5e-6, CacheWritePerToken: 1.25e-6, CacheReadPerToken: 0.1e-6, Source: "Official Anthropic Claude pricing, Haiku 4.5, 5m cache writes and cache hits, verified 2026-06-27"}
	}
	if strings.Contains(key, "haiku") {
		return claudeModelPricing{InputPerToken: 0.8e-6, OutputPerToken: 4e-6, CacheWritePerToken: 1e-6, CacheReadPerToken: 0.08e-6, Source: "Official Anthropic Claude pricing, legacy Haiku class, 5m cache writes and cache hits, verified 2026-06-27"}
	}
	return claudeModelPricing{InputPerToken: 3e-6, OutputPerToken: 15e-6, CacheWritePerToken: 3.75e-6, CacheReadPerToken: 0.3e-6, Source: "Official Anthropic Claude pricing, Sonnet 4.x, 5m cache writes and cache hits, verified 2026-06-27"}
}

func (s *Service) runClaudeTokenAudit(ctx context.Context, client *http.Client, baseURL string, apiKey string, model string, probeCtx claudeProbeContext, emitSample func(TokenAuditSample)) *TokenAuditReport {
	pricing := claudeModelPricingFor(model)
	auditNonce := newAuditNonce(model, s.currentTime())
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
	history := make([]claudeAuditTurn, 0, tokenAuditSamples)
	for i := 1; i <= tokenAuditSamples; i++ {
		select {
		case <-auditCtx.Done():
			report.Summary = "Token 用量审计超时，已保留完成样本。"
			finalizeClaudeTokenAudit(report)
			return report
		default:
		}
		roundCtx, cancelRound := context.WithTimeout(auditCtx, tokenAuditRoundTimeout)
		historyMessages := len(history) * 2
		probe := s.probeClaudeAudit(roundCtx, client, baseURL, apiKey, model, i, auditNonce, probeCtx, history)
		cancelRound()
		sample := claudeTokenAuditSampleFromProbe(i, probe, pricing, historyMessages)
		report.Samples = append(report.Samples, sample)
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
		if sample.CacheCreationFieldPresent {
			report.CacheCreationFieldObserved = true
		}
		if sample.CachedTokensFieldPresent {
			report.CacheReadFieldObserved = true
			report.CachedTokensFieldObserved = true
		}
		totalOfficial += sample.OfficialBaselineUSD
		totalActual += sample.ActualCostUSD
		if assistantText := claudeAssistantTextFromBody(probe.Body); assistantText != "" {
			history = append(history, claudeAuditTurn{
				Round:         i,
				UserText:      claudeAuditUserText(i, auditNonce),
				AssistantText: assistantText,
			})
		}
	}
	report.OfficialBaselineUSD = roundMoney(totalOfficial)
	report.ActualCostUSD = roundMoney(totalActual)
	finalizeClaudeTokenAudit(report)
	return report
}

func finalizeClaudeTokenAudit(report *TokenAuditReport) {
	finalizeTokenAudit(report)
	applyClaudeHistoryReplayStats(report)
	applyClaudeWarmCacheHitRate(report)
	applyClaudeTokenAuditAnomalies(report)
}

func applyClaudeHistoryReplayStats(report *TokenAuditReport) {
	if report == nil || len(report.Samples) == 0 {
		return
	}
	attempted := 0
	passed := 0
	linked := 0
	for _, sample := range report.Samples {
		if sample.RequestMode != claudeTokenAuditModeHistoryReplay {
			continue
		}
		if sample.Round > 1 {
			attempted++
		}
		if sample.Status != CheckStatusPass {
			continue
		}
		if sample.Round > 1 {
			passed++
		}
		if sample.StateLinked {
			linked++
		}
	}
	report.HistoryReplayRounds = passed
	report.HistoryReplayLinks = linked
	report.HistoryReplayLinksExpected = attempted
	report.HistoryReplayOK = report.HistoryReplayLinksExpected == 0 || report.HistoryReplayLinks >= report.HistoryReplayLinksExpected
	report.ContextReplayRounds = report.HistoryReplayRounds
	report.ContextReplayLinks = report.HistoryReplayLinks
	report.ContextReplayLinksExpected = report.HistoryReplayLinksExpected
	report.ContextReplayOK = report.HistoryReplayOK
	report.StatefulRounds = report.HistoryReplayLinks
	report.PreviousChainOK = report.HistoryReplayOK
	if !report.HistoryReplayOK {
		report.Status = CheckStatusWarn
		report.Summary = "Claude 历史消息重放不完整"
		report.Anomalies = appendUniqueString(report.Anomalies, "claude_history_replay_incomplete")
	}
}

func applyClaudeWarmCacheHitRate(report *TokenAuditReport) {
	if report == nil || len(report.Samples) < 2 {
		return
	}
	var inputTokens int64
	var cacheCreationTokens int64
	var cachedTokens int64
	for _, sample := range report.Samples[1:] {
		if sample.Status != CheckStatusPass {
			continue
		}
		inputTokens += sample.InputTokens
		cacheCreationTokens += sample.CacheCreationTokens
		cachedTokens += sample.CachedTokens
	}
	denominator := inputTokens + cacheCreationTokens + cachedTokens
	if denominator <= 0 {
		return
	}
	report.CacheHitRate = roundRatio(float64(cachedTokens) / float64(denominator))
	report.CacheHitRatePercent = math.Round(report.CacheHitRate * 100)
}

func applyClaudeTokenAuditAnomalies(report *TokenAuditReport) {
	if report == nil || report.SampleCount == 0 || report.Status == CheckStatusFail {
		return
	}
	passed := 0
	cacheShapeBad := false
	fieldShapeBad := false
	costShapeBad := false
	for _, sample := range report.Samples {
		if sample.Status != CheckStatusPass {
			continue
		}
		passed++
		if sample.BaselineCacheCreation > 0 && !sample.CacheCreationFieldPresent {
			fieldShapeBad = true
		}
		if sample.BaselineCacheRead > 0 && !sample.CachedTokensFieldPresent {
			fieldShapeBad = true
		}
		if sample.BaselineCacheCreation > 0 && sample.CacheCreationInputTokens == 0 {
			cacheShapeBad = true
		}
		if sample.BaselineCacheRead > 0 && sample.CacheReadInputTokens == 0 {
			cacheShapeBad = true
		}
		if sample.Multiplier > 0 && (sample.Multiplier < 0.5 || sample.Multiplier > 1.5) {
			costShapeBad = true
		}
	}
	if passed == 0 {
		return
	}
	if fieldShapeBad {
		report.Status = CheckStatusWarn
		report.Summary = "Claude usage 未返回 cache_creation/cache_read 字段"
		report.Anomalies = appendUniqueString(report.Anomalies, "claude_cache_usage_fields_missing")
	}
	if cacheShapeBad {
		report.Status = CheckStatusWarn
		if !fieldShapeBad {
			report.Summary = "Claude 缓存计量形态异常"
		}
		report.Anomalies = appendUniqueString(report.Anomalies, "claude_cache_accounting_missing")
	}
	if costShapeBad {
		report.Status = CheckStatusWarn
		if !cacheShapeBad && !fieldShapeBad {
			report.Summary = "Claude Token 成本倍率异常"
		}
		report.Anomalies = appendUniqueString(report.Anomalies, "cost_multiplier_anomaly")
	}
}

func claudeTokenAuditSampleFromProbe(index int, probe httpProbe, pricing claudeModelPricing, historyMessages int) TokenAuditSample {
	sample := TokenAuditSample{
		Index:                     index,
		Round:                     index,
		LatencyMS:                 probe.LatencyMS,
		Status:                    CheckStatusFail,
		StatusCode:                probe.StatusCode,
		ErrorClass:                strings.TrimSpace(probe.ErrorClass),
		ErrorMessage:              strings.TrimSpace(probe.ErrorMessage),
		RequestMode:               claudeTokenAuditModeHistoryReplay,
		HistoryMessages:           historyMessages,
		SystemCacheControlBlocks:  2,
		MessageCacheControlBlocks: 1,
	}
	expectedHistoryMessages := maxInt(0, (index-1)*2)
	sample.StateLinked = expectedHistoryMessages > 0 && historyMessages >= expectedHistoryMessages
	applyClaudeTokenAuditBaseline(&sample, pricing)
	if probe.StatusCode < 200 || probe.StatusCode >= 300 {
		applyTokenAuditProbeFailure(&sample, probe, "Claude 用量审计请求未成功")
		return sample
	}
	usage := parseClaudeUsage(probe.Body)
	if usage == nil {
		applyTokenAuditProbeFailure(&sample, probe, "Claude 响应未返回 usage 字段")
		return sample
	}
	sample.InputTokens = usage.InputTokens
	sample.OutputTokens = usage.OutputTokens
	sample.CachedTokens = usage.CachedTokens
	sample.CacheReadInputTokens = usage.CachedTokens
	sample.CacheCreationTokens = usage.CacheCreationTokens
	sample.CacheCreationInputTokens = usage.CacheCreationTokens
	sample.CacheCreationFieldPresent = usage.CacheCreationFieldPresent
	sample.CachedTokensFieldPresent = usage.CachedTokensFieldPresent
	sample.UncachedInputTokens = usage.InputTokens
	sample.TotalTokens = usage.TotalTokens
	sample.ActualCostUSD = roundMoney(claudeTokenCost(usage, pricing, true))
	sample.CostUSD = sample.ActualCostUSD
	if sample.OfficialBaselineUSD > 0 {
		sample.Multiplier = roundRatio(sample.ActualCostUSD / sample.OfficialBaselineUSD)
		sample.Ratio = sample.Multiplier
	}
	sample.Status = CheckStatusPass
	applyTokenAuditSampleDerivedFields(&sample)
	return sample
}

func applyClaudeTokenAuditBaseline(sample *TokenAuditSample, pricing claudeModelPricing) {
	if sample == nil {
		return
	}
	baseline := claudeTokenAuditBaselineUsage(sample.Index)
	if baseline == nil {
		return
	}
	sample.BaselineInputTokens = baseline.InputTokens
	sample.BaselineOutputTokens = baseline.OutputTokens
	sample.BaselineCacheCreation = baseline.CacheCreationTokens
	sample.BaselineCacheRead = baseline.CachedTokens
	sample.OfficialBaselineUSD = roundMoney(claudeTokenCost(baseline, pricing, true))
	sample.BaselineCostUSD = sample.OfficialBaselineUSD
}

func claudeTokenAuditBaselineUsage(round int) *TokenUsage {
	round = clampAuditRound(round)
	inputTokens := []int64{2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	outputTokens := []int64{152, 162, 770, 336, 616, 668, 1413, 783, 128, 111, 387}
	cacheCreationTokens := []int64{21300, 422, 169, 822, 400, 676, 572, 2441, 919, 1135, 418}
	cacheReadTokens := []int64{15590, 23911, 24291, 24441, 25263, 25643, 26319, 26891, 29332, 30251, 31386}
	return &TokenUsage{
		InputTokens:         inputTokens[round-1],
		OutputTokens:        outputTokens[round-1],
		CacheCreationTokens: cacheCreationTokens[round-1],
		CachedTokens:        cacheReadTokens[round-1],
		TotalTokens:         inputTokens[round-1] + outputTokens[round-1] + cacheCreationTokens[round-1] + cacheReadTokens[round-1],
	}
}

func claudeAssistantTextFromBody(body []byte) string {
	content := gjson.GetBytes(body, "content")
	if !content.IsArray() {
		return ""
	}
	var parts []string
	for _, item := range content.Array() {
		if strings.TrimSpace(item.Get("type").String()) != "text" {
			continue
		}
		text := strings.TrimSpace(item.Get("text").String())
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func claudeTokenCost(usage *TokenUsage, pricing claudeModelPricing, cacheAware bool) float64 {
	if usage == nil {
		return 0
	}
	cost := float64(usage.InputTokens) * pricing.InputPerToken
	if cacheAware {
		cost += float64(usage.CacheCreationTokens) * pricing.CacheWritePerToken
		cost += float64(usage.CachedTokens) * pricing.CacheReadPerToken
	} else {
		cost += float64(usage.CacheCreationTokens+usage.CachedTokens) * pricing.InputPerToken
	}
	cost += float64(usage.OutputTokens) * pricing.OutputPerToken
	return cost
}
