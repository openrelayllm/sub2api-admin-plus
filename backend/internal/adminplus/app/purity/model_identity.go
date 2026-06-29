package purity

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	modelIdentityReasonExactMatch                  = "exact_match"
	modelIdentityReasonCompatibleAlias             = "compatible_alias"
	modelIdentityReasonResponseModelMissing        = "response_model_missing"
	modelIdentityReasonProbeFallback               = "probe_model_fallback"
	modelIdentityReasonCrossVendorAlias            = "cross_vendor_alias"
	modelIdentityReasonFamilyMismatch              = "family_mismatch"
	modelIdentityReasonVersionDowngrade            = "version_downgrade"
	modelIdentityReasonTierDowngrade               = "tier_downgrade"
	modelIdentityReasonProtocolVendorMismatch      = "protocol_model_vendor_mismatch"
	modelIdentityReasonWrapperVendorSignalMismatch = "wrapper_vendor_signal_mismatch"
	modelIdentityReasonReasoningTokensMismatch     = "reasoning_tokens_mismatch"
)

type modelProfile struct {
	Raw       string
	Canonical string
	Vendor    string
	Family    string
	Version   []int
	Tier      string
	TierRank  int
}

func appendAndEmitModelIdentity(report *PublicReport, emit PublicCheckEventSink) {
	check := buildModelIdentityCheck(report)
	appendAndEmitChecks(report, emit, check)
	upsertAndEmitValidation(report, emit, validationFromExecutedChecks("model_identity", "模型身份验证", []CheckResult{check}))
}

func buildModelIdentityCheck(report *PublicReport) CheckResult {
	result := evaluateModelIdentity(report)
	if report != nil {
		report.ModelIdentity = &result
	}
	details := modelIdentityDetails(result)
	switch result.Status {
	case CheckStatusPass:
		return CheckResult{ID: "model_identity", Name: "模型身份一致性", Status: CheckStatusPass, Score: 0, MaxScore: 0, Message: modelIdentityPassMessage(result), Details: details}
	case CheckStatusFail:
		return CheckResult{ID: "model_identity", Name: "模型身份一致性", Status: CheckStatusFail, Score: 0, MaxScore: 0, Message: modelIdentityFailureMessage(result), Details: details}
	default:
		return CheckResult{ID: "model_identity", Name: "模型身份一致性", Status: CheckStatusWarn, Score: 0, MaxScore: 0, Message: modelIdentityWarningMessage(result), Details: details}
	}
}

func evaluateModelIdentity(report *PublicReport) ModelIdentityResult {
	requested := parseModelProfile(firstNonEmptyString(modelString(report, "expected"), modelString(report, "model")))
	response := parseModelProfile(modelString(report, "response"))
	modelListed, modelListedKnown := modelListContainsRequested(report)
	evidence := map[string]any{}
	if report != nil {
		evidence["wrapper_signals"] = append([]string(nil), report.WrapperSignals...)
		if report.ResponseModelSource != "" {
			evidence["response_model_source"] = report.ResponseModelSource
		}
		if report.responseBodyModel != "" {
			evidence["response_body_model"] = report.responseBodyModel
		}
		if report.responseHeaderModel != "" {
			evidence["openai_model_header"] = report.responseHeaderModel
		}
		if report.Metrics.Usage != nil && report.Metrics.Usage.ReasoningTokens > 0 {
			evidence["reasoning_tokens"] = report.Metrics.Usage.ReasoningTokens
		}
	}
	if modelListedKnown {
		evidence["model_list_contains_requested"] = modelListed
	}
	result := ModelIdentityResult{
		Status:          CheckStatusPass,
		Reason:          modelIdentityReasonExactMatch,
		RequestedModel:  requested.Raw,
		ResponseModel:   response.Raw,
		RequestedVendor: requested.Vendor,
		ResponseVendor:  response.Vendor,
		RequestedFamily: requested.Family,
		ResponseFamily:  response.Family,
		Evidence:        evidence,
	}
	if modelListedKnown {
		result.ModelListContainsRequested = boolPtr(modelListed)
	}
	if fallbackUsed, requestedFallback, probeFallback, fallbackCause := geminiProbeFallbackInfo(report); fallbackUsed {
		if requestedFallback != "" {
			result.Evidence["requested_model"] = requestedFallback
		}
		if probeFallback != "" {
			result.Evidence["probe_model"] = probeFallback
		}
		if fallbackCause != "" {
			result.Evidence["model_fallback_cause"] = fallbackCause
		}
		result.Evidence["model_fallback"] = true
		if response.Vendor == "google" && response.Family == "gemini" {
			result.Status = CheckStatusPass
			result.Reason = modelIdentityReasonProbeFallback
			return result
		}
	}
	if requested.Canonical == "" {
		result.Status = CheckStatusWarn
		result.Reason = modelIdentityReasonResponseModelMissing
		return result
	}
	if expectedVendor := protocolExpectedVendor(reportProvider(report)); expectedVendor != "" && requested.Vendor != "" && requested.Vendor != expectedVendor {
		result.Status = CheckStatusFail
		result.Reason = modelIdentityReasonProtocolVendorMismatch
		result.Evidence["protocol_expected_vendor"] = expectedVendor
		return result
	}
	if suspectedVendor := wrapperVendorMismatch(requested.Vendor, reportWrapperSignals(report)); suspectedVendor != "" {
		result.Status = CheckStatusFail
		result.Reason = modelIdentityReasonWrapperVendorSignalMismatch
		result.Evidence["suspected_upstream_vendor"] = suspectedVendor
		return result
	}
	if response.Canonical == "" {
		result.Status = CheckStatusWarn
		result.Reason = modelIdentityReasonResponseModelMissing
		return result
	}
	if requested.Vendor != "" && response.Vendor != "" && requested.Vendor != response.Vendor {
		result.Status = CheckStatusFail
		result.Reason = modelIdentityReasonCrossVendorAlias
		return result
	}
	if requested.Family != "" && response.Family != "" && requested.Family != response.Family {
		result.Status = CheckStatusFail
		result.Reason = modelIdentityReasonFamilyMismatch
		return result
	}
	if compareVersion(response.Version, requested.Version) < 0 {
		result.Status = CheckStatusFail
		result.Reason = modelIdentityReasonVersionDowngrade
		result.VersionDelta = fmt.Sprintf("%s<%s", versionString(response.Version), versionString(requested.Version))
		return result
	}
	if requested.TierRank > 0 && response.TierRank > 0 && response.TierRank < requested.TierRank {
		result.Status = CheckStatusFail
		result.Reason = modelIdentityReasonTierDowngrade
		result.TierDelta = fmt.Sprintf("%s<%s", response.Tier, requested.Tier)
		return result
	}
	if hasUnexpectedOpenAIReasoningTokens(report, requested) {
		result.Status = CheckStatusFail
		result.Reason = modelIdentityReasonReasoningTokensMismatch
		result.Evidence["expected_reasoning_tokens"] = false
		return result
	}
	if requested.Canonical != response.Canonical {
		result.Status = CheckStatusWarn
		result.Reason = modelIdentityReasonCompatibleAlias
		return result
	}
	return result
}

func parseModelProfile(model string) modelProfile {
	raw := strings.TrimSpace(model)
	canonical := normalizeModelName(raw)
	profile := modelProfile{
		Raw:       raw,
		Canonical: canonical,
		Version:   numericParts(canonical),
	}
	if canonical == "" {
		return profile
	}
	profile.Vendor, profile.Family = inferModelVendorAndFamily(canonical)
	profile.Tier, profile.TierRank = inferModelTier(canonical)
	return profile
}

func normalizeModelName(model string) string {
	value := strings.ToLower(strings.TrimSpace(model))
	value = strings.TrimPrefix(value, "models/")
	value = strings.TrimPrefix(value, "model/")
	value = strings.ReplaceAll(value, "_", "-")
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return strings.Trim(value, "- ")
}

func inferModelVendorAndFamily(model string) (string, string) {
	switch {
	case strings.HasPrefix(model, "gpt") || strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o2") || strings.HasPrefix(model, "o3") || strings.HasPrefix(model, "o4"):
		return "openai", openAIFamily(model)
	case strings.Contains(model, "claude"):
		return "anthropic", "claude"
	case strings.Contains(model, "gemini"):
		return "google", "gemini"
	case strings.Contains(model, "qwen") || strings.Contains(model, "dashscope") || strings.Contains(model, "tongyi"):
		return "qwen", "qwen"
	case strings.Contains(model, "glm") || strings.Contains(model, "chatglm") || strings.Contains(model, "zhipu"):
		return "glm", "glm"
	case strings.Contains(model, "doubao") || strings.Contains(model, "volcengine") || strings.Contains(model, "bytedance") || strings.HasPrefix(model, "seed-"):
		return "doubao", "doubao"
	case strings.Contains(model, "kimi") || strings.Contains(model, "moonshot") || strings.HasPrefix(model, "k2"):
		return "kimi", "kimi"
	case strings.Contains(model, "minimax") || strings.Contains(model, "abab") || strings.Contains(model, "hailuo"):
		return "minimax", "minimax"
	case strings.Contains(model, "hunyuan") || strings.Contains(model, "hy3") || strings.Contains(model, "tencent"):
		return "hunyuan", "hunyuan"
	case strings.Contains(model, "mimo") || strings.Contains(model, "mi-moment") || strings.Contains(model, "xiaomi"):
		return "mimo", "mimo"
	case strings.Contains(model, "grok") || strings.Contains(model, "xai") || strings.Contains(model, "x-ai"):
		return "xai", "grok"
	case strings.Contains(model, "deepseek"):
		return "deepseek", "deepseek"
	default:
		return "", ""
	}
}

func openAIFamily(model string) string {
	if strings.HasPrefix(model, "gpt") {
		return "gpt"
	}
	if len(model) >= 2 && model[0] == 'o' && isASCIIDigit(model[1]) {
		return "o"
	}
	return "openai"
}

func hasUnexpectedOpenAIReasoningTokens(report *PublicReport, requested modelProfile) bool {
	if report == nil || report.Provider != ProviderOpenAI || report.Metrics.Usage == nil || report.Metrics.Usage.ReasoningTokens <= 0 {
		return false
	}
	if requested.Canonical == "" {
		return false
	}
	if requested.Vendor != "openai" && !strings.HasPrefix(requested.Canonical, "gpt") && !strings.Contains(requested.Canonical, "chat-latest") {
		return false
	}
	return !openAIModelAllowsReasoningTokens(requested.Canonical)
}

func openAIModelAllowsReasoningTokens(model string) bool {
	model = normalizeModelName(model)
	if model == "" {
		return true
	}
	if strings.Contains(model, "codex") || strings.Contains(model, "reasoning") || strings.Contains(model, "thinking") {
		return true
	}
	return len(model) >= 2 && model[0] == 'o' && isASCIIDigit(model[1])
}

func inferModelTier(model string) (string, int) {
	switch {
	case strings.Contains(model, "max"):
		return "max", 100
	case strings.Contains(model, "pro") || strings.Contains(model, "opus"):
		return firstContainedTier(model, "pro", "opus"), 90
	case strings.Contains(model, "sonnet"):
		return "sonnet", 70
	case strings.Contains(model, "mini"):
		return "mini", 45
	case strings.Contains(model, "haiku") || strings.Contains(model, "flash"):
		return firstContainedTier(model, "haiku", "flash"), 35
	case strings.Contains(model, "nano") || strings.Contains(model, "lite"):
		return firstContainedTier(model, "nano", "lite"), 20
	default:
		return "standard", 60
	}
}

func firstContainedTier(model string, values ...string) string {
	for _, value := range values {
		if strings.Contains(model, value) {
			return value
		}
	}
	return values[0]
}

func numericParts(value string) []int {
	parts := make([]int, 0, 3)
	start := -1
	flush := func(end int) {
		if start < 0 {
			return
		}
		token := value[start:end]
		start = -1
		if len(token) >= 8 {
			return
		}
		number, err := strconv.Atoi(token)
		if err == nil {
			parts = append(parts, number)
		}
	}
	for i := 0; i < len(value); i++ {
		if isASCIIDigit(value[i]) {
			if start < 0 {
				start = i
			}
			continue
		}
		flush(i)
	}
	flush(len(value))
	if len(parts) > 3 {
		return parts[:3]
	}
	return parts
}

func compareVersion(left []int, right []int) int {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	maxLen := len(left)
	if len(right) > maxLen {
		maxLen = len(right)
	}
	for i := 0; i < maxLen; i++ {
		lv := 0
		rv := 0
		if i < len(left) {
			lv = left[i]
		}
		if i < len(right) {
			rv = right[i]
		}
		if lv < rv {
			return -1
		}
		if lv > rv {
			return 1
		}
	}
	return 0
}

func versionString(version []int) string {
	if len(version) == 0 {
		return ""
	}
	parts := make([]string, 0, len(version))
	for _, value := range version {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ".")
}

func wrapperVendorMismatch(requestedVendor string, signals []string) string {
	requestedVendor = strings.TrimSpace(requestedVendor)
	if requestedVendor == "" {
		return ""
	}
	for _, signal := range signals {
		vendor := vendorFromWrapperSignal(signal)
		if vendor == "" || vendor == requestedVendor {
			continue
		}
		return vendor
	}
	return ""
}

func protocolExpectedVendor(provider string) string {
	switch strings.TrimSpace(provider) {
	case ProviderOpenAI:
		return "openai"
	case ProviderAnthropic:
		return "anthropic"
	case ProviderGemini:
		return "google"
	default:
		return ""
	}
}

func vendorFromWrapperSignal(signal string) string {
	switch strings.ToLower(strings.TrimSpace(signal)) {
	case "gemini", "vertex", "aistudio", "antigravity":
		return "google"
	case "qwen":
		return "qwen"
	case "glm":
		return "glm"
	case "doubao":
		return "doubao"
	case "minimax":
		return "minimax"
	case "hunyuan":
		return "hunyuan"
	case "kimi":
		return "kimi"
	case "mimo":
		return "mimo"
	case "xai":
		return "xai"
	case "deepseek":
		return "deepseek"
	default:
		return ""
	}
}

func reportWrapperSignals(report *PublicReport) []string {
	if report == nil {
		return nil
	}
	return report.WrapperSignals
}

func reportProvider(report *PublicReport) string {
	if report == nil {
		return ""
	}
	return report.Provider
}

func geminiProbeFallbackInfo(report *PublicReport) (bool, string, string, string) {
	if report == nil || report.Provider != ProviderGemini {
		return false, "", "", ""
	}
	for _, check := range report.Checks {
		if check.ID != "responses_schema" || check.Details == nil {
			continue
		}
		fallback, _ := check.Details["model_fallback"].(bool)
		if !fallback {
			continue
		}
		requested, _ := check.Details["requested_model"].(string)
		probe, _ := check.Details["probe_model"].(string)
		cause, _ := check.Details["model_fallback_cause"].(string)
		return true, requested, probe, cause
	}
	return false, "", "", ""
}

func modelString(report *PublicReport, kind string) string {
	if report == nil {
		return ""
	}
	switch kind {
	case "expected":
		return report.ExpectedModel
	case "response":
		return report.ResponseModel
	default:
		return report.ModelID
	}
}

func modelListContainsRequested(report *PublicReport) (bool, bool) {
	if report == nil {
		return false, false
	}
	for _, check := range report.Checks {
		if check.ID != "models_schema" || check.Details == nil {
			continue
		}
		value, ok := check.Details["model_listed"].(bool)
		return value, ok
	}
	return false, false
}

func modelIdentityDetails(result ModelIdentityResult) map[string]any {
	return map[string]any{
		"status":                        result.Status,
		"reason":                        result.Reason,
		"requested_model":               result.RequestedModel,
		"response_model":                result.ResponseModel,
		"requested_vendor":              result.RequestedVendor,
		"response_vendor":               result.ResponseVendor,
		"requested_family":              result.RequestedFamily,
		"response_family":               result.ResponseFamily,
		"version_delta":                 result.VersionDelta,
		"tier_delta":                    result.TierDelta,
		"model_list_contains_requested": result.ModelListContainsRequested,
		"evidence":                      result.Evidence,
	}
}

func modelIdentityFailureMessage(result ModelIdentityResult) string {
	switch result.Reason {
	case modelIdentityReasonCrossVendorAlias:
		return fmt.Sprintf("请求模型 %s 与响应模型 %s 属于不同厂商，存在跨厂商伪装风险。", result.RequestedModel, result.ResponseModel)
	case modelIdentityReasonFamilyMismatch:
		return fmt.Sprintf("请求模型 %s 与响应模型 %s 属于不同模型家族。", result.RequestedModel, result.ResponseModel)
	case modelIdentityReasonVersionDowngrade:
		return fmt.Sprintf("请求模型 %s 可能被降级为 %s。", result.RequestedModel, result.ResponseModel)
	case modelIdentityReasonTierDowngrade:
		return fmt.Sprintf("请求模型 %s 可能被低配档位 %s 替代。", result.RequestedModel, result.ResponseModel)
	case modelIdentityReasonProtocolVendorMismatch:
		return fmt.Sprintf("请求模型 %s 与当前检测协议厂商不一致，存在兼容通道或模型别名映射风险。", result.RequestedModel)
	case modelIdentityReasonWrapperVendorSignalMismatch:
		return fmt.Sprintf("请求模型 %s 与包装层暴露的上游厂商信号不一致。", result.RequestedModel)
	case modelIdentityReasonReasoningTokensMismatch:
		return fmt.Sprintf("请求模型 %s 声称为非 reasoning 模型，但响应 usage 暴露了 reasoning_tokens。", result.RequestedModel)
	default:
		return "模型身份存在不一致风险。"
	}
}

func modelIdentityPassMessage(result ModelIdentityResult) string {
	switch result.Reason {
	case modelIdentityReasonProbeFallback:
		return "请求模型不可用，已使用同协议可用 Gemini 模型完成探针；未发现跨厂商或降级伪装证据。"
	default:
		return "请求模型与响应模型身份一致。"
	}
}

func modelIdentityWarningMessage(result ModelIdentityResult) string {
	switch result.Reason {
	case modelIdentityReasonResponseModelMissing:
		return "响应未返回 model 字段，无法完整确认模型身份。"
	case modelIdentityReasonProbeFallback:
		return "请求模型不可用，已使用同协议可用模型完成探针；需要结合模型列表确认目标模型是否可调度。"
	case modelIdentityReasonCompatibleAlias:
		return fmt.Sprintf("请求模型 %s 与响应模型 %s 不完全一致，可能是别名或预览版本。", result.RequestedModel, result.ResponseModel)
	default:
		return "模型身份只能部分确认。"
	}
}

func boolPtr(value bool) *bool {
	return &value
}
