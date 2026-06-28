package purity

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
)

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(payload))
}
func payloadHasInputImage(payload map[string]any) bool {
	body, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	return strings.Contains(string(body), `"type":"input_image"`)
}
func openAIStoreIncludeProbeRequest(payload map[string]any) bool {
	if store, ok := payload["store"].(bool); !ok || store {
		return false
	}
	if stream, ok := payload["stream"].(bool); !ok || stream {
		return false
	}
	reasoning, _ := payload["reasoning"].(map[string]any)
	if reasoning == nil {
		return false
	}
	if effort, _ := reasoning["effort"].(string); effort != "minimal" {
		return false
	}
	includes, ok := payload["include"].([]any)
	if !ok {
		return false
	}
	for _, item := range includes {
		if value, _ := item.(string); value == "reasoning.encrypted_content" {
			return true
		}
	}
	return false
}
func writeOpenAITextResponse(t *testing.T, w http.ResponseWriter, id string, model string, text string) {
	t.Helper()
	writeJSON(t, w, map[string]any{
		"id":     id,
		"object": "response",
		"model":  model,
		"output": []map[string]any{
			{"type": "message", "role": "assistant", "content": []map[string]any{{"type": "output_text", "text": text}}},
		},
		"usage": map[string]any{
			"input_tokens":  8,
			"output_tokens": 2,
			"total_tokens":  10,
		},
	})
}
func findCheck(t *testing.T, report *PublicReport, id string) CheckResult {
	t.Helper()
	for _, check := range report.Checks {
		if check.ID == id {
			return check
		}
	}
	t.Fatalf("check %s not found", id)
	return CheckResult{}
}
func findValidation(t *testing.T, report *PublicReport, id string) ValidationResult {
	t.Helper()
	for _, validation := range report.Validations {
		if validation.ID == id {
			return validation
		}
	}
	t.Fatalf("validation %s not found", id)
	return ValidationResult{}
}
