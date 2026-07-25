package purity

import "strings"

const (
	capabilityStatusSupported   = "supported"
	capabilityStatusLimited     = "limited"
	capabilityStatusUnsupported = "unsupported"
	capabilityStatusUnknown     = "unknown"
)

func finalizeIndependentScores(report *PublicReport) {
	if report == nil {
		return
	}
	report.ProtocolScore = protocolCompatibilityScore(report)
	report.OfficialBehaviorScore = officialBehaviorScore(report)
	report.MeteringScore, report.MeteringStatus = meteringIntegrityScore(report)
	report.CapabilityMatrix = buildCapabilityMatrix(report)
}

func protocolCompatibilityScore(report *PublicReport) int {
	if report == nil {
		return 0
	}
	included := make([]CheckResult, 0, len(report.Checks))
	for _, check := range report.Checks {
		if check.MaxScore <= 0 || !protocolCheckID(check.ID) {
			continue
		}
		included = append(included, check)
	}
	return percent(checkScore(included), checkMaxScore(included))
}

func officialBehaviorScore(report *PublicReport) int {
	if report == nil {
		return 0
	}
	if hasValidation(report.Validations, "signature") {
		return validationWeightedScore(report, "signature", 100)
	}
	included := make([]CheckResult, 0, 4)
	for _, check := range report.Checks {
		switch check.ID {
		case "claude_thinking_signature", "claude_thinking_budget", "claude_cache_control_overflow", "responses_store_include":
			if check.MaxScore > 0 {
				included = append(included, check)
			}
		}
	}
	if len(included) == 0 {
		return 0
	}
	return percent(checkScore(included), checkMaxScore(included))
}

func meteringIntegrityScore(report *PublicReport) (int, string) {
	if report == nil || report.TokenAudit == nil {
		return 0, capabilityStatusUnknown
	}
	for _, check := range report.Checks {
		if check.ID != "token_audit" || check.MaxScore <= 0 {
			continue
		}
		status := capabilityStatusSupported
		if check.Status == CheckStatusWarn {
			status = capabilityStatusLimited
		} else if check.Status == CheckStatusFail {
			status = capabilityStatusUnsupported
		}
		return percent(check.Score, check.MaxScore), status
	}
	return 0, capabilityStatusUnknown
}

func protocolCheckID(id string) bool {
	switch id {
	case "responses_schema", "claude_messages_schema", "gemini_generate_schema",
		"tool_call", "claude_tool_use", "gemini_tool_call",
		"streaming", "claude_streaming", "gemini_streaming",
		"multimodal", "claude_multimodal", "gemini_multimodal",
		"responses_structured_output", "chat_completions", "chat_completions_n":
		return true
	default:
		return false
	}
}

func buildCapabilityMatrix(report *PublicReport) []CapabilityResult {
	if report == nil {
		return nil
	}
	out := make([]CapabilityResult, 0, len(report.Checks)+1)
	for _, check := range report.Checks {
		if !capabilityCheckID(check.ID) {
			continue
		}
		status := capabilityStatusSupported
		switch check.Status {
		case CheckStatusWarn:
			status = capabilityStatusLimited
		case CheckStatusFail:
			status = capabilityStatusUnsupported
		}
		if skipped, _ := check.Details["skipped"].(bool); skipped {
			status = capabilityStatusUnknown
		}
		out = append(out, CapabilityResult{
			ID:         check.ID,
			Name:       check.Name,
			Status:     status,
			Mode:       capabilityMode(check.ID),
			Summary:    check.Message,
			ReasonCode: capabilityReasonCode(check.ID, status),
			Score:      check.Score,
			MaxScore:   check.MaxScore,
		})
	}

	webSearch := CapabilityResult{
		ID:         "anthropic_managed_websearch",
		Name:       "Anthropic 托管 WebSearch",
		Status:     capabilityStatusUnknown,
		Mode:       "unknown",
		Summary:    "本轮未执行会产生外部搜索成本的托管 WebSearch 主动探针。",
		ReasonCode: "managed_websearch_not_probed",
	}
	if report.Provider == ProviderAnthropic && report.ChannelAttribution != nil && report.ChannelAttribution.Channel == "aws_bedrock" {
		webSearch.Status = capabilityStatusUnsupported
		webSearch.Mode = "unsupported_by_upstream"
		webSearch.Summary = "AWS Bedrock 官方能力边界不支持 Anthropic 托管 WebSearch；其他 Messages 能力继续独立评估。"
		webSearch.ReasonCode = "managed_websearch_unsupported_by_bedrock"
		webSearch.Limitations = []string{"managed_websearch_unsupported"}
	}
	if report.Provider == ProviderAnthropic {
		out = append(out, webSearch)
	}
	return out
}

func capabilityCheckID(id string) bool {
	if protocolCheckID(id) {
		return true
	}
	switch id {
	case "usage", "claude_thinking_signature", "claude_signature_provenance", "channel_attribution", "token_audit":
		return true
	default:
		return false
	}
}

func capabilityMode(id string) string {
	switch {
	case id == "claude_thinking_signature":
		return "provider_constraint"
	case id == "claude_signature_provenance" || id == "channel_attribution":
		return "channel_evidence"
	case id == "token_audit" || id == "usage":
		return "metering"
	case strings.Contains(id, "tool"):
		return "client_tool"
	default:
		return "provider_native"
	}
}

func capabilityReasonCode(id string, status string) string {
	return id + "_" + status
}

func checkScore(checks []CheckResult) int {
	total := 0
	for _, check := range checks {
		total += check.Score
	}
	return total
}

func checkMaxScore(checks []CheckResult) int {
	total := 0
	for _, check := range checks {
		total += check.MaxScore
	}
	return total
}

func cloneCapabilityMatrix(values []CapabilityResult) []CapabilityResult {
	out := append([]CapabilityResult(nil), values...)
	for index := range out {
		out[index].Limitations = append([]string(nil), values[index].Limitations...)
	}
	return out
}
