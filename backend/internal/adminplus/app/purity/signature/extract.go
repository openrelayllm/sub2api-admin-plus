package signature

import (
	"encoding/json"
	"strings"
)

type JSONAnalysis struct {
	Fingerprints []Fingerprint
	ParseErrors  int
	Found        int
}

func AnalyzeClaudeJSON(body []byte) JSONAnalysis {
	var payload struct {
		Content []struct {
			Type      string `json:"type"`
			Signature string `json:"signature"`
		} `json:"content"`
	}
	if len(body) == 0 || json.Unmarshal(body, &payload) != nil {
		return JSONAnalysis{}
	}
	result := JSONAnalysis{}
	seen := make(map[string]struct{})
	for _, block := range payload.Content {
		if block.Type != "thinking" && block.Type != "redacted_thinking" {
			continue
		}
		encoded := strings.TrimSpace(block.Signature)
		if encoded == "" {
			continue
		}
		result.Found++
		fingerprint, err := Analyze(encoded)
		if err != nil {
			result.ParseErrors++
			continue
		}
		if _, ok := seen[fingerprint.DedupHash]; ok {
			continue
		}
		seen[fingerprint.DedupHash] = struct{}{}
		result.Fingerprints = append(result.Fingerprints, fingerprint)
	}
	return result
}
