package purity

import (
	"net/url"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
)

const defaultGeminiBaseURL = "https://generativelanguage.googleapis.com"

func (s *Service) normalizeGeminiBaseURL(raw string, allowPrivate bool) (string, string, bool, error) {
	if strings.TrimSpace(raw) == "" {
		raw = defaultGeminiBaseURL
	}
	normalized, err := urlvalidator.ValidateHTTPURL(raw, true, urlvalidator.ValidationOptions{
		AllowPrivate: allowPrivate,
	})
	if err != nil {
		return "", "", false, infraerrors.BadRequest("PURITY_BASE_URL_INVALID", "invalid api base url")
	}
	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", "", false, infraerrors.BadRequest("PURITY_BASE_URL_INVALID", "invalid api base url")
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = normalizeGeminiBasePath(parsed.EscapedPath())
	parsed.RawPath = ""

	host := strings.ToLower(parsed.Host)
	officialHost := strings.EqualFold(parsed.Hostname(), "generativelanguage.googleapis.com")
	return strings.TrimRight(parsed.String(), "/"), host, officialHost, nil
}

func normalizeGeminiBasePath(path string) string {
	value := strings.TrimRight(strings.TrimSpace(path), "/")
	if value == "" {
		return ""
	}
	for _, marker := range []string{"/v1beta/models", "/v1/models", "/models"} {
		if idx := strings.Index(value, marker); idx >= 0 {
			return strings.TrimRight(value[:idx], "/")
		}
	}
	return value
}

func buildGeminiModelsURL(base string) string {
	normalized := strings.TrimRight(strings.TrimSpace(base), "/")
	if geminiBaseURLHasVersionSuffix(normalized) {
		return normalized + "/models"
	}
	return normalized + "/v1beta/models"
}

func buildGeminiGenerateURL(base string, model string, action string, stream bool) string {
	normalized := strings.TrimRight(strings.TrimSpace(base), "/")
	model = strings.TrimPrefix(strings.TrimSpace(model), "models/")
	if model == "" {
		model = defaultGeminiModel
	}
	action = strings.TrimSpace(action)
	if action == "" {
		action = "generateContent"
	}
	prefix := "/v1beta/models/"
	if geminiBaseURLHasVersionSuffix(normalized) {
		prefix = "/models/"
	}
	out := normalized + prefix + url.PathEscape(model) + ":" + action
	if stream {
		out += "?alt=sse"
	}
	return out
}

func geminiBaseURLHasVersionSuffix(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return false
	}
	pathValue := ""
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		pathValue = parsed.Path
	} else if slash := strings.Index(trimmed, "/"); slash >= 0 {
		pathValue = trimmed[slash:]
	}
	pathValue = strings.TrimRight(pathValue, "/")
	if pathValue == "" {
		return false
	}
	lastSlash := strings.LastIndex(pathValue, "/")
	segment := pathValue
	if lastSlash >= 0 {
		segment = pathValue[lastSlash+1:]
	}
	return isOpenAIAPIVersionSegment(segment)
}
