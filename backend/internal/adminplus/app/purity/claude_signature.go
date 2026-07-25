package purity

import (
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/attribution"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/signature"
)

func buildClaudeSignatureProvenanceCheck(report *PublicReport, nonStreamProbe httpProbe, nonStream signature.JSONAnalysis, streamProbe claudeStreamProbe) CheckResult {
	status := CheckStatusWarn
	message := "正向 thinking 探针未返回可归因的签名，渠道保持未知；这不代表模型身份失败。"
	if report != nil && report.ChannelAttribution != nil {
		switch report.ChannelAttribution.Status {
		case attribution.StatusIdentified:
			if hasSignatureAttributionEvidence(report.ChannelAttribution.Evidence) {
				status = CheckStatusPass
				message = fmt.Sprintf("流式或非流式 thinking signature 已匹配 %s 脱敏结构族。", report.ChannelAttribution.Channel)
			}
		case attribution.StatusLikely:
			message = fmt.Sprintf("thinking signature 提供了 %s 候选，但样本覆盖尚不足。", report.ChannelAttribution.Channel)
		case attribution.StatusConflicted:
			message = "流式与非流式 thinking signature 或渠道证据存在冲突，未强行归因。"
		}
	}
	return CheckResult{
		ID:       "claude_signature_provenance",
		Name:     "Thinking 签名来源",
		Status:   status,
		Score:    0,
		MaxScore: 0,
		Message:  message,
		Details: map[string]any{
			"non_stream_status_code":   nonStreamProbe.StatusCode,
			"non_stream_found":         nonStream.Found,
			"non_stream_parsed":        len(nonStream.Fingerprints),
			"non_stream_parse_errors":  nonStream.ParseErrors,
			"stream_status_code":       streamProbe.StatusCode,
			"stream_found":             streamProbe.SignatureFound,
			"stream_parsed":            len(streamProbe.SignatureFingerprints),
			"stream_parse_errors":      streamProbe.SignatureParseErrors,
			"stream_thinking_observed": streamProbe.SeenThinkingDelta,
			"signature_after_thinking": streamProbe.SignatureAfterThinking,
		},
	}
}

func analyzeClaudeThinkingProbe(probe *httpProbe) signature.JSONAnalysis {
	if probe == nil {
		return signature.JSONAnalysis{}
	}
	analysis := signature.AnalyzeClaudeJSON(probe.Body)
	for index := range probe.Body {
		probe.Body[index] = 0
	}
	probe.Body = nil
	return analysis
}

func hasSignatureAttributionEvidence(evidence []attribution.Evidence) bool {
	for _, item := range evidence {
		if item.Kind == "signature_structure" {
			return true
		}
	}
	return false
}
