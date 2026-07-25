package purity

import (
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/attribution"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/signature"
)

type channelSignatureEvidence struct {
	NonStream         signature.JSONAnalysis
	Stream            []signature.Fingerprint
	StreamFound       int
	StreamParseErrors int
}

func applyChannelAttribution(report *PublicReport, headerSets []map[string]string, signatureEvidence channelSignatureEvidence) {
	if report == nil {
		return
	}
	registry, registryErr := signature.DefaultRegistry()
	observations := make([]attribution.SignatureObservation, 0, len(signatureEvidence.NonStream.Fingerprints)+len(signatureEvidence.Stream))
	if registryErr == nil {
		for _, fingerprint := range signatureEvidence.NonStream.Fingerprints {
			observations = append(observations, attribution.SignatureObservation{
				Transport:      "non_stream",
				Classification: registry.Classify(fingerprint, report.ResponseModel),
			})
		}
		for _, fingerprint := range signatureEvidence.Stream {
			observations = append(observations, attribution.SignatureObservation{
				Transport:      "stream",
				Classification: registry.Classify(fingerprint, report.ResponseModel),
			})
		}
	}
	failures := signatureEvidence.NonStream.ParseErrors + signatureEvidence.StreamParseErrors
	if registryErr != nil {
		failures++
	}
	report.ChannelAttribution = pointerToAttributionResult(attribution.Evaluate(attribution.Input{
		Provider:          report.Provider,
		Host:              report.APIBaseHost,
		Model:             firstNonEmptyString(report.ResponseModel, report.ExpectedModel, report.ModelID),
		CheckedAt:         report.CheckedAt,
		HeaderSets:        headerSets,
		WrapperSignals:    report.WrapperSignals,
		StreamChannel:     report.StreamChannel,
		NonStreamChannel:  report.NonStreamChannel,
		Signatures:        observations,
		SignatureFound:    signatureEvidence.NonStream.Found + signatureEvidence.StreamFound,
		SignatureFailures: failures,
	}))
}

func appendAndEmitChannelAttribution(report *PublicReport, emit PublicCheckEventSink) {
	if report == nil || report.ChannelAttribution == nil {
		return
	}
	result := report.ChannelAttribution
	status := CheckStatusWarn
	message := "渠道证据不足，当前上游来源保持未知。"
	switch result.Status {
	case attribution.StatusIdentified:
		status = CheckStatusPass
		message = fmt.Sprintf("已识别上游渠道 %s，置信度 %.0f%%。", result.Channel, result.Confidence*100)
	case attribution.StatusLikely:
		message = fmt.Sprintf("上游渠道很可能为 %s，置信度 %.0f%%，仍需更多独立样本确认。", result.Channel, result.Confidence*100)
	case attribution.StatusConflicted:
		message = "流式、非流式或渠道元数据存在互斥证据，未强行选择来源。"
	}
	check := CheckResult{
		ID:       "channel_attribution",
		Name:     "上游渠道归因",
		Status:   status,
		Score:    0,
		MaxScore: 0,
		Message:  message,
		Details: map[string]any{
			"channel":             result.Channel,
			"confidence":          result.Confidence,
			"status":              result.Status,
			"evidence_count":      len(result.Evidence),
			"contradiction_count": len(result.Contradictions),
			"reason_codes":        append([]string(nil), result.ReasonCodes...),
			"detector_version":    result.DetectorVersion,
		},
	}
	upsertAndEmitCheck(report, emit, check)
	upsertAndEmitValidation(report, emit, validationFromExecutedChecks("channel_attribution", "上游渠道归因", []CheckResult{check}))
}

func upsertAndEmitCheck(report *PublicReport, emit PublicCheckEventSink, check CheckResult) {
	if report == nil {
		return
	}
	for index := range report.Checks {
		if report.Checks[index].ID == check.ID {
			report.Checks[index] = check
			checkCopy := check
			emitPublicCheckEvent(emit, PublicCheckEvent{Type: PublicCheckEventCheck, ReportID: report.ReportID, Check: &checkCopy})
			return
		}
	}
	appendAndEmitChecks(report, emit, check)
}

func ensureChannelAttribution(report *PublicReport) {
	if report == nil || report.ChannelAttribution != nil {
		return
	}
	applyChannelAttribution(report, nil, channelSignatureEvidence{})
	check := CheckResult{
		ID:       "channel_attribution",
		Name:     "上游渠道归因",
		Status:   CheckStatusWarn,
		Score:    0,
		MaxScore: 0,
		Message:  "渠道证据不足，当前上游来源保持未知。",
		Details:  map[string]any{"status": attribution.StatusUnknown},
	}
	report.Checks = append(report.Checks, check)
	upsertValidation(report, validationFromExecutedChecks("channel_attribution", "上游渠道归因", []CheckResult{check}))
}

func pointerToAttributionResult(result attribution.Result) *attribution.Result {
	return &result
}

func cloneChannelAttribution(result *attribution.Result) *attribution.Result {
	if result == nil {
		return nil
	}
	out := *result
	out.Evidence = cloneAttributionEvidence(result.Evidence)
	out.Contradictions = cloneAttributionEvidence(result.Contradictions)
	out.Limitations = append([]string(nil), result.Limitations...)
	out.ReasonCodes = append([]string(nil), result.ReasonCodes...)
	return &out
}

func cloneAttributionEvidence(evidence []attribution.Evidence) []attribution.Evidence {
	out := append([]attribution.Evidence(nil), evidence...)
	for index := range out {
		out[index].Limitations = append([]string(nil), evidence[index].Limitations...)
		out[index].RiskCodes = append([]string(nil), evidence[index].RiskCodes...)
	}
	return out
}

func attributionSummary(report *PublicReport) string {
	if report == nil || report.ChannelAttribution == nil {
		return ""
	}
	result := report.ChannelAttribution
	switch result.Status {
	case attribution.StatusIdentified:
		return fmt.Sprintf("上游渠道识别为 %s（%.0f%% 置信度）。", result.Channel, result.Confidence*100)
	case attribution.StatusLikely:
		return fmt.Sprintf("上游渠道可能为 %s（%.0f%% 置信度）。", result.Channel, result.Confidence*100)
	case attribution.StatusConflicted:
		return "上游渠道证据存在冲突，当前不做单一来源断言。"
	default:
		if strings.TrimSpace(result.Channel) != "" && result.Channel != "unknown" {
			return fmt.Sprintf("当前仅确认 %s 协议兼容形态，真实上游仍未知。", result.Channel)
		}
		return "真实上游渠道暂时未知。"
	}
}
