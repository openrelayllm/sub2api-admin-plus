package purity

import (
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

func buildGeminiBaseURLCheck(host string, officialHost bool) CheckResult {
	if officialHost {
		return CheckResult{
			ID:       "base_url",
			Name:     "API Base 域名",
			Status:   CheckStatusPass,
			Score:    20,
			MaxScore: 20,
			Message:  "命中 Google Gemini 官方 API 域名。",
			Details:  map[string]any{"host": host, "official_host": true},
		}
	}
	return CheckResult{
		ID:       "base_url",
		Name:     "API Base 域名",
		Status:   CheckStatusPass,
		Score:    20,
		MaxScore: 20,
		Message:  "当前为自定义 Gemini API Base；仅作为链路信息，不单独影响纯度评分。",
		Details:  map[string]any{"host": host, "official_host": false},
	}
}

func buildGeminiModelsCheck(probe httpProbe, model string) CheckResult {
	details := probeDetails(probe)
	if probe.StatusCode == 0 {
		return failCheck("models_schema", "模型列表结构", 15, "无法连接 Gemini 模型列表端点。", details)
	}
	if probe.ErrorClass == errorClassAccountBalanceInsufficient {
		return failCheck("models_schema", "模型列表结构", 15, "账号余额不足，Gemini 模型列表探测无法执行。", details)
	}
	if probe.StatusCode == http.StatusUnauthorized || probe.StatusCode == http.StatusForbidden {
		return failCheck("models_schema", "模型列表结构", 15, "Gemini API Key 鉴权失败。", details)
	}
	if probe.StatusCode < 200 || probe.StatusCode >= 300 {
		return CheckResult{ID: "models_schema", Name: "模型列表结构", Status: CheckStatusWarn, Score: 4, MaxScore: 15, Message: "Gemini 模型列表端点未返回标准 2xx 响应。", Details: details}
	}
	models := gjson.GetBytes(probe.Body, "models")
	if !models.IsArray() {
		return CheckResult{ID: "models_schema", Name: "模型列表结构", Status: CheckStatusWarn, Score: 6, MaxScore: 15, Message: "模型列表响应不是标准 Gemini models[] 结构。", Details: details}
	}
	modelListed := false
	count := 0
	normalizedExpected := strings.TrimPrefix(strings.TrimSpace(model), "models/")
	for _, item := range models.Array() {
		count++
		name := strings.TrimPrefix(strings.TrimSpace(item.Get("name").String()), "models/")
		if name == normalizedExpected {
			modelListed = true
		}
	}
	details["model_count"] = count
	details["model_listed"] = modelListed
	if modelListed {
		return passCheck("models_schema", "模型列表结构", 15, "Gemini models[] 结构标准，且包含本次检测模型。", details)
	}
	return CheckResult{ID: "models_schema", Name: "模型列表结构", Status: CheckStatusWarn, Score: 12, MaxScore: 15, Message: "Gemini models[] 结构标准，但未列出本次检测模型。", Details: details}
}

func buildGeminiGenerateContentCheck(probe httpProbe, apiKey string) CheckResult {
	details := probeDetails(probe)
	if probe.StatusCode == 0 {
		return failCheck("responses_schema", "GenerateContent 非流式结构", 20, "无法连接 GenerateContent 端点。", details)
	}
	if probe.ErrorClass == errorClassAccountBalanceInsufficient {
		return failCheck("responses_schema", "GenerateContent 非流式结构", 20, "账号余额不足，GenerateContent 探测无法执行。", details)
	}
	if probe.StatusCode == http.StatusUnauthorized || probe.StatusCode == http.StatusForbidden {
		return failCheck("responses_schema", "GenerateContent 非流式结构", 20, "GenerateContent 端点鉴权失败。", details)
	}
	if probe.StatusCode == http.StatusNotFound || probe.StatusCode == http.StatusMethodNotAllowed {
		return failCheck("responses_schema", "GenerateContent 非流式结构", 20, "GenerateContent 端点不存在或方法不支持。", details)
	}
	if probe.StatusCode < 200 || probe.StatusCode >= 300 {
		details["error_message"] = sanitizeMessage(upstreamErrorMessage(probe.Body), apiKey)
		return failCheck("responses_schema", "GenerateContent 非流式结构", 20, "GenerateContent 端点未返回可用响应。", details)
	}
	if gjson.GetBytes(probe.Body, "candidates").IsArray() {
		return passCheck("responses_schema", "GenerateContent 非流式结构", 20, "GenerateContent 响应结构符合 Gemini 预期。", details)
	}
	return CheckResult{ID: "responses_schema", Name: "GenerateContent 非流式结构", Status: CheckStatusWarn, Score: 8, MaxScore: 20, Message: "GenerateContent 返回 2xx，但响应结构不完整。", Details: details}
}

func buildGeminiToolCallCheck(probe httpProbe) CheckResult {
	details := probeDetails(probe)
	if probe.StatusCode < 200 || probe.StatusCode >= 300 {
		return failCheck("tool_call", "强制工具调用", 20, "GenerateContent 探测未成功，无法确认工具调用。", details)
	}
	ok, toolDetails := geminiBodyHasExpectedFunctionCall(probe.Body)
	for key, value := range toolDetails {
		details[key] = value
	}
	if ok {
		return passCheck("tool_call", "强制工具调用", 20, "functionCallingConfig=ANY 成功产出 probe_ping(ok=true) functionCall。", details)
	}
	return failCheck("tool_call", "强制工具调用", 20, "强制工具调用没有产出预期 functionCall。", details)
}

func buildGeminiUsageCheck(usage *TokenUsage, probe httpProbe) CheckResult {
	details := probeDetails(probe)
	if probe.StatusCode < 200 || probe.StatusCode >= 300 {
		return failCheck("usage", "Usage 计量", 10, "GenerateContent 探测未成功，无法读取 usageMetadata。", details)
	}
	if usage == nil {
		return failCheck("usage", "Usage 计量", 10, "响应缺少 usageMetadata 计量字段。", details)
	}
	details["input_tokens"] = usage.InputTokens
	details["output_tokens"] = usage.OutputTokens
	details["total_tokens"] = usage.TotalTokens
	if usage.TotalTokens >= usage.InputTokens+usage.OutputTokens && usage.TotalTokens > 0 {
		return passCheck("usage", "Usage 计量", 10, "usageMetadata token 计量字段完整。", details)
	}
	return CheckResult{ID: "usage", Name: "Usage 计量", Status: CheckStatusWarn, Score: 5, MaxScore: 10, Message: "usageMetadata 字段存在，但 token 汇总不完全一致。", Details: details}
}

func buildGeminiStreamingCheck(probe streamProbe, apiKey string) CheckResult {
	details := map[string]any{
		"status_code":      probe.StatusCode,
		"first_token_ms":   probe.FirstTokenMS,
		"total_latency_ms": probe.TotalLatencyMS,
		"seen_data":        probe.SeenData,
		"seen_delta":       probe.SeenDelta,
		"seen_completed":   probe.SeenCompleted,
		"seen_done":        probe.SeenDone,
	}
	if probe.ErrorClass != "" {
		details["error_class"] = probe.ErrorClass
		details["error_message"] = sanitizeMessage(probe.ErrorMessage, apiKey)
	}
	if probe.StatusCode == 0 {
		return failCheck("streaming", "StreamGenerateContent 流式事件", 15, "无法连接 StreamGenerateContent 流式端点。", details)
	}
	if probe.ErrorClass == errorClassAccountBalanceInsufficient {
		return failCheck("streaming", "StreamGenerateContent 流式事件", 15, "账号余额不足，StreamGenerateContent 流式探测无法执行。", details)
	}
	if probe.StatusCode < 200 || probe.StatusCode >= 300 {
		return failCheck("streaming", "StreamGenerateContent 流式事件", 15, "StreamGenerateContent 流式端点未返回可用响应。", details)
	}
	if probe.SeenDelta && (probe.SeenCompleted || probe.SeenDone) && probe.ErrorClass == "" {
		return passCheck("streaming", "StreamGenerateContent 流式事件", 15, "SSE 文本增量与完成事件完整。", details)
	}
	if probe.SeenData && (probe.SeenCompleted || probe.SeenDone) {
		return CheckResult{ID: "streaming", Name: "StreamGenerateContent 流式事件", Status: CheckStatusWarn, Score: 8, MaxScore: 15, Message: "SSE 生命周期结束，但未观察到文本增量。", Details: details}
	}
	return failCheck("streaming", "StreamGenerateContent 流式事件", 15, "SSE 生命周期不完整。", details)
}

func buildGeminiMultimodalCheck(probe httpProbe, apiKey string) CheckResult {
	details := probeDetails(probe)
	if probe.ErrorMessage != "" {
		details["error_message"] = sanitizeMessage(probe.ErrorMessage, apiKey)
	}
	if probe.StatusCode == 0 {
		return failCheck("multimodal", "多模态输入", 10, "无法连接 Gemini 多模态探测端点。", details)
	}
	if probe.ErrorClass == errorClassAccountBalanceInsufficient {
		return failCheck("multimodal", "多模态输入", 10, "账号余额不足，多模态探测无法执行。", details)
	}
	if probe.StatusCode >= 200 && probe.StatusCode < 300 && gjson.GetBytes(probe.Body, "candidates").IsArray() {
		return passCheck("multimodal", "多模态输入", 10, "GenerateContent 接受 inlineData image/png 多模态输入结构。", details)
	}
	if probe.StatusCode == http.StatusBadRequest || probe.StatusCode == http.StatusUnprocessableEntity {
		return CheckResult{ID: "multimodal", Name: "多模态输入", Status: CheckStatusWarn, Score: 5, MaxScore: 10, Message: "端点存在，但当前模型或上游不接受 inlineData 图像输入。", Details: details}
	}
	return failCheck("multimodal", "多模态输入", 10, "多模态探测未返回标准 GenerateContent 响应。", details)
}

func buildGeminiTokenAuditUnsupportedCheck() CheckResult {
	return CheckResult{
		ID:       "token_audit",
		Name:     "Token 用量审计",
		Status:   CheckStatusWarn,
		Score:    0,
		MaxScore: 0,
		Message:  "Gemini 原生检测当前读取 usageMetadata，但尚未执行多轮成本倍率审计。",
		Details:  map[string]any{"skipped": true, "reason": "gemini_token_audit_not_implemented"},
	}
}

func geminiBodyHasExpectedFunctionCall(body []byte) (bool, map[string]any) {
	details := map[string]any{"function_call_seen": false}
	for _, candidate := range gjson.GetBytes(body, "candidates").Array() {
		for _, part := range candidate.Get("content.parts").Array() {
			functionCall := part.Get("functionCall")
			if !functionCall.Exists() {
				continue
			}
			details["function_call_seen"] = true
			details["function_name"] = functionCall.Get("name").String()
			if strings.TrimSpace(functionCall.Get("name").String()) != "probe_ping" {
				continue
			}
			if functionCall.Get("args.ok").Bool() {
				details["arguments_ok"] = true
				return true, details
			}
			details["arguments_ok"] = false
		}
	}
	return false, details
}

func skippedGeminiCoreChecks(message string) []CheckResult {
	return []CheckResult{
		failCheck("responses_schema", "GenerateContent 非流式结构", 20, message, nil),
		failCheck("tool_call", "强制工具调用", 20, message, nil),
		failCheck("streaming", "StreamGenerateContent 流式事件", 15, message, nil),
		failCheck("usage", "Usage 计量", 10, message, nil),
		failCheck("multimodal", "多模态输入", 10, message, nil),
		buildGeminiTokenAuditUnsupportedCheck(),
	}
}
