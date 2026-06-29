package purity

import (
	"strings"

	"github.com/tidwall/gjson"
)

type geminiModelSelection struct {
	Model         string
	Requested     string
	FallbackUsed  bool
	FallbackCause string
}

func selectGeminiProbeModel(requested string, modelsProbe httpProbe, generateProbe httpProbe) geminiModelSelection {
	requested = strings.TrimPrefix(strings.TrimSpace(requested), "models/")
	if requested == "" {
		requested = defaultGeminiModel
	}
	selection := geminiModelSelection{Model: requested, Requested: requested}
	if !shouldFallbackGeminiModel(generateProbe) {
		return selection
	}
	candidate := firstAvailableGeminiProbeModel(modelsProbe, requested)
	if candidate == "" || candidate == requested {
		return selection
	}
	selection.Model = candidate
	selection.FallbackUsed = true
	selection.FallbackCause = "requested_model_unavailable"
	return selection
}

func shouldFallbackGeminiModel(probe httpProbe) bool {
	if probe.StatusCode == 0 || probe.StatusCode == 401 || probe.StatusCode == 403 || probe.ErrorClass == errorClassAccountBalanceInsufficient {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(firstNonEmptyString(probe.ErrorMessage, upstreamErrorMessage(probe.Body))))
	if message == "" {
		return probe.StatusCode == 404
	}
	if strings.Contains(message, "not found") ||
		strings.Contains(message, "not supported for generatecontent") ||
		strings.Contains(message, "model not found") ||
		strings.Contains(message, "model_not_found") ||
		strings.Contains(message, "does not exist") ||
		strings.Contains(message, "unknown model") ||
		strings.Contains(message, "invalid model") ||
		strings.Contains(message, "no such model") ||
		strings.Contains(message, "no available channel") ||
		strings.Contains(message, "failed to get available channel") {
		return true
	}
	return probe.StatusCode == 404
}

func firstAvailableGeminiProbeModel(modelsProbe httpProbe, requested string) string {
	available := geminiAvailableModelSet(modelsProbe)
	candidates := []string{requested}
	for _, model := range geminiPreferredProbeModels {
		candidates = appendUniqueString(candidates, model)
	}
	for _, candidate := range candidates {
		normalized := strings.TrimPrefix(strings.TrimSpace(candidate), "models/")
		if normalized == "" || normalized == requested {
			continue
		}
		if len(available) == 0 || available[normalized] {
			return normalized
		}
	}
	return ""
}

func geminiAvailableModelSet(probe httpProbe) map[string]bool {
	models := gjson.GetBytes(probe.Body, "models")
	if !models.IsArray() {
		return nil
	}
	out := make(map[string]bool)
	for _, item := range models.Array() {
		name := strings.TrimPrefix(strings.TrimSpace(item.Get("name").String()), "models/")
		if name == "" {
			continue
		}
		if methods := item.Get("supportedGenerationMethods"); methods.IsArray() {
			supportsGenerate := false
			for _, method := range methods.Array() {
				if method.String() == "generateContent" {
					supportsGenerate = true
					break
				}
			}
			if !supportsGenerate {
				continue
			}
		}
		out[name] = true
	}
	return out
}
