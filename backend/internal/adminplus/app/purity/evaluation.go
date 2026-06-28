package purity

import (
	"fmt"
	"strings"
)

func decideVerdict(report *PublicReport) string {
	if report == nil {
		return VerdictUnknown
	}
	responsesOK := false
	messagesOK := false
	toolOK := false
	streamOK := false
	chatOK := false
	for _, check := range report.Checks {
		switch check.ID {
		case "responses_schema":
			responsesOK = check.Status == CheckStatusPass
		case "claude_messages_schema":
			messagesOK = check.Status == CheckStatusPass
		case "tool_call":
			toolOK = check.Status == CheckStatusPass
		case "claude_tool_use":
			toolOK = check.Status == CheckStatusPass
		case "streaming":
			streamOK = check.Status == CheckStatusPass
		case "claude_streaming":
			streamOK = check.Status == CheckStatusPass
		case "chat_completions":
			chatOK = check.Status == CheckStatusPass
		}
	}
	if report.Provider == ProviderGemini {
		if hasWrapperObfuscationFingerprint(report) {
			if report.CompatibilityScore >= 80 && responsesOK && streamOK {
				return VerdictGeminiCompatible
			}
			if report.CompatibilityScore >= 50 {
				return VerdictPartialCompatible
			}
			return VerdictInvalidOrUnavailable
		}
		if !modelIdentityFailed(report) && responsesOK && toolOK && streamOK && report.Score >= 85 {
			return VerdictOfficialGemini
		}
		if report.CompatibilityScore >= 80 && responsesOK && streamOK {
			return VerdictGeminiCompatible
		}
		if report.CompatibilityScore >= 50 {
			return VerdictPartialCompatible
		}
		return VerdictInvalidOrUnavailable
	}
	if report.Provider == ProviderAnthropic {
		if hasWrapperObfuscationFingerprint(report) {
			if report.CompatibilityScore >= 80 && messagesOK && streamOK {
				return VerdictClaudeCompatible
			}
			if report.CompatibilityScore >= 50 {
				return VerdictPartialCompatible
			}
			return VerdictInvalidOrUnavailable
		}
		if !modelIdentityFailed(report) && messagesOK && toolOK && streamOK && report.Score >= 85 {
			return VerdictOfficialClaude
		}
		if report.CompatibilityScore >= 80 && messagesOK && streamOK {
			return VerdictClaudeCompatible
		}
		if report.CompatibilityScore >= 50 {
			return VerdictPartialCompatible
		}
		return VerdictInvalidOrUnavailable
	}
	if !modelIdentityFailed(report) && !hasWrapperObfuscationFingerprint(report) && responsesOK && toolOK && streamOK && report.Score >= 85 {
		return VerdictOfficialOpenAI
	}
	if report.CompatibilityScore >= 80 && responsesOK && streamOK {
		return VerdictOpenAICompatible
	}
	if report.CompatibilityScore >= 50 || chatOK {
		return VerdictPartialCompatible
	}
	return VerdictInvalidOrUnavailable
}

func officialPurityScore(report *PublicReport, fallback int) int {
	if report == nil {
		return fallback
	}
	breakdown := scoreBreakdown(report)
	if len(breakdown) == 0 {
		return fallback
	}
	score := officialScoreFromBreakdown(report, breakdown, fallback)
	if score == 0 && fallback > 0 {
		score = fallback
	}
	if score > 100 {
		score = 100
	}
	if hasWrapperObfuscationFingerprint(report) {
		score = minInt(score, wrapperPurityScoreCap(report))
	}
	if modelIdentityFailed(report) {
		score = minInt(score, modelIdentityScoreCap(report))
	}
	return score
}

func wrapperPurityScoreCap(report *PublicReport) int {
	if report == nil {
		return 35
	}
	capValue := 100
	if hasWrapperObfuscationFingerprint(report) {
		capValue = 55
	}
	if report.Provider == ProviderOpenAI && validationFailedAfterProbe(report, "signature") {
		capValue = 75
	}
	if report.Provider == ProviderAnthropic && (validationFailedAfterProbe(report, "signature") || hasTokenAuditAnomaly(report, "claude_cache_accounting_missing", "cost_multiplier_anomaly")) {
		capValue = 45
	}
	if validationFailedAfterProbe(report, "behavior") && validationFailedAfterProbe(report, "signature") {
		capValue = 35
	}
	return capValue
}

func validationFailedAfterProbe(report *PublicReport, id string) bool {
	if report == nil {
		return false
	}
	for _, validation := range report.Validations {
		if validation.ID != id {
			continue
		}
		if validation.Status != CheckStatusFail {
			return false
		}
		if skipped, _ := validation.Details["skipped"].(bool); skipped {
			return false
		}
		return true
	}
	return false
}

func hasTokenAuditAnomaly(report *PublicReport, anomalies ...string) bool {
	if report == nil || report.TokenAudit == nil || len(report.TokenAudit.Anomalies) == 0 {
		return false
	}
	for _, actual := range report.TokenAudit.Anomalies {
		for _, expected := range anomalies {
			if actual == expected {
				return true
			}
		}
	}
	return false
}

func hasWrapperFingerprint(report *PublicReport) bool {
	if report == nil {
		return false
	}
	return len(report.WrapperSignals) > 0 || report.IsKiro
}

func hasWrapperObfuscationFingerprint(report *PublicReport) bool {
	return len(wrapperObfuscationSignals(report)) > 0
}

func wrapperObfuscationSignals(report *PublicReport) []string {
	if report == nil {
		return nil
	}
	signals := make([]string, 0, 4)
	for _, signal := range report.WrapperSignals {
		switch signal {
		case "cliproxyapi-codex-identity", "cliproxyapi-model-mapping", "cliproxyapi-signature-bridge", "new-api-model-mapping", "sub2api-model-mapping", "sub2api-protocol-bridge":
			signals = appendUniqueString(signals, signal)
		}
	}
	if modelIdentityFailed(report) {
		signals = appendUniqueString(signals, "model_identity")
	}
	if report.Provider == ProviderAnthropic && validationFailedAfterProbe(report, "signature") {
		signals = appendUniqueString(signals, "claude_signature")
	}
	if report.Provider == ProviderOpenAI && validationFailedAfterProbe(report, "signature") {
		signals = appendUniqueString(signals, "openai_signature")
	}
	if hasTokenAuditAnomaly(report, "claude_cache_accounting_missing", "cost_multiplier_anomaly") {
		signals = appendUniqueString(signals, "token_audit")
	}
	return signals
}

func modelIdentityFailed(report *PublicReport) bool {
	return report != nil && report.ModelIdentity != nil && report.ModelIdentity.Status == CheckStatusFail
}

func modelIdentityScoreCap(report *PublicReport) int {
	if report == nil || report.ModelIdentity == nil {
		return 50
	}
	switch report.ModelIdentity.Reason {
	case modelIdentityReasonCrossVendorAlias, modelIdentityReasonProtocolVendorMismatch, modelIdentityReasonWrapperVendorSignalMismatch:
		return 40
	default:
		return 50
	}
}

func summaryForReport(report *PublicReport) string {
	if report == nil {
		return summaryForVerdict(VerdictUnknown)
	}
	identitySummary := modelIdentitySummary(report)
	if hasWrapperFingerprint(report) {
		signals := strings.Join(report.WrapperSignals, "、")
		if strings.TrimSpace(signals) == "" && report.IsKiro {
			signals = "kiro"
		}
		obfuscationSignals := wrapperObfuscationSignals(report)
		suffix := ""
		if identitySummary != "" {
			suffix = identitySummary
		}
		if len(obfuscationSignals) > 0 {
			riskSignals := strings.Join(obfuscationSignals, "、")
			if report.Provider == ProviderAnthropic {
				return fmt.Sprintf("当前为兼容受限状态。检测到包装/中转信号：%s；并存在模型或协议混淆风险：%s。协议表面兼容 Claude，但不是原生 Anthropic Claude API。%s", signals, riskSignals, suffix)
			}
			if report.Provider == ProviderGemini {
				return fmt.Sprintf("当前为兼容受限状态。检测到包装/中转信号：%s；并存在模型或协议混淆风险：%s。协议表面兼容 Gemini，但不是原生 Google Gemini API。%s", signals, riskSignals, suffix)
			}
			return fmt.Sprintf("当前为兼容受限状态。检测到包装/中转信号：%s；并存在模型或协议混淆风险：%s。协议表面兼容 OpenAI，但不是原生 OpenAI 官方 API。%s", signals, riskSignals, suffix)
		}
		if report.Provider == ProviderAnthropic {
			return fmt.Sprintf("检测到透明中转/兼容网关信号：%s；当前证据未显示模型或协议混淆，可继续按 Claude 上游纯度评估。%s", signals, suffix)
		}
		if report.Provider == ProviderGemini {
			return fmt.Sprintf("检测到透明中转/兼容网关信号：%s；当前证据未显示模型或协议混淆，可继续按 Gemini 上游纯度评估。%s", signals, suffix)
		}
		return fmt.Sprintf("检测到透明中转/兼容网关信号：%s；当前证据未显示模型或协议混淆，可继续按 OpenAI 上游纯度评估。%s", signals, suffix)
	}
	if identitySummary != "" {
		return strings.TrimSpace(identitySummary)
	}
	return summaryForVerdict(report.Verdict)
}

func modelIdentitySummary(report *PublicReport) string {
	if !modelIdentityFailed(report) {
		return ""
	}
	identity := report.ModelIdentity
	switch identity.Reason {
	case modelIdentityReasonVersionDowngrade, modelIdentityReasonTierDowngrade:
		return fmt.Sprintf("模型身份异常：请求 %s，响应 %s，存在降级或低配替代风险。", identity.RequestedModel, identity.ResponseModel)
	case modelIdentityReasonCrossVendorAlias:
		return fmt.Sprintf("模型身份异常：请求 %s，响应 %s，存在跨厂商伪装风险。", identity.RequestedModel, identity.ResponseModel)
	case modelIdentityReasonProtocolVendorMismatch:
		return fmt.Sprintf("模型身份异常：请求 %s 与当前协议厂商不一致，存在兼容通道或模型别名映射风险。", identity.RequestedModel)
	case modelIdentityReasonWrapperVendorSignalMismatch:
		return fmt.Sprintf("模型身份异常：请求 %s 与包装层暴露的上游厂商信号不一致。", identity.RequestedModel)
	case modelIdentityReasonReasoningTokensMismatch:
		return fmt.Sprintf("模型身份异常：请求 %s 声称为非 reasoning 模型，但响应 usage 暴露了 reasoning_tokens。", identity.RequestedModel)
	default:
		return "模型身份异常：请求模型与响应模型不一致。"
	}
}
