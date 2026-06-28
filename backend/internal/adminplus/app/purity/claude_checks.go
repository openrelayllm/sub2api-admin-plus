package purity

import (
	"github.com/tidwall/gjson"
	"net/http"
	"strings"
)

func buildClaudeBaseURLCheck(host string, officialHost bool) CheckResult {
	if officialHost {
		return CheckResult{
			ID:       "base_url",
			Name:     "API Base 域名",
			Status:   CheckStatusPass,
			Score:    20,
			MaxScore: 20,
			Message:  "命中 Anthropic 官方 API 域名。",
			Details:  map[string]any{"host": host, "official_host": true},
		}
	}
	return CheckResult{
		ID:       "base_url",
		Name:     "API Base 域名",
		Status:   CheckStatusPass,
		Score:    20,
		MaxScore: 20,
		Message:  "当前为自定义 Claude API Base；仅作为链路信息，不单独影响纯度评分。",
		Details:  map[string]any{"host": host, "official_host": false},
	}
}

func buildClaudeMessagesSchemaCheck(probe httpProbe, apiKey string) CheckResult {
	details := probeDetails(probe)
	if probe.StatusCode == 0 {
		return failCheck("claude_messages_schema", "Messages 非流式结构", 20, "无法连接 Messages 端点。", details)
	}
	if probe.ErrorClass == errorClassAccountBalanceInsufficient {
		return failCheck("claude_messages_schema", "Messages 非流式结构", 20, "账号余额不足，Messages 探测无法执行。", details)
	}
	if probe.StatusCode == http.StatusUnauthorized || probe.StatusCode == http.StatusForbidden {
		return failCheck("claude_messages_schema", "Messages 非流式结构", 20, "API Key 鉴权失败。", details)
	}
	if probe.StatusCode == http.StatusNotFound || probe.StatusCode == http.StatusMethodNotAllowed {
		return failCheck("claude_messages_schema", "Messages 非流式结构", 20, "Messages 端点不存在或方法不支持。", details)
	}
	if probe.StatusCode < 200 || probe.StatusCode >= 300 {
		details["error_message"] = sanitizeMessage(upstreamErrorMessage(probe.Body), apiKey)
		return failCheck("claude_messages_schema", "Messages 非流式结构", 20, "Messages 端点未返回可用响应。", details)
	}
	if gjson.GetBytes(probe.Body, "type").String() == "message" && gjson.GetBytes(probe.Body, "content").IsArray() {
		details["response_model"] = gjson.GetBytes(probe.Body, "model").String()
		return passCheck("claude_messages_schema", "Messages 非流式结构", 20, "Messages 响应结构符合 Anthropic 预期。", details)
	}
	return CheckResult{ID: "claude_messages_schema", Name: "Messages 非流式结构", Status: CheckStatusWarn, Score: 8, MaxScore: 20, Message: "Messages 返回 2xx，但响应结构不完整。", Details: details}
}

func buildClaudeToolUseCheck(probe httpProbe) CheckResult {
	details := probeDetails(probe)
	if probe.StatusCode < 200 || probe.StatusCode >= 300 {
		return failCheck("claude_tool_use", "强制工具调用", 20, "Messages 探测未成功，无法确认工具调用。", details)
	}
	ok, toolDetails := claudeBodyHasExpectedToolUse(probe.Body)
	for key, value := range toolDetails {
		details[key] = value
	}
	if ok {
		return passCheck("claude_tool_use", "强制工具调用", 20, "tool_choice 成功产出 probe_ping(ok=true) tool_use。", details)
	}
	return failCheck("claude_tool_use", "强制工具调用", 20, "强制工具调用没有产出预期 tool_use。", details)
}

func buildClaudeUsageCheck(usage *TokenUsage, probe httpProbe) CheckResult {
	details := probeDetails(probe)
	if probe.StatusCode < 200 || probe.StatusCode >= 300 {
		return failCheck("claude_usage", "Usage 计量", 10, "Messages 探测未成功，无法读取 usage。", details)
	}
	if usage == nil {
		return failCheck("claude_usage", "Usage 计量", 10, "响应缺少 usage 计量字段。", details)
	}
	details["input_tokens"] = usage.InputTokens
	details["output_tokens"] = usage.OutputTokens
	details["cache_creation_input_tokens"] = usage.CacheCreationTokens
	details["cache_read_input_tokens"] = usage.CachedTokens
	if usage.InputTokens+usage.OutputTokens+usage.CacheCreationTokens+usage.CachedTokens > 0 {
		return passCheck("claude_usage", "Usage 计量", 10, "usage token 计量字段完整。", details)
	}
	return CheckResult{ID: "claude_usage", Name: "Usage 计量", Status: CheckStatusWarn, Score: 5, MaxScore: 10, Message: "usage 字段存在，但 token 值为空。", Details: details}
}

func buildClaudeThinkingSignatureCheck(probe httpProbe, apiKey string) CheckResult {
	details := probeDetails(probe)
	if probe.ErrorMessage != "" {
		details["error_message"] = sanitizeMessage(probe.ErrorMessage, apiKey)
	}
	if probe.StatusCode == 0 {
		return failCheck("claude_thinking_signature", "Thinking 签名拒绝", 20, "无法连接 Messages 签名探测端点。", details)
	}
	if probe.ErrorClass == errorClassAccountBalanceInsufficient {
		return failCheck("claude_thinking_signature", "Thinking 签名拒绝", 20, "账号余额不足，签名探测无法执行。", details)
	}
	if probe.StatusCode >= 200 && probe.StatusCode < 300 {
		return failCheck("claude_thinking_signature", "Thinking 签名拒绝", 20, "伪造 thinking signature 被接受，未观察到 Claude 官方签名校验。", details)
	}
	errorMessage := strings.ToLower(firstNonEmptyString(probe.ErrorMessage, upstreamErrorMessage(probe.Body)))
	if probe.StatusCode == http.StatusBadRequest && strings.Contains(errorMessage, "signature") {
		return passCheck("claude_thinking_signature", "Thinking 签名拒绝", 20, "伪造 thinking signature 被拒绝，符合 Claude 签名校验预期。", details)
	}
	if probe.StatusCode >= 400 && probe.StatusCode < 500 && (strings.Contains(errorMessage, "thinking") || strings.Contains(errorMessage, "content")) {
		return CheckResult{ID: "claude_thinking_signature", Name: "Thinking 签名拒绝", Status: CheckStatusWarn, Score: 10, MaxScore: 20, Message: "签名探测被拒绝，但错误形态不完全匹配 Claude signature 校验。", Details: details}
	}
	return failCheck("claude_thinking_signature", "Thinking 签名拒绝", 20, "签名探测未返回 Claude 预期的 signature 拒绝错误。", details)
}

func buildClaudeThinkingBudgetCheck(probe httpProbe, apiKey string) CheckResult {
	details := probeDetails(probe)
	if probe.ErrorMessage != "" {
		details["error_message"] = sanitizeMessage(probe.ErrorMessage, apiKey)
	}
	if probe.StatusCode == 0 {
		return failCheck("claude_thinking_budget", "Thinking 预算约束", 10, "无法连接 Messages thinking 预算探测端点。", details)
	}
	if probe.ErrorClass == errorClassAccountBalanceInsufficient {
		return failCheck("claude_thinking_budget", "Thinking 预算约束", 10, "账号余额不足，thinking 预算探测无法执行。", details)
	}
	if probe.StatusCode >= 200 && probe.StatusCode < 300 {
		return failCheck("claude_thinking_budget", "Thinking 预算约束", 10, "thinking budget 越界请求被接受，未观察到 Claude 官方 max_tokens > budget_tokens 约束。", details)
	}
	errorMessage := strings.ToLower(firstNonEmptyString(probe.ErrorMessage, upstreamErrorMessage(probe.Body)))
	if probe.StatusCode == http.StatusBadRequest && (strings.Contains(errorMessage, "budget") || strings.Contains(errorMessage, "max_tokens") || strings.Contains(errorMessage, "thinking")) {
		return passCheck("claude_thinking_budget", "Thinking 预算约束", 10, "thinking budget 越界请求被拒绝，符合 Claude 官方约束。", details)
	}
	if probe.StatusCode >= 400 && probe.StatusCode < 500 {
		return CheckResult{ID: "claude_thinking_budget", Name: "Thinking 预算约束", Status: CheckStatusWarn, Score: 5, MaxScore: 10, Message: "thinking budget 探测被拒绝，但错误形态不完全匹配 Claude 官方约束。", Details: details}
	}
	return failCheck("claude_thinking_budget", "Thinking 预算约束", 10, "thinking budget 探测未返回 Claude 预期的拒绝错误。", details)
}

func buildClaudeCacheControlOverflowCheck(probe httpProbe, apiKey string) CheckResult {
	details := probeDetails(probe)
	if probe.ErrorMessage != "" {
		details["error_message"] = sanitizeMessage(probe.ErrorMessage, apiKey)
	}
	if probe.StatusCode == 0 {
		return failCheck("claude_cache_control_overflow", "Cache-Control 上限约束", 10, "无法连接 Messages cache_control 探测端点。", details)
	}
	if probe.ErrorClass == errorClassAccountBalanceInsufficient {
		return failCheck("claude_cache_control_overflow", "Cache-Control 上限约束", 10, "账号余额不足，cache_control 探测无法执行。", details)
	}
	if probe.StatusCode >= 200 && probe.StatusCode < 300 {
		return failCheck("claude_cache_control_overflow", "Cache-Control 上限约束", 10, "过量 cache_control system block 被接受，未观察到 Claude 官方 cache_control 数量约束。", details)
	}
	if probe.StatusCode == http.StatusBadRequest {
		return passCheck("claude_cache_control_overflow", "Cache-Control 上限约束", 10, "过量 cache_control system block 被拒绝，符合 Claude 官方约束。", details)
	}
	if probe.StatusCode >= 400 && probe.StatusCode < 500 {
		return CheckResult{ID: "claude_cache_control_overflow", Name: "Cache-Control 上限约束", Status: CheckStatusWarn, Score: 5, MaxScore: 10, Message: "cache_control 探测被拒绝，但错误形态不完全匹配 Claude 官方约束。", Details: details}
	}
	return failCheck("claude_cache_control_overflow", "Cache-Control 上限约束", 10, "cache_control 探测未返回 Claude 预期的拒绝错误。", details)
}

func buildClaudeMultimodalCheck(probe httpProbe, apiKey string) CheckResult {
	details := probeDetails(probe)
	if probe.ErrorMessage != "" {
		details["error_message"] = sanitizeMessage(probe.ErrorMessage, apiKey)
	}
	if probe.StatusCode == 0 {
		return failCheck("claude_multimodal", "多模态输入", 10, "无法连接 Messages 多模态探测端点。", details)
	}
	if probe.ErrorClass == errorClassAccountBalanceInsufficient {
		return failCheck("claude_multimodal", "多模态输入", 10, "账号余额不足，多模态探测无法执行。", details)
	}
	if probe.StatusCode >= 200 && probe.StatusCode < 300 && gjson.GetBytes(probe.Body, "type").String() == "message" {
		return passCheck("claude_multimodal", "多模态输入", 10, "Messages 接受 image block 多模态输入结构。", details)
	}
	if probe.StatusCode == http.StatusBadRequest || probe.StatusCode == http.StatusUnprocessableEntity {
		return CheckResult{ID: "claude_multimodal", Name: "多模态输入", Status: CheckStatusWarn, Score: 5, MaxScore: 10, Message: "端点存在，但当前模型或上游不接受 image block。", Details: details}
	}
	return failCheck("claude_multimodal", "多模态输入", 10, "多模态探测未返回标准 Messages 响应。", details)
}

func claudeBodyHasExpectedToolUse(body []byte) (bool, map[string]any) {
	details := map[string]any{"tool_use_seen": false}
	content := gjson.GetBytes(body, "content")
	if !content.IsArray() {
		return false, details
	}
	for _, item := range content.Array() {
		if strings.TrimSpace(item.Get("type").String()) != "tool_use" {
			continue
		}
		details["tool_use_seen"] = true
		details["tool_name"] = item.Get("name").String()
		if strings.TrimSpace(item.Get("name").String()) != "probe_ping" {
			continue
		}
		if item.Get("input.ok").Bool() {
			details["arguments_ok"] = true
			return true, details
		}
		details["arguments_ok"] = false
	}
	return false, details
}

func parseClaudeUsage(body []byte) *TokenUsage {
	usage := gjson.GetBytes(body, "usage")
	if !usage.Exists() || !usage.IsObject() {
		return nil
	}
	input := usage.Get("input_tokens").Int()
	output := usage.Get("output_tokens").Int()
	cacheCreate := usage.Get("cache_creation_input_tokens").Int()
	cacheRead := usage.Get("cache_read_input_tokens").Int()
	return &TokenUsage{
		InputTokens:         input,
		OutputTokens:        output,
		TotalTokens:         input + output + cacheCreate + cacheRead,
		CacheCreationTokens: cacheCreate,
		CachedTokens:        cacheRead,
	}
}

func buildClaudeStreamingCheck(probe claudeStreamProbe, apiKey string) CheckResult {
	details := map[string]any{
		"status_code":              probe.StatusCode,
		"first_token_ms":           probe.FirstTokenMS,
		"total_latency_ms":         probe.TotalLatencyMS,
		"seen_data":                probe.SeenData,
		"seen_message_start":       probe.SeenMessageStart,
		"seen_content_block_start": probe.SeenContentBlockStart,
		"seen_delta":               probe.SeenDelta,
		"seen_message_delta":       probe.SeenMessageDelta,
		"seen_message_stop":        probe.SeenMessageStop,
		"seen_tool_use":            probe.SeenToolUse,
	}
	if probe.ErrorClass != "" {
		details["error_class"] = probe.ErrorClass
		details["error_message"] = sanitizeMessage(probe.ErrorMessage, apiKey)
	}
	if probe.StatusCode == 0 {
		return failCheck("claude_streaming", "Messages 流式事件", 15, "无法连接 Messages 流式端点。", details)
	}
	if probe.ErrorClass == errorClassAccountBalanceInsufficient {
		return failCheck("claude_streaming", "Messages 流式事件", 15, "账号余额不足，Messages 流式探测无法执行。", details)
	}
	if probe.StatusCode < 200 || probe.StatusCode >= 300 {
		return failCheck("claude_streaming", "Messages 流式事件", 15, "Messages 流式端点未返回可用响应。", details)
	}
	if probe.SeenMessageStart && probe.SeenContentBlockStart && probe.SeenDelta && probe.SeenMessageDelta && probe.SeenMessageStop && probe.ErrorClass == "" {
		return passCheck("claude_streaming", "Messages 流式事件", 15, "SSE message/content/delta/stop 事件序列完整。", details)
	}
	if probe.SeenMessageStart && probe.SeenMessageStop {
		return CheckResult{ID: "claude_streaming", Name: "Messages 流式事件", Status: CheckStatusWarn, Score: 8, MaxScore: 15, Message: "SSE 生命周期结束，但中间 delta 或 message_delta 不完整。", Details: details}
	}
	return failCheck("claude_streaming", "Messages 流式事件", 15, "SSE 生命周期不完整。", details)
}

func skippedClaudeCoreChecks(message string) []CheckResult {
	return []CheckResult{
		failCheck("claude_streaming", "Messages 流式事件", 15, message, nil),
		failCheck("claude_thinking_signature", "Thinking 签名拒绝", 20, message, nil),
		failCheck("claude_thinking_budget", "Thinking 预算约束", 10, message, nil),
		failCheck("claude_cache_control_overflow", "Cache-Control 上限约束", 10, message, nil),
		failCheck("claude_multimodal", "多模态输入", 10, message, nil),
		failCheck("token_audit", "Token 用量审计", 15, message, nil),
	}
}
