package purity

import (
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"net/url"
	"strings"
)

func (s *Service) normalizeAnthropicBaseURL(raw string, allowPrivate bool) (string, string, bool, error) {
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
	parsed.Path = normalizeAnthropicBasePath(parsed.EscapedPath())
	parsed.RawPath = ""

	host := strings.ToLower(parsed.Host)
	officialHost := strings.EqualFold(parsed.Hostname(), "api.anthropic.com")
	return strings.TrimRight(parsed.String(), "/"), host, officialHost, nil
}

func normalizeAnthropicBasePath(path string) string {
	value := strings.TrimRight(strings.TrimSpace(path), "/")
	if value == "" {
		return ""
	}
	for _, suffix := range []string{"/v1/messages/count_tokens", "/messages/count_tokens", "/v1/messages", "/messages", "/v1/models", "/models"} {
		if strings.HasSuffix(value, suffix) {
			value = strings.TrimRight(strings.TrimSuffix(value, suffix), "/")
			break
		}
	}
	return value
}
