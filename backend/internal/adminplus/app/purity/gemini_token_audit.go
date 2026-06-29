package purity

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

const geminiTokenAuditModeHistoryReplay = "gemini_history_replay"
const geminiTokenAuditModeMinimalRetry = "gemini_minimal_retry"

type geminiAuditTurn struct {
	Round         int
	UserText      string
	AssistantText string
}

type geminiModelPricing struct {
	InputPerToken     float64
	OutputPerToken    float64
	CacheReadPerToken float64
	Source            string
}

func geminiModelPricingFor(model string) geminiModelPricing {
	key := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(key, "flash-lite") || strings.Contains(key, "lite"):
		return geminiModelPricing{InputPerToken: 0.10e-6, OutputPerToken: 0.40e-6, CacheReadPerToken: 0.025e-6, Source: "Google Gemini API pricing, Gemini Flash Lite class baseline, verified 2026-06-28"}
	case strings.Contains(key, "flash"):
		return geminiModelPricing{InputPerToken: 0.30e-6, OutputPerToken: 2.50e-6, CacheReadPerToken: 0.075e-6, Source: "Google Gemini API pricing, Gemini Flash class baseline, verified 2026-06-28"}
	case strings.Contains(key, "pro"):
		return geminiModelPricing{InputPerToken: 1.25e-6, OutputPerToken: 10e-6, CacheReadPerToken: 0.31e-6, Source: "Google Gemini API pricing, Gemini Pro class baseline, verified 2026-06-28"}
	default:
		return geminiModelPricing{InputPerToken: 0.30e-6, OutputPerToken: 2.50e-6, CacheReadPerToken: 0.075e-6, Source: "Google Gemini API pricing, Gemini Flash default baseline, verified 2026-06-28"}
	}
}

func (s *Service) runGeminiTokenAudit(ctx context.Context, client *http.Client, baseURL string, apiKey string, model string, emitSample func(TokenAuditSample)) *TokenAuditReport {
	pricing := geminiModelPricingFor(model)
	auditNonce := newAuditNonce(model, s.currentTime())
	report := &TokenAuditReport{
		Status:      CheckStatusWarn,
		Summary:     "Gemini usageMetadata 审计样本不足。",
		PriceSource: pricing.Source,
		Samples:     make([]TokenAuditSample, 0, tokenAuditSamples),
	}
	auditCtx, cancelAudit := context.WithTimeout(ctx, tokenAuditTimeout)
	defer cancelAudit()
	history := make([]geminiAuditTurn, 0, tokenAuditSamples)
	for i := 1; i <= tokenAuditSamples; i++ {
		select {
		case <-auditCtx.Done():
			report.Summary = "Gemini Token 用量审计超时，已保留完成样本。"
			finalizeGeminiTokenAudit(report)
			return report
		default:
		}
		roundCtx, cancelRound := context.WithTimeout(auditCtx, tokenAuditRoundTimeout)
		historyMessages := len(history) * 2
		probeResult := s.probeGeminiTokenAudit(roundCtx, client, baseURL, apiKey, model, i, auditNonce, history)
		cancelRound()
		sample := geminiTokenAuditSampleFromProbe(i, probeResult.probe, pricing, historyMessages, probeResult.requestMode)
		sample.Retried = probeResult.retried
		report.Samples = append(report.Samples, sample)
		if emitSample != nil {
			emitSample(sample)
		}
		if sample.Status != CheckStatusPass {
			continue
		}
		report.InputTokens += sample.InputTokens
		report.OutputTokens += sample.OutputTokens
		report.CachedTokens += sample.CachedTokens
		if sample.CachedTokensFieldPresent {
			report.CachedTokensFieldObserved = true
			report.CacheReadFieldObserved = true
		}
		report.OfficialBaselineUSD += sample.OfficialBaselineUSD
		report.UncachedBaselineUSD += sample.UncachedBaselineUSD
		report.ActualCostUSD += sample.ActualCostUSD
		if assistantText := geminiAssistantTextFromBody(probeResult.probe.Body); assistantText != "" {
			history = append(history, geminiAuditTurn{
				Round:         i,
				UserText:      geminiAuditUserText(i, auditNonce),
				AssistantText: assistantText,
			})
		}
	}
	report.OfficialBaselineUSD = roundMoney(report.OfficialBaselineUSD)
	report.UncachedBaselineUSD = roundMoney(report.UncachedBaselineUSD)
	report.ActualCostUSD = roundMoney(report.ActualCostUSD)
	finalizeGeminiTokenAudit(report)
	return report
}

type geminiTokenAuditProbeResult struct {
	probe       httpProbe
	requestMode string
	retried     bool
}

func (s *Service) probeGeminiTokenAudit(ctx context.Context, client *http.Client, baseURL string, apiKey string, model string, round int, auditNonce string, history []geminiAuditTurn) geminiTokenAuditProbeResult {
	probe := s.doGeminiJSON(ctx, client, http.MethodPost, buildGeminiGenerateURL(baseURL, model, "generateContent", false), apiKey, geminiAuditProbePayload(model, round, auditNonce, history), "application/json")
	if shouldRetryGeminiTokenAuditMinimal(probe) {
		fallbackProbe := s.doGeminiJSON(ctx, client, http.MethodPost, buildGeminiGenerateURL(baseURL, model, "generateContent", false), apiKey, geminiAuditMinimalProbePayload(round, auditNonce), "application/json")
		return geminiTokenAuditProbeResult{probe: fallbackProbe, requestMode: geminiTokenAuditModeMinimalRetry, retried: true}
	}
	return geminiTokenAuditProbeResult{probe: probe, requestMode: geminiTokenAuditModeHistoryReplay}
}

func geminiAuditProbePayload(model string, round int, auditNonce string, history []geminiAuditTurn) []byte {
	body, _ := json.Marshal(map[string]any{
		"contents":          geminiAuditContents(round, auditNonce, history),
		"systemInstruction": geminiAuditSystemInstruction(auditNonce),
		"generationConfig": map[string]any{
			"maxOutputTokens": tokenAuditOutputBudget(round),
			"temperature":     0,
		},
	})
	return body
}

func geminiAuditMinimalProbePayload(round int, auditNonce string) []byte {
	body, _ := json.Marshal(map[string]any{
		"contents": []map[string]any{
			geminiTextContent("user", strings.Join([]string{
				"Gemini usageMetadata minimal audit.",
				"audit_nonce=" + auditNonce,
				tokenAuditRoundInstruction(round),
			}, "\n")),
		},
		"generationConfig": map[string]any{
			"maxOutputTokens": tokenAuditOutputBudget(round),
			"temperature":     0,
		},
	})
	return body
}

func shouldRetryGeminiTokenAuditMinimal(probe httpProbe) bool {
	return probe.StatusCode == http.StatusBadRequest ||
		probe.StatusCode == http.StatusUnprocessableEntity ||
		strings.Contains(strings.ToLower(firstNonEmptyString(probe.ErrorMessage, upstreamErrorMessage(probe.Body))), "top-level must be a list")
}

func geminiAuditSystemInstruction(auditNonce string) map[string]any {
	return map[string]any{
		"parts": []map[string]any{
			{"text": strings.Join([]string{
				"proxyai.best Gemini CLI token audit.",
				"Use Gemini GenerateContent contents history exactly as provided.",
				"Do not call tools unless explicitly requested.",
				"audit_nonce=" + auditNonce,
			}, " ")},
		},
	}
}

func geminiAuditContents(round int, auditNonce string, history []geminiAuditTurn) []map[string]any {
	contents := make([]map[string]any, 0, len(history)*2+1)
	for _, turn := range history {
		userText := strings.TrimSpace(turn.UserText)
		assistantText := strings.TrimSpace(turn.AssistantText)
		if userText == "" || assistantText == "" {
			continue
		}
		contents = append(contents,
			geminiTextContent("user", userText),
			geminiTextContent("model", assistantText),
		)
	}
	contents = append(contents, geminiTextContent("user", geminiAuditUserText(round, auditNonce)))
	return contents
}

func geminiTextContent(role string, text string) map[string]any {
	return map[string]any{
		"role": role,
		"parts": []map[string]any{
			{"text": text},
		},
	}
}

func geminiAuditUserText(round int, auditNonce string) string {
	return strings.Join([]string{
		"Gemini CLI history replay audit.",
		"Round marker=" + geminiAuditRoundMarker(round, auditNonce) + ".",
		tokenAuditRoundInstruction(round),
	}, "\n")
}

func geminiAuditAssistantMemory(round int, auditNonce string) string {
	return "Gemini history replay memory " + geminiAuditRoundMarker(round, auditNonce) + " acknowledged."
}

func geminiAuditRoundMarker(round int, auditNonce string) string {
	return strings.TrimSpace(auditNonce) + "-round-" + twoDigitRound(round)
}

func twoDigitRound(round int) string {
	return fmt.Sprintf("%02d", clampAuditRound(round))
}

func geminiTokenAuditSampleFromProbe(index int, probe httpProbe, pricing geminiModelPricing, historyMessages int, requestMode string) TokenAuditSample {
	sample := TokenAuditSample{
		Index:           index,
		Round:           index,
		LatencyMS:       probe.LatencyMS,
		Status:          CheckStatusFail,
		StatusCode:      probe.StatusCode,
		ErrorClass:      strings.TrimSpace(probe.ErrorClass),
		ErrorMessage:    strings.TrimSpace(probe.ErrorMessage),
		RequestMode:     requestMode,
		HistoryMessages: historyMessages,
	}
	expectedHistoryMessages := maxInt(0, (index-1)*2)
	sample.StateLinked = expectedHistoryMessages > 0 && historyMessages >= expectedHistoryMessages
	if probe.StatusCode < 200 || probe.StatusCode >= 300 {
		applyTokenAuditProbeFailure(&sample, probe, "Gemini 用量审计请求未成功")
		return sample
	}
	usage := parseGeminiUsage(probe.Body)
	if usage == nil {
		applyTokenAuditProbeFailure(&sample, probe, "Gemini 响应未返回 usageMetadata 字段")
		return sample
	}
	sample.InputTokens = usage.InputTokens
	sample.OutputTokens = usage.OutputTokens
	sample.CachedTokens = usage.CachedTokens
	sample.CacheReadInputTokens = usage.CachedTokens
	sample.CachedTokensFieldPresent = usage.CachedTokensFieldPresent
	sample.UncachedInputTokens = maxInt64(0, usage.InputTokens-usage.CachedTokens)
	sample.TotalTokens = usage.TotalTokens
	sample.UncachedBaselineUSD = roundMoney(geminiTokenCost(usage, pricing, false))
	sample.OfficialBaselineUSD = roundMoney(geminiTokenCost(usage, pricing, true))
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

func finalizeGeminiTokenAudit(report *TokenAuditReport) {
	finalizeTokenAudit(report)
	applyGeminiHistoryReplayStats(report)
	applyGeminiUsageMetadataAnomalies(report)
	geminiCacheHitRate(report)
}

func applyGeminiHistoryReplayStats(report *TokenAuditReport) {
	if report == nil || len(report.Samples) == 0 {
		return
	}
	attempted := 0
	passed := 0
	linked := 0
	for _, sample := range report.Samples {
		if sample.RequestMode != geminiTokenAuditModeHistoryReplay {
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
		report.Summary = "Gemini contents 历史重放不完整"
		report.Anomalies = appendUniqueString(report.Anomalies, "gemini_history_replay_incomplete")
	}
}

func applyGeminiUsageMetadataAnomalies(report *TokenAuditReport) {
	if report == nil || report.SampleCount == 0 || report.Status == CheckStatusFail {
		return
	}
	passed := 0
	totalShapeBad := false
	for _, sample := range report.Samples {
		if sample.Status != CheckStatusPass {
			continue
		}
		passed++
		if sample.TotalTokens <= 0 || sample.TotalTokens < sample.InputTokens+sample.OutputTokens {
			totalShapeBad = true
		}
	}
	if passed == 0 {
		return
	}
	if totalShapeBad {
		report.Status = CheckStatusWarn
		report.Summary = "Gemini usageMetadata token 汇总异常"
		report.Anomalies = appendUniqueString(report.Anomalies, "gemini_usage_metadata_shape_anomaly")
		return
	}
	if report.Status == CheckStatusPass {
		report.Summary = "Gemini usageMetadata 正常，contents 历史重放完整"
	}
}

func geminiAssistantTextFromBody(body []byte) string {
	parts := gjson.GetBytes(body, "candidates.0.content.parts")
	if !parts.IsArray() {
		return ""
	}
	texts := make([]string, 0, len(parts.Array()))
	for _, item := range parts.Array() {
		text := strings.TrimSpace(item.Get("text").String())
		if text != "" {
			texts = append(texts, text)
		}
	}
	return strings.Join(texts, "\n")
}

func geminiTokenCost(usage *TokenUsage, pricing geminiModelPricing, cacheAware bool) float64 {
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
	cost := float64(inputTokens) * pricing.InputPerToken
	if cacheAware && cachedTokens > 0 {
		uncached := inputTokens - cachedTokens
		cost = float64(uncached)*pricing.InputPerToken + float64(cachedTokens)*pricing.CacheReadPerToken
	}
	cost += float64(usage.OutputTokens) * pricing.OutputPerToken
	return cost
}

func geminiCacheHitRate(report *TokenAuditReport) {
	if report == nil {
		return
	}
	if report.InputTokens <= 0 {
		return
	}
	report.CacheHitRate = roundRatio(float64(report.CachedTokens) / float64(report.InputTokens))
	report.CacheHitRatePercent = math.Round(report.CacheHitRate * 100)
}
