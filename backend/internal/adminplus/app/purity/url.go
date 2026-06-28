package purity

import (
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"net/url"
	"strings"
)

func (s *Service) normalizeBaseURL(raw string, allowPrivate bool) (string, string, bool, error) {
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
	parsed.Path = normalizeOpenAIBasePath(parsed.EscapedPath())
	parsed.RawPath = ""

	host := strings.ToLower(parsed.Host)
	officialHost := strings.EqualFold(parsed.Hostname(), "api.openai.com")
	return strings.TrimRight(parsed.String(), "/"), host, officialHost, nil
}

func normalizeOpenAIBasePath(path string) string {
	value := strings.TrimRight(strings.TrimSpace(path), "/")
	if value == "" {
		return ""
	}
	for _, suffix := range []string{"/v1/chat/completions", "/chat/completions", "/v1/responses", "/responses", "/v1/models", "/models"} {
		if strings.HasSuffix(value, suffix) {
			value = strings.TrimRight(strings.TrimSuffix(value, suffix), "/")
			break
		}
	}
	return value
}

func buildGatewayRootURL(base string) string {
	trimmed := strings.TrimSpace(base)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.TrimRight(trimmed, "/") + "/"
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = "/"
	parsed.RawPath = ""
	return parsed.String()
}

func shouldProbeGatewayRoot(host string) bool {
	value := strings.ToLower(strings.TrimSpace(host))
	if value == "" {
		return false
	}
	return !strings.HasPrefix(value, "127.") &&
		!strings.HasPrefix(value, "localhost") &&
		!strings.HasPrefix(value, "[::1]") &&
		!strings.HasPrefix(value, "::1")
}

func buildOpenAIEndpointURL(base string, endpoint string) string {
	normalized := strings.TrimRight(strings.TrimSpace(base), "/")
	endpoint = "/" + strings.TrimLeft(strings.TrimSpace(endpoint), "/")
	relative := strings.TrimPrefix(endpoint, "/v1")
	if strings.HasSuffix(normalized, endpoint) || strings.HasSuffix(normalized, relative) {
		return normalized
	}
	if openAIBaseURLHasVersionSuffix(normalized) {
		return normalized + relative
	}
	return normalized + endpoint
}

func openAIBaseURLHasVersionSuffix(raw string) bool {
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

func isOpenAIAPIVersionSegment(segment string) bool {
	s := strings.ToLower(strings.TrimSpace(segment))
	if len(s) < 2 || s[0] != 'v' || !isASCIIDigit(s[1]) {
		return false
	}
	i := 1
	for i < len(s) && isASCIIDigit(s[i]) {
		i++
	}
	if i == len(s) {
		return true
	}
	if s[i] == '.' {
		i++
		if i == len(s) || !isASCIIDigit(s[i]) {
			return false
		}
		for i < len(s) && isASCIIDigit(s[i]) {
			i++
		}
		return i == len(s)
	}
	suffix := s[i:]
	return strings.HasPrefix(suffix, "alpha") || strings.HasPrefix(suffix, "beta") || strings.HasPrefix(suffix, "preview")
}

func isASCIIDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
