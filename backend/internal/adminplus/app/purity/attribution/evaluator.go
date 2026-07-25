package attribution

import (
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/signature"
)

const DetectorVersion = "channel-attribution/2026-07-25.1"

const (
	StatusIdentified = "identified"
	StatusLikely     = "likely"
	StatusUnknown    = "unknown"
	StatusConflicted = "conflicted"

	StrengthStrong = "strong"
	StrengthMedium = "medium"
	StrengthWeak   = "weak"
)

type Evidence struct {
	Kind            string   `json:"kind"`
	Code            string   `json:"code"`
	Channel         string   `json:"channel,omitempty"`
	Strength        string   `json:"strength"`
	SourceType      string   `json:"source_type"`
	Summary         string   `json:"summary"`
	SampleCount     int      `json:"sample_count,omitempty"`
	Transport       string   `json:"transport,omitempty"`
	DetectorVersion string   `json:"detector_version"`
	ObservedAt      string   `json:"observed_at,omitempty"`
	Limitations     []string `json:"limitations,omitempty"`
	sourceGroup     string
	confidence      float64
}

type Result struct {
	Channel         string     `json:"channel"`
	Confidence      float64    `json:"confidence"`
	Status          string     `json:"status"`
	Evidence        []Evidence `json:"evidence,omitempty"`
	Contradictions  []Evidence `json:"contradictions,omitempty"`
	Limitations     []string   `json:"limitations,omitempty"`
	ReasonCodes     []string   `json:"reason_codes,omitempty"`
	DetectorVersion string     `json:"detector_version"`
}

type SignatureObservation struct {
	Transport      string
	Classification signature.Classification
}

type Input struct {
	Provider          string
	Host              string
	Model             string
	CheckedAt         time.Time
	HeaderSets        []map[string]string
	WrapperSignals    []string
	StreamChannel     string
	NonStreamChannel  string
	Signatures        []SignatureObservation
	SignatureFound    int
	SignatureFailures int
}

func Evaluate(input Input) Result {
	evidence, limitations := collectEvidence(input)
	result := Result{
		Channel:         fallbackChannel(input.Provider),
		Status:          StatusUnknown,
		Evidence:        []Evidence{},
		Contradictions:  []Evidence{},
		Limitations:     limitations,
		ReasonCodes:     []string{"channel_evidence_insufficient"},
		DetectorVersion: DetectorVersion,
	}

	candidates := candidateEvidence(evidence)
	if len(candidates) == 0 {
		result.Evidence = publicEvidence(evidence)
		return result
	}

	topChannels := materialChannels(candidates)
	if len(topChannels) > 1 {
		result.Channel = "unknown"
		result.Status = StatusConflicted
		result.Confidence = 0
		result.ReasonCodes = []string{"channel_evidence_conflict"}
		result.Evidence, result.Contradictions = splitConflictEvidence(candidates, topChannels[0])
		return result
	}

	channel := topChannels[0]
	supporting := evidenceForChannel(candidates, channel)
	result.Channel = channel
	result.Evidence = publicEvidence(supporting)
	result.ReasonCodes = []string{"channel_evidence_matched"}
	if hasStrength(supporting, StrengthStrong) {
		result.Status = StatusIdentified
		result.Confidence = maxConfidence(supporting, 0.9)
		return result
	}
	result.Status = StatusLikely
	result.Confidence = mediumConfidence(supporting)
	return result
}

func collectEvidence(input Input) ([]Evidence, []string) {
	observedAt := ""
	if !input.CheckedAt.IsZero() {
		observedAt = input.CheckedAt.UTC().Format(time.RFC3339)
	}
	evidence := make([]Evidence, 0, 12)
	limitations := make([]string, 0, 4)
	appendEvidence := func(item Evidence) {
		item.DetectorVersion = DetectorVersion
		item.ObservedAt = observedAt
		evidence = append(evidence, item)
	}

	for _, observation := range input.Signatures {
		classification := observation.Classification
		if classification.Channel == "" || classification.Status == signature.ClassificationUnknown {
			limitations = appendUnique(limitations, "signature_structure_not_in_calibrated_registry")
			continue
		}
		strength := StrengthMedium
		if classification.Status == signature.ClassificationIdentified {
			strength = StrengthStrong
		}
		appendEvidence(Evidence{
			Kind:        "signature_structure",
			Code:        "signature_family_match",
			Channel:     classification.Channel,
			Strength:    strength,
			SourceType:  firstNonEmpty(classification.SourceType, "authorized_observation"),
			Summary:     "Thinking signature 与已校准的脱敏结构族一致。",
			SampleCount: classification.SampleCount,
			Transport:   observation.Transport,
			Limitations: append([]string(nil), classification.Limitations...),
			sourceGroup: "signature_" + observation.Transport,
			confidence:  classification.Confidence,
		})
		for _, limitation := range classification.Limitations {
			limitations = appendUnique(limitations, limitation)
		}
	}
	if input.SignatureFound > 0 && len(input.Signatures) == 0 {
		limitations = appendUnique(limitations, "signature_structure_not_in_calibrated_registry")
	}
	if input.SignatureFailures > 0 {
		limitations = appendUnique(limitations, "one_or_more_signatures_could_not_be_safely_parsed")
	}

	host := strings.ToLower(strings.TrimSpace(input.Host))
	switch {
	case hostNameEquals(host, "api.anthropic.com"):
		appendEvidence(endpointEvidence("anthropic_native", "official_anthropic_endpoint", observedAt))
	case hostNameEquals(host, "api.openai.com"):
		appendEvidence(endpointEvidence("openai_native", "official_openai_endpoint", observedAt))
	case hostNameEquals(host, "generativelanguage.googleapis.com"):
		appendEvidence(endpointEvidence("google_ai_studio", "official_google_ai_endpoint", observedAt))
	case strings.Contains(host, "bedrock") && strings.Contains(host, "amazonaws.com"):
		appendEvidence(endpointEvidence("aws_bedrock", "official_aws_bedrock_endpoint", observedAt))
	case strings.Contains(host, "aiplatform.googleapis.com") || strings.Contains(host, "vertex"):
		appendEvidence(endpointEvidence("google_vertex", "official_google_vertex_endpoint", observedAt))
	}

	headers := mergeHeaders(input.HeaderSets)
	if headerPresent(headers, "x-amzn-requestid", "x-amzn-trace-id") {
		appendEvidence(Evidence{
			Kind:        "provider_header",
			Code:        "aws_response_headers",
			Channel:     "aws_bedrock",
			Strength:    StrengthMedium,
			SourceType:  "authorized_observation",
			Summary:     "响应包含 AWS 请求链路头。",
			sourceGroup: "provider_headers",
			confidence:  0.72,
		})
	}
	if headerPresent(headers, "x-goog-request-id", "x-cloud-trace-context") {
		appendEvidence(Evidence{
			Kind:        "provider_header",
			Code:        "google_response_headers",
			Channel:     "google_vertex",
			Strength:    StrengthMedium,
			SourceType:  "authorized_observation",
			Summary:     "响应包含 Google Cloud 请求链路头。",
			sourceGroup: "provider_headers",
			confidence:  0.72,
		})
	}

	for _, channel := range []string{input.StreamChannel, input.NonStreamChannel} {
		if normalized := normalizeLegacyChannel(channel); normalized != "" {
			appendEvidence(Evidence{
				Kind:        "protocol_channel",
				Code:        "protocol_channel_signal",
				Channel:     normalized,
				Strength:    StrengthWeak,
				SourceType:  "engineering_inference",
				Summary:     "协议响应形态提供了弱渠道线索。",
				sourceGroup: "legacy_protocol_channel",
				confidence:  0.35,
			})
		}
	}
	for _, signal := range input.WrapperSignals {
		channel := channelFromSignal(signal)
		if channel == "" {
			continue
		}
		appendEvidence(Evidence{
			Kind:        "channel_signal",
			Code:        "channel_detector_signal",
			Channel:     channel,
			Strength:    StrengthMedium,
			SourceType:  "authorized_observation",
			Summary:     "响应特征与渠道 detector 信号一致。",
			sourceGroup: "channel_detector",
			confidence:  0.66,
		})
	}
	return dedupeEvidence(evidence), limitations
}

func endpointEvidence(channel string, code string, _ string) Evidence {
	return Evidence{
		Kind:        "endpoint_contract",
		Code:        code,
		Channel:     channel,
		Strength:    StrengthStrong,
		SourceType:  "official_contract",
		Summary:     "请求目标为厂商公开的官方 API 端点。",
		sourceGroup: "endpoint",
		confidence:  0.99,
	}
}

func candidateEvidence(evidence []Evidence) []Evidence {
	out := make([]Evidence, 0, len(evidence))
	for _, item := range evidence {
		if item.Channel == "" || strengthRank(item.Strength) < strengthRank(StrengthMedium) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func materialChannels(evidence []Evidence) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, 2)
	for _, item := range evidence {
		if _, ok := seen[item.Channel]; ok {
			continue
		}
		seen[item.Channel] = struct{}{}
		out = append(out, item.Channel)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return channelEvidenceRank(evidence, out[i]) > channelEvidenceRank(evidence, out[j])
	})
	return out
}

func channelEvidenceRank(evidence []Evidence, channel string) int {
	best := 0
	for _, item := range evidence {
		if item.Channel == channel && strengthRank(item.Strength) > best {
			best = strengthRank(item.Strength)
		}
	}
	return best
}

func splitConflictEvidence(evidence []Evidence, primary string) ([]Evidence, []Evidence) {
	supporting := make([]Evidence, 0, len(evidence))
	contradictions := make([]Evidence, 0, len(evidence))
	for _, item := range evidence {
		if item.Channel == primary {
			supporting = append(supporting, item)
		} else {
			contradictions = append(contradictions, item)
		}
	}
	return publicEvidence(supporting), publicEvidence(contradictions)
}

func evidenceForChannel(evidence []Evidence, channel string) []Evidence {
	out := make([]Evidence, 0, len(evidence))
	for _, item := range evidence {
		if item.Channel == channel {
			out = append(out, item)
		}
	}
	return out
}

func hasStrength(evidence []Evidence, strength string) bool {
	for _, item := range evidence {
		if item.Strength == strength {
			return true
		}
	}
	return false
}

func maxConfidence(evidence []Evidence, fallback float64) float64 {
	value := fallback
	for _, item := range evidence {
		if item.confidence > value {
			value = item.confidence
		}
	}
	if value > 0.99 {
		return 0.99
	}
	return value
}

func mediumConfidence(evidence []Evidence) float64 {
	groups := make(map[string]struct{})
	value := 0.62
	for _, item := range evidence {
		groups[item.sourceGroup] = struct{}{}
		if item.confidence > value {
			value = item.confidence
		}
	}
	if len(groups) >= 2 {
		value += 0.1
	}
	if value > 0.84 {
		value = 0.84
	}
	return value
}

func publicEvidence(evidence []Evidence) []Evidence {
	out := make([]Evidence, len(evidence))
	copy(out, evidence)
	for index := range out {
		out[index].sourceGroup = ""
		out[index].confidence = 0
	}
	return out
}

func dedupeEvidence(evidence []Evidence) []Evidence {
	seen := make(map[string]struct{}, len(evidence))
	out := make([]Evidence, 0, len(evidence))
	for _, item := range evidence {
		key := strings.Join([]string{item.Channel, item.Code, item.sourceGroup, item.Transport}, "\x00")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func mergeHeaders(headerSets []map[string]string) map[string]string {
	out := make(map[string]string)
	for _, headers := range headerSets {
		for key, value := range headers {
			key = strings.ToLower(strings.TrimSpace(key))
			value = strings.TrimSpace(value)
			if key != "" && value != "" {
				out[key] = value
			}
		}
	}
	return out
}

func headerPresent(headers map[string]string, keys ...string) bool {
	for _, key := range keys {
		if strings.TrimSpace(headers[strings.ToLower(key)]) != "" {
			return true
		}
	}
	return false
}

func hostNameEquals(host string, expected string) bool {
	host = strings.TrimSpace(strings.Split(host, ":")[0])
	return strings.EqualFold(host, expected)
}

func normalizeLegacyChannel(channel string) string {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "anthropic":
		return "anthropic_native"
	case "aws-bedrock", "bedrock":
		return "aws_bedrock"
	case "vertex":
		return "google_vertex"
	case "openai":
		return "openai_native"
	case "gemini":
		return "google_ai_studio"
	default:
		return ""
	}
}

func channelFromSignal(signal string) string {
	switch strings.ToLower(strings.TrimSpace(signal)) {
	case "bedrock":
		return "aws_bedrock"
	case "vertex":
		return "google_vertex"
	case "kiro":
		return "kiro"
	case "antigravity":
		return "antigravity"
	default:
		return ""
	}
}

func fallbackChannel(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "anthropic":
		return "anthropic_compatible"
	case "openai":
		return "openai_compatible"
	case "gemini":
		return "gemini_compatible"
	default:
		return "unknown"
	}
}

func strengthRank(strength string) int {
	switch strength {
	case StrengthStrong:
		return 3
	case StrengthMedium:
		return 2
	case StrengthWeak:
		return 1
	default:
		return 0
	}
}

func appendUnique(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
