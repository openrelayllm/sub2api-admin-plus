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
	report.ScoreAdjustments = nil
	score = applyModelIdentityDimensionDeduction(report, breakdown, score)
	score = applyProvenanceScoreAdjustment(report, score)
	return score
}

func applyModelIdentityDimensionDeduction(report *PublicReport, breakdown map[string]int, score int) int {
	if !modelIdentityFailed(report) || breakdown["tag_check"] <= 0 {
		return score
	}
	adjustedBreakdown := make(map[string]int, len(breakdown))
	for key, value := range breakdown {
		adjustedBreakdown[key] = value
	}
	adjustedBreakdown["tag_check"] = 0
	result := officialScoreFromBreakdown(report, adjustedBreakdown, 0)
	if report.Scores != nil {
		report.Scores["tag_check"] = 0
	}
	report.ScoreAdjustments = append(report.ScoreAdjustments, ScoreAdjustment{
		ID:           "model_identity_dimension_deduction",
		Category:     "client_capability",
		ReasonCode:   "model_identity_conflict",
		CaseID:       "PURITY-CLIENT-IDENTITY-001",
		ClientImpact: clientImpactBreaking,
		ImpactScope:  "tag_check",
		BaseScore:    score,
		Points:       result - score,
		ResultScore:  result,
		Evidence:     []string{firstNonEmptyString(report.ModelIdentity.Reason, "model_identity_mismatch")},
	})
	return result
}

func applyProvenanceScoreAdjustment(report *PublicReport, score int) int {
	if !hasWrapperObfuscationFingerprint(report) || hasClientImpactFailure(report) {
		return score
	}
	const penalty = 5
	result := maxInt(0, score-penalty)
	reasonCode := "wrapper_obfuscation_signal"
	caseID := "PURITY-PROVENANCE-001"
	adjustmentID := "provenance_transparency_penalty"
	impactScope := "provenance_transparency_only"
	evidence := wrapperObfuscationSignals(report)
	if attributionHasReasonCode(report, "bedrock_anthropic_signature_mask") {
		adjustmentID = "bedrock_anthropic_signature_mask_penalty"
		reasonCode = "bedrock_anthropic_signature_mask"
		caseID = "PURITY-BEDROCK-MASK-001"
		impactScope = "channel_attribution_only"
		evidence = []string{
			"bedrock_metadata_family_present",
			"anthropic_native_metadata_present",
		}
	}
	report.ScoreAdjustments = append(report.ScoreAdjustments, ScoreAdjustment{
		ID:           adjustmentID,
		Category:     "provenance_transparency",
		ReasonCode:   reasonCode,
		CaseID:       caseID,
		ClientImpact: clientImpactNone,
		ImpactScope:  impactScope,
		BaseScore:    score,
		Points:       -penalty,
		ResultScore:  result,
		Evidence:     evidence,
	})
	return result
}

func hasClientImpactFailure(report *PublicReport) bool {
	if report == nil {
		return false
	}
	if modelIdentityFailed(report) {
		return true
	}
	for _, dimension := range report.ScorePolicy.Dimensions {
		for _, validation := range report.Validations {
			if validation.ID != dimension.ValidationID || validation.Status != CheckStatusFail {
				continue
			}
			if skipped, _ := validation.Details["skipped"].(bool); skipped {
				continue
			}
			return true
		}
	}
	return false
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
	if attributionHasReasonCode(report, "bedrock_anthropic_signature_mask") {
		signals = appendUniqueString(signals, "bedrock_anthropic_signature_mask")
	}
	return signals
}

func attributionHasReasonCode(report *PublicReport, expected string) bool {
	if report == nil || report.ChannelAttribution == nil {
		return false
	}
	for _, code := range report.ChannelAttribution.ReasonCodes {
		if code == expected {
			return true
		}
	}
	return false
}

func modelIdentityFailed(report *PublicReport) bool {
	return report != nil && report.ModelIdentity != nil && report.ModelIdentity.Status == CheckStatusFail
}

func summaryForReport(report *PublicReport) string {
	if report == nil {
		return summaryForVerdict(VerdictUnknown)
	}
	if report.Assessment != nil && strings.TrimSpace(report.Assessment.Summary) != "" {
		return report.Assessment.Summary
	}
	identitySummary := modelIdentitySummary(report)
	channelSummary := attributionSummary(report)
	contextSummary := strings.TrimSpace(strings.Join(nonEmptyStrings(channelSummary, identitySummary), " "))
	if hasWrapperFingerprint(report) {
		signals := strings.Join(report.WrapperSignals, "、")
		if strings.TrimSpace(signals) == "" && report.IsKiro {
			signals = "kiro"
		}
		obfuscationSignals := wrapperObfuscationSignals(report)
		suffix := contextSummary
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
	if contextSummary != "" {
		return contextSummary
	}
	return summaryForVerdict(report.Verdict)
}

func nonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
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
