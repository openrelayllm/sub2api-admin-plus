package purity

import (
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/attribution"
)

const (
	assessmentKindOfficialNative       = "official_native"
	assessmentKindOfficialCloud        = "official_cloud_channel"
	assessmentKindTransparentRelay     = "transparent_relay"
	assessmentKindCompatible           = "compatible_channel"
	assessmentKindChannelConflicted    = "channel_conflicted"
	assessmentKindIdentityConflict     = "identity_conflict"
	assessmentKindCompatibilityRisk    = "compatibility_risk"
	assessmentKindInvalidOrUnavailable = "invalid_or_unavailable"

	assessmentStatusReady   = "ready"
	assessmentStatusLimited = "limited"
	assessmentStatusRisky   = "risky"
	assessmentStatusInvalid = "invalid"
)

func buildAssessment(report *PublicReport) *AssessmentResult {
	if report == nil {
		return nil
	}
	result := &AssessmentResult{
		Kind:           assessmentKindCompatible,
		Status:         assessmentStatusReady,
		Channel:        "unknown",
		ChannelStatus:  attribution.StatusUnknown,
		IdentityStatus: "unknown",
		ProtocolStatus: protocolAssessmentStatus(report.ProtocolScore),
		WrapperMode:    wrapperAssessmentMode(report),
		MeteringStatus: report.MeteringStatus,
		DimensionTotal: len(report.DimensionMatrix),
		Limitations:    []string{},
		ReasonCodes:    []string{},
	}
	if report.ChannelAttribution != nil {
		result.Channel = firstNonEmptyString(report.ChannelAttribution.Channel, "unknown")
		result.ChannelStatus = firstNonEmptyString(report.ChannelAttribution.Status, attribution.StatusUnknown)
		result.ChannelConfidence = report.ChannelAttribution.Confidence
		result.ReasonCodes = append(result.ReasonCodes, report.ChannelAttribution.ReasonCodes...)
	}
	if report.ModelIdentity != nil {
		result.IdentityStatus = firstNonEmptyString(report.ModelIdentity.Status, "unknown")
	}
	for _, dimension := range report.DimensionMatrix {
		if dimension.Status != dimensionStatusNotRun && dimension.Status != dimensionStatusNotApplicable {
			result.DimensionExecuted++
		}
		if dimension.Scored {
			result.DimensionScored++
		}
		switch dimension.Status {
		case dimensionStatusUnsupportedByUpstream:
			result.Limitations = appendUniqueString(result.Limitations, dimension.Name+"：上游不支持")
		case dimensionStatusFail:
			result.Limitations = appendUniqueString(result.Limitations, dimension.Name+"：检测失败")
		case dimensionStatusWarn:
			if dimension.ID != "fingerprint" {
				result.Limitations = appendUniqueString(result.Limitations, dimension.Name+"：结果受限")
			}
		}
	}
	if result.DimensionExecuted < result.DimensionTotal {
		result.ReasonCodes = appendUniqueString(result.ReasonCodes, "partial_probe_coverage")
	}
	if !report.CheckTokenUsage {
		result.MeteringStatus = "not_tested"
		result.ReasonCodes = appendUniqueString(result.ReasonCodes, "token_audit_not_requested")
	}

	applyAssessmentClassification(report, result)
	result.Title = assessmentTitle(report, result)
	result.Summary = assessmentSummary(report, result)
	return result
}

func applyAssessmentClassification(report *PublicReport, result *AssessmentResult) {
	if report == nil || result == nil {
		return
	}
	switch {
	case report.Status == RunStatusError || report.ProtocolScore < 50:
		result.Kind = assessmentKindInvalidOrUnavailable
		result.Status = assessmentStatusInvalid
		result.ReasonCodes = appendUniqueString(result.ReasonCodes, "protocol_unavailable")
	case modelIdentityFailed(report):
		result.Kind = assessmentKindIdentityConflict
		result.Status = assessmentStatusRisky
		result.ReasonCodes = appendUniqueString(result.ReasonCodes, "model_identity_conflict")
	case result.ChannelStatus == attribution.StatusConflicted:
		result.Kind = assessmentKindChannelConflicted
		result.Status = assessmentStatusLimited
		result.ReasonCodes = appendUniqueString(result.ReasonCodes, "channel_evidence_conflict")
	case hasWrapperObfuscationFingerprint(report):
		result.Kind = assessmentKindCompatibilityRisk
		result.Status = assessmentStatusRisky
		result.ReasonCodes = appendUniqueString(result.ReasonCodes, "wrapper_obfuscation_signal")
	case result.WrapperMode == "transparent":
		result.Kind = assessmentKindTransparentRelay
	case attribution.ChannelKind(result.Channel) == attribution.ChannelKindOfficialCloud:
		result.Kind = assessmentKindOfficialCloud
	case attribution.ChannelKind(result.Channel) == attribution.ChannelKindOfficialNative:
		result.Kind = assessmentKindOfficialNative
	default:
		result.Kind = assessmentKindCompatible
	}
	if result.Status == assessmentStatusReady && (len(result.Limitations) > 0 || report.ProtocolScore < 90) {
		result.Status = assessmentStatusLimited
	}
}

func assessmentTitle(report *PublicReport, result *AssessmentResult) string {
	channel := channelDisplayName(result.Channel)
	model := firstNonEmptyString(report.ResponseModel, report.ExpectedModel, report.ModelID, providerDisplayName(report.Provider))
	switch result.Kind {
	case assessmentKindOfficialNative:
		return fmt.Sprintf("%s 原生渠道 · %s", channel, model)
	case assessmentKindOfficialCloud:
		return fmt.Sprintf("%s 官方云渠道 · %s", channel, model)
	case assessmentKindTransparentRelay:
		return fmt.Sprintf("透明中转 · 上游 %s · %s", channel, model)
	case assessmentKindChannelConflicted:
		return fmt.Sprintf("渠道证据冲突 · %s", model)
	case assessmentKindIdentityConflict:
		return fmt.Sprintf("模型身份冲突 · 请求 %s", firstNonEmptyString(report.ExpectedModel, report.ModelID))
	case assessmentKindCompatibilityRisk:
		return fmt.Sprintf("兼容链路存在混淆风险 · %s", model)
	case assessmentKindInvalidOrUnavailable:
		return fmt.Sprintf("%s 接口不可用或协议不完整", providerDisplayName(report.Provider))
	default:
		return fmt.Sprintf("%s 兼容渠道 · %s", providerDisplayName(report.Provider), model)
	}
}

func assessmentSummary(report *PublicReport, result *AssessmentResult) string {
	wrapperSummary := wrapperModeDisplayName(result.WrapperMode)
	if len(report.WrapperSignals) > 0 {
		wrapperSummary += "（信号：" + strings.Join(report.WrapperSignals, "、") + "）"
	}
	parts := []string{
		fmt.Sprintf("整体状态：%s", assessmentStatusDisplayName(result.Status)),
		fmt.Sprintf("模型身份：%s", identityAssessmentText(report)),
		fmt.Sprintf("上游渠道：%s", channelAssessmentText(result)),
		fmt.Sprintf("%s协议兼容：%d/100（%s）", providerProtocolName(report.Provider), report.ProtocolScore, protocolStatusDisplayName(result.ProtocolStatus)),
		fmt.Sprintf("网关包装：%s", wrapperSummary),
	}
	if len(result.Limitations) > 0 {
		parts = append(parts, "限制："+strings.Join(result.Limitations, "；"))
	} else {
		parts = append(parts, "关键能力：本轮已执行项未发现明确限制")
	}
	if report.CheckTokenUsage {
		parts = append(parts, fmt.Sprintf("Token 用量审计：%s", meteringStatusDisplayName(result.MeteringStatus)))
	}
	parts = append(parts, fmt.Sprintf("维度覆盖：已执行 %d/%d，已评分 %d 项", result.DimensionExecuted, result.DimensionTotal, result.DimensionScored))
	return strings.Join(parts, "。") + "。"
}

func assessmentStatusDisplayName(status string) string {
	switch status {
	case assessmentStatusReady:
		return "可用"
	case assessmentStatusLimited:
		return "兼容受限"
	case assessmentStatusRisky:
		return "兼容受限，存在高风险信号"
	default:
		return "不可用"
	}
}

func protocolAssessmentStatus(score int) string {
	switch {
	case score >= 90:
		return "high"
	case score >= 70:
		return "medium"
	case score >= 50:
		return "low"
	default:
		return "unavailable"
	}
}

func wrapperAssessmentMode(report *PublicReport) string {
	if hasWrapperObfuscationFingerprint(report) {
		return "obfuscating"
	}
	if hasWrapperFingerprint(report) {
		return "transparent"
	}
	return "none"
}

func identityAssessmentText(report *PublicReport) string {
	if report == nil || report.ModelIdentity == nil {
		return "未确认"
	}
	identity := report.ModelIdentity
	switch identity.Status {
	case CheckStatusPass:
		return fmt.Sprintf("一致（%s）", firstNonEmptyString(identity.ResponseModel, report.ResponseModel, report.ModelID))
	case CheckStatusWarn:
		return fmt.Sprintf("需复核（%s）", firstNonEmptyString(identity.Reason, "证据不足"))
	default:
		return fmt.Sprintf("冲突（请求 %s，响应 %s）", firstNonEmptyString(identity.RequestedModel, report.ExpectedModel), firstNonEmptyString(identity.ResponseModel, report.ResponseModel, "未知"))
	}
}

func channelAssessmentText(result *AssessmentResult) string {
	if result == nil || result.ChannelStatus == attribution.StatusUnknown {
		return "未知（证据不足，不据此判模型失败）"
	}
	if result.ChannelStatus == attribution.StatusConflicted {
		return "证据冲突（未强行选择来源）"
	}
	return fmt.Sprintf("%s（%s，%.0f%%）", channelDisplayName(result.Channel), attributionStatusDisplayName(result.ChannelStatus), result.ChannelConfidence*100)
}

func channelDisplayName(channel string) string {
	switch channel {
	case "anthropic_native":
		return "Anthropic API"
	case "aws_bedrock":
		return "AWS Bedrock"
	case "google_vertex":
		return "Google Vertex AI"
	case "google_ai_studio":
		return "Google AI Studio"
	case "openai_native":
		return "OpenAI API"
	case "azure_openai":
		return "Azure OpenAI"
	case "alibaba_bailian":
		return "Alibaba Cloud Model Studio"
	case "baidu_wenxin":
		return "Baidu Wenxin"
	case "baidu_qianfan":
		return "Baidu Qianfan"
	case "ai360":
		return "360 AI"
	case "zhipu_bigmodel":
		return "Zhipu BigModel"
	case "tencent_hunyuan":
		return "Tencent Hunyuan"
	case "moonshot":
		return "Moonshot AI"
	case "perplexity":
		return "Perplexity AI"
	case "yi":
		return "01.AI"
	case "cohere":
		return "Cohere"
	case "minimax":
		return "MiniMax"
	case "siliconflow":
		return "SiliconFlow"
	case "mistral":
		return "Mistral AI"
	case "deepseek":
		return "DeepSeek"
	case "volcengine_ark":
		return "Volcengine Ark"
	case "xai":
		return "xAI"
	case "zai_coding":
		return "Z.AI Coding"
	case "kimi_coding":
		return "Kimi Coding"
	case "openai_codex_subscription":
		return "OpenAI Codex Subscription"
	case "openrouter":
		return "OpenRouter"
	case "cloudflare_workers_ai":
		return "Cloudflare Workers AI"
	case "dify":
		return "Dify"
	case "coze":
		return "Coze"
	case "fastgpt":
		return "FastGPT"
	case "submodel":
		return "Submodel"
	case "openai_sb":
		return "OpenAI-SB"
	case "openaimax":
		return "OpenAIMax"
	case "ohmygpt":
		return "OhMyGPT"
	case "caipacity":
		return "CaiPac"
	case "aiproxy":
		return "AIProxy"
	case "api2gpt":
		return "API2GPT"
	case "aigc2d":
		return "AIGC2D"
	case "kiro":
		return "Kiro"
	case "antigravity":
		return "Antigravity"
	default:
		return "未知兼容渠道"
	}
}

func providerDisplayName(provider string) string {
	switch provider {
	case ProviderAnthropic:
		return "Claude"
	case ProviderGemini:
		return "Gemini"
	default:
		return "OpenAI"
	}
}

func providerProtocolName(provider string) string {
	switch provider {
	case ProviderAnthropic:
		return "Anthropic Messages "
	case ProviderGemini:
		return "Gemini GenerateContent "
	default:
		return "OpenAI Responses "
	}
}

func attributionStatusDisplayName(status string) string {
	switch status {
	case attribution.StatusIdentified:
		return "已识别"
	case attribution.StatusLikely:
		return "较可能"
	case attribution.StatusConflicted:
		return "证据冲突"
	default:
		return "未知"
	}
}

func protocolStatusDisplayName(status string) string {
	switch status {
	case "high":
		return "高"
	case "medium":
		return "中"
	case "low":
		return "低"
	default:
		return "不可用"
	}
}

func wrapperModeDisplayName(mode string) string {
	switch mode {
	case "transparent":
		return "检测到透明中转/兼容网关，未自动扣减协议分"
	case "obfuscating":
		return "检测到可能混淆模型或协议的包装/中转信号"
	default:
		return "未发现明显包装信号"
	}
}

func meteringStatusDisplayName(status string) string {
	switch status {
	case capabilityStatusSupported:
		return "通过"
	case capabilityStatusLimited:
		return "需复核"
	case capabilityStatusUnsupported:
		return "异常"
	case "not_tested":
		return "未检测"
	default:
		return "未知"
	}
}

func cloneAssessment(value *AssessmentResult) *AssessmentResult {
	if value == nil {
		return nil
	}
	out := *value
	out.Limitations = append([]string(nil), value.Limitations...)
	out.ReasonCodes = append([]string(nil), value.ReasonCodes...)
	return &out
}
