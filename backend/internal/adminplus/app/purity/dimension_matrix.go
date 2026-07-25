package purity

import "math"

const (
	dimensionStatusPass                  = "pass"
	dimensionStatusWarn                  = "warn"
	dimensionStatusFail                  = "fail"
	dimensionStatusNotRun                = "not_run"
	dimensionStatusNotApplicable         = "not_applicable"
	dimensionStatusUnsupportedByUpstream = "unsupported_by_upstream"
)

func buildDimensionMatrix(report *PublicReport) []DimensionResult {
	if report == nil {
		return nil
	}
	dimensions := []DimensionResult{
		dimensionFromChecks(report, "tag_check", "LLM 指纹验证", "model_identity", 10, "identity", []string{"base_url", "models_schema", "model_identity"}),
		dimensionFromChecks(report, "stream_structure", "流结构校验", "protocol", 10, "stream", []string{"streaming", "claude_streaming"}),
		dimensionFromChecks(report, "non_stream", "非流结构校验", "protocol", 5, "non_stream", []string{"responses_schema", "claude_messages_schema"}),
		webSearchDimension(report),
		signatureDimension(report),
		structuredOutputDimension(report),
		dimensionFromChecks(report, "server_tool", "工具调用", "behavior", 10, "client_tool", []string{"tool_call", "claude_tool_use"}),
		notRunDimension("token_inject", "Token 注入", "request_integrity", 5, "本轮未执行独立随机 nonce 请求完整性探针。", "active_probe_not_implemented"),
		notRunDimension("knowledge", "知识库检测", "model_identity", 5, "本轮未执行版本化知识题库探针；模型身份不依赖单题结论。", "versioned_knowledge_probe_not_run"),
		notRunDimension("doc_recognition", "文档识别", "multimodal", 10, "本轮未发送合成文档资产，文档识别能力尚未确认。", "synthetic_document_probe_not_run"),
		dimensionFromChecks(report, "image_recognition", "图片识别", "multimodal", 10, "synthetic_image", []string{"multimodal", "claude_multimodal"}),
		gatewayFingerprintDimension(report),
	}
	return dimensions
}

func dimensionFromChecks(report *PublicReport, id string, name string, category string, weight int, mode string, checkIDs []string) DimensionResult {
	result := DimensionResult{
		ID:       id,
		Name:     name,
		Category: category,
		Status:   dimensionStatusNotRun,
		MaxScore: weight,
		Message:  "本轮没有返回可用于该维度的独立探针结果。",
		Mode:     mode,
		Details:  map[string]any{},
	}
	sourceResults := make([]map[string]any, 0, len(checkIDs))
	score := 0
	maxScore := 0
	status := dimensionStatusPass
	for _, checkID := range checkIDs {
		check, ok := checkByID(report.Checks, checkID)
		if !ok {
			continue
		}
		result.SourceCheckIDs = append(result.SourceCheckIDs, check.ID)
		skipped, _ := check.Details["skipped"].(bool)
		sourceResults = append(sourceResults, map[string]any{
			"id":        check.ID,
			"name":      check.Name,
			"status":    check.Status,
			"score":     check.Score,
			"max_score": check.MaxScore,
			"message":   check.Message,
			"skipped":   skipped,
		})
		if skipped {
			continue
		}
		status = worseDimensionStatus(status, check.Status)
		if check.MaxScore > 0 {
			score += check.Score
			maxScore += check.MaxScore
		}
	}
	if len(result.SourceCheckIDs) == 0 {
		return result
	}
	result.Details["source_results"] = sourceResults
	if maxScore <= 0 {
		result.Status = status
		result.Message = dimensionMessageFromSources(sourceResults, result.Message)
		return result
	}
	result.Scored = true
	result.Score = int(math.Round(float64(score) * float64(weight) / float64(maxScore)))
	result.Status = status
	result.Message = dimensionMessageFromSources(sourceResults, result.Message)
	return result
}

func webSearchDimension(report *PublicReport) DimensionResult {
	result := notRunDimension("websearch", "WebSearch", "capability", 10, "本轮未执行会产生外部搜索成本的托管 WebSearch 主动探针。", "managed_websearch_not_probed")
	result.Mode = "provider_native"
	if report.Provider != ProviderAnthropic {
		result.Status = dimensionStatusNotApplicable
		result.Message = "Anthropic 托管 WebSearch 不适用于当前协议；其他搜索能力需要按对应厂商独立检测。"
		result.Limitations = []string{"anthropic_managed_websearch_not_applicable"}
		return result
	}
	if report.ChannelAttribution != nil && report.ChannelAttribution.Channel == "aws_bedrock" {
		result.Status = dimensionStatusUnsupportedByUpstream
		result.Message = "AWS Bedrock 官方能力边界不支持 Anthropic 托管 WebSearch；不影响其他 Messages 能力。"
		result.Limitations = []string{"managed_websearch_unsupported_by_bedrock"}
	}
	return result
}

func signatureDimension(report *PublicReport) DimensionResult {
	switch report.Provider {
	case ProviderAnthropic:
		return dimensionFromChecks(report, "signature_proto", "签名校验", "channel_attribution", 10, "behavior_and_provenance", []string{"claude_thinking_signature", "claude_signature_provenance"})
	case ProviderOpenAI:
		return dimensionFromChecks(report, "signature_proto", "加密推理行为", "official_behavior", 10, "encrypted_reasoning_behavior", []string{"responses_store_include"})
	default:
		result := notRunDimension("signature_proto", "签名校验", "channel_attribution", 10, "当前协议没有与 Claude thinking signature 等价的本地归因探针。", "signature_probe_not_applicable")
		result.Status = dimensionStatusNotApplicable
		return result
	}
}

func structuredOutputDimension(report *PublicReport) DimensionResult {
	if report.Provider == ProviderOpenAI {
		return dimensionFromChecks(report, "output_config", "结构化输出", "capability", 10, "json_schema", []string{"responses_structured_output"})
	}
	return notRunDimension("output_config", "结构化输出", "capability", 10, "本轮未执行当前协议的独立结构化输出 schema 探针。", "structured_output_probe_not_run")
}

func gatewayFingerprintDimension(report *PublicReport) DimensionResult {
	result := dimensionFromChecks(report, "fingerprint", "协议与包装指纹", "gateway", 5, "evidence_only", []string{"wrapper_fingerprint"})
	result.Scored = false
	result.Score = 0
	result.Message = firstNonEmptyString(result.Message, "包装信号作为事实展示，不自动扣减协议兼容分。")
	result.Limitations = appendUniqueString(result.Limitations, "gateway_fingerprint_not_protocol_score")
	return result
}

func notRunDimension(id string, name string, category string, weight int, message string, limitation string) DimensionResult {
	result := DimensionResult{
		ID:       id,
		Name:     name,
		Category: category,
		Status:   dimensionStatusNotRun,
		MaxScore: weight,
		Message:  message,
		Details:  map[string]any{"reason": limitation},
	}
	if limitation != "" {
		result.Limitations = []string{limitation}
	}
	return result
}

func checkByID(checks []CheckResult, id string) (CheckResult, bool) {
	for _, check := range checks {
		if check.ID == id {
			return check, true
		}
	}
	return CheckResult{}, false
}

func worseDimensionStatus(current string, next string) string {
	rank := map[string]int{dimensionStatusPass: 0, dimensionStatusWarn: 1, dimensionStatusFail: 2}
	if rank[next] > rank[current] {
		return next
	}
	return current
}

func dimensionMessageFromSources(sources []map[string]any, fallback string) string {
	for index := len(sources) - 1; index >= 0; index-- {
		if message, _ := sources[index]["message"].(string); message != "" {
			return message
		}
	}
	return fallback
}

func cloneDimensionMatrix(values []DimensionResult) []DimensionResult {
	out := append([]DimensionResult(nil), values...)
	for index := range out {
		out[index].SourceCheckIDs = append([]string(nil), values[index].SourceCheckIDs...)
		out[index].Limitations = append([]string(nil), values[index].Limitations...)
		out[index].Details = cloneDimensionDetails(values[index].Details)
	}
	return out
}

func cloneDimensionDetails(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		switch typed := value.(type) {
		case []string:
			out[key] = append([]string(nil), typed...)
		case []map[string]any:
			items := make([]map[string]any, len(typed))
			for index := range typed {
				items[index] = cloneDimensionDetails(typed[index])
			}
			out[key] = items
		case map[string]any:
			out[key] = cloneDimensionDetails(typed)
		default:
			out[key] = value
		}
	}
	return out
}
