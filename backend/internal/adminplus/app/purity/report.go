package purity

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *Service) finalizeAndSave(ctx context.Context, report *PublicReport, baseURL string) {
	finalizeReport(report)
	if s == nil || s.repo == nil || report == nil {
		return
	}
	_ = s.repo.SavePublicReport(ctx, PublicReportRecord{
		RequestHash:       reportHash(report, baseURL),
		Provider:          report.Provider,
		APIBaseHost:       report.APIBaseHost,
		Report:            report,
		PublicSummaryJSON: buildPublicSummary(report),
	})
}

func finalizeReport(report *PublicReport) {
	if report == nil {
		return
	}
	totalScore := 0
	totalMax := 0
	capabilityScore := 0
	capabilityMax := 0
	for _, check := range report.Checks {
		if check.MaxScore <= 0 {
			continue
		}
		totalScore += check.Score
		totalMax += check.MaxScore
		if check.ID != "base_url" {
			capabilityScore += check.Score
			capabilityMax += check.MaxScore
		}
	}
	report.CompatibilityScore = percent(capabilityScore, capabilityMax)
	finalizeValidations(report)
	report.Scores = scoreBreakdown(report)
	report.OfficialScore = officialPurityScore(report, percent(totalScore, totalMax))
	report.Score = report.OfficialScore
	report.Verdict = decideVerdict(report)
	report.Summary = summaryForReport(report)
	if report.Status == RunStatusError && report.Metrics.ErrorClass == "" {
		report.Metrics.ErrorClass, report.Metrics.ErrorMessage = firstProbeError(report.Checks)
	}
	if report.Status == RunStatusError && report.Error == "" {
		report.Error = firstNonEmptyString(report.Metrics.ErrorMessage, "请求目标 API 失败")
	}
	if report.Status == RunStatusError {
		report.Score = 0
		report.OfficialScore = 0
		report.CompatibilityScore = 0
		report.Verdict = VerdictInvalidOrUnavailable
		report.Summary = summaryForReportError(report)
		report.Scores = scoreBreakdown(report)
	}
	syncReportCompat(report)
}

func syncReportCompat(report *PublicReport) {
	if report == nil {
		return
	}
	report.StepNameCompat = report.StepName
	report.AccessModeCompat = report.AccessMode
	report.BillingModeCompat = report.BillingMode
	report.Total = report.Score
	report.VerdictKey = cctestVerdictKey(report.Verdict)
	report.ExpectedModelCompat = report.ExpectedModel
	report.ResponseModelCompat = report.ResponseModel
	report.ResponseModelSourceCompat = report.ResponseModelSource
	report.StreamChannelCompat = report.StreamChannel
	report.NonStreamChannelCompat = report.NonStreamChannel
	report.HasVertexCompat = report.HasVertex
	report.IsKiroCompat = report.IsKiro
	report.WrapperSignalsCompat = append([]string(nil), report.WrapperSignals...)
	report.ModelIdentityCompat = cloneModelIdentity(report.ModelIdentity)
	syncMetricsCompat(&report.Metrics)
}

func syncMetricsCompat(metrics *PublicCheckMetrics) {
	if metrics == nil {
		return
	}
	metrics.LatencyMSCompat = metrics.LatencyMS
	metrics.TTFBMSCompat = metrics.StreamFirstTokenMS
	metrics.TokensPerSecondCompat = metrics.TokensPerSecond
	if metrics.Usage != nil {
		metrics.InputTokensCompat = metrics.Usage.InputTokens
		metrics.OutputTokensCompat = metrics.Usage.OutputTokens
	}
}

func cctestVerdictKey(verdict string) string {
	switch verdict {
	case VerdictOfficialOpenAI, VerdictOfficialClaude, VerdictOfficialGemini:
		return "official"
	case VerdictOpenAICompatible, VerdictClaudeCompatible, VerdictGeminiCompatible:
		return "compatible"
	case VerdictPartialCompatible:
		return "partial"
	case VerdictInvalidOrUnavailable:
		return "invalid"
	case VerdictUnknown, "":
		return "unknown"
	default:
		return verdict
	}
}

func newReportID(provider string, host string, model string, checkedAt time.Time) string {
	raw := strings.Join([]string{provider, host, model, checkedAt.UTC().Format(time.RFC3339Nano)}, "\x00")
	hash := sha256Hex(raw)
	if len(hash) > 36 {
		hash = hash[:36]
	}
	return hash
}

func summaryForVerdict(verdict string) string {
	switch verdict {
	case VerdictOfficialOpenAI:
		return "该接口表现为 OpenAI 官方协议与模型行为，Responses、工具调用和流式事件均通过。"
	case VerdictOpenAICompatible:
		return "该接口 OpenAI 兼容能力较完整，仍建议结合 Token 用量审计复核。"
	case VerdictOfficialClaude:
		return "该接口表现为 Anthropic Claude 官方协议与模型行为，Messages、工具调用、流式事件和 usage 均通过。"
	case VerdictClaudeCompatible:
		return "该接口 Claude Messages 兼容能力较完整，仍建议结合 Token 用量审计复核。"
	case VerdictOfficialGemini:
		return "该接口表现为 Google Gemini 官方协议与模型行为，模型列表、GenerateContent、工具调用、流式事件和 usage 均通过。"
	case VerdictGeminiCompatible:
		return "该接口 Gemini API 兼容能力较完整，仍建议结合响应头、模型身份和 usage metadata 复核。"
	case VerdictPartialCompatible:
		return "该接口处于兼容受限状态，生产前需复核模型、工具调用和流式行为。"
	case VerdictInvalidOrUnavailable:
		return "该接口未通过基础鉴权或协议检查。"
	default:
		return "该接口检测结果暂无法判断。"
	}
}

func summaryForReportError(report *PublicReport) string {
	if report != nil && report.Metrics.ErrorClass == errorClassAccountBalanceInsufficient {
		return "账号余额不足，无法完成纯度检测；请充值或切换有余额的账号后重试。"
	}
	return summaryForVerdict(VerdictInvalidOrUnavailable)
}

func markReportProbeError(report *PublicReport, probe httpProbe, fallback string) {
	if report == nil {
		return
	}
	report.Status = RunStatusError
	report.Step = 1
	report.StepName = "tag"
	report.Progress = roundProgress(1.0 / 7.0)
	report.Scores = scoreBreakdown(report)
	report.Error = targetAPIErrorMessage(probe, fallback)
	syncReportCompat(report)
}

func targetAPIErrorMessage(probe httpProbe, fallback string) string {
	message := strings.TrimSpace(probe.ErrorMessage)
	if message == "" {
		message = strings.TrimSpace(fallback)
	}
	if message == "" {
		message = "请求目标 API 失败"
	}
	if probe.ErrorClass == errorClassAccountBalanceInsufficient {
		message = "账号余额不足，无法完成纯度检测；请充值或切换有余额的账号后重试。"
	}
	if probe.StatusCode > 0 {
		return fmt.Sprintf("请求目标 API 失败: %d %s", probe.StatusCode, message)
	}
	return "请求目标 API 失败: " + message
}

func buildPublicSummary(report *PublicReport) map[string]any {
	if report == nil {
		return nil
	}
	out := map[string]any{
		"provider":              report.Provider,
		"report_id":             report.ReportID,
		"api_base_host":         report.APIBaseHost,
		"model_id":              report.ModelID,
		"expected_model":        report.ExpectedModel,
		"response_model":        report.ResponseModel,
		"response_model_source": report.ResponseModelSource,
		"status":                report.Status,
		"step":                  report.Step,
		"step_name":             report.StepName,
		"progress":              report.Progress,
		"scores":                report.Scores,
		"score":                 report.Score,
		"official_score":        report.OfficialScore,
		"compatibility_score":   report.CompatibilityScore,
		"verdict":               report.Verdict,
		"summary":               report.Summary,
		"error":                 report.Error,
		"stream_channel":        report.StreamChannel,
		"non_stream_channel":    report.NonStreamChannel,
		"has_vertex":            report.HasVertex,
		"is_kiro":               report.IsKiro,
		"wrapper_signals":       report.WrapperSignals,
		"model_identity":        report.ModelIdentity,
		"validations":           report.Validations,
		"checked_at":            report.CheckedAt,
	}
	if report.TokenAudit != nil {
		out["token_audit"] = map[string]any{
			"status":                report.TokenAudit.Status,
			"summary":               report.TokenAudit.Summary,
			"sample_count":          report.TokenAudit.SampleCount,
			"official_baseline_usd": report.TokenAudit.OfficialBaselineUSD,
			"uncached_baseline_usd": report.TokenAudit.UncachedBaselineUSD,
			"actual_cost_usd":       report.TokenAudit.ActualCostUSD,
			"total_cost":            report.TokenAudit.TotalCostUSD,
			"multiplier":            report.TokenAudit.Multiplier,
			"overall_ratio":         report.TokenAudit.OverallRatio,
			"cache_hit_rate":        report.TokenAudit.CacheHitRate,
			"prompt_cache_key":      report.TokenAudit.PromptCacheKey,
			"store_enabled":         report.TokenAudit.StoreEnabled,
			"stateful_rounds":       report.TokenAudit.StatefulRounds,
			"previous_chain_ok":     report.TokenAudit.PreviousChainOK,
			"anomalies":             report.TokenAudit.Anomalies,
		}
	}
	return out
}

func cloneModelIdentity(identity *ModelIdentityResult) *ModelIdentityResult {
	if identity == nil {
		return nil
	}
	out := *identity
	if identity.ModelListContainsRequested != nil {
		value := *identity.ModelListContainsRequested
		out.ModelListContainsRequested = &value
	}
	if identity.Evidence != nil {
		out.Evidence = make(map[string]any, len(identity.Evidence))
		for key, value := range identity.Evidence {
			out.Evidence[key] = value
		}
	}
	return &out
}
