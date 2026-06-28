package purity

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

type httpProbe struct {
	StatusCode   int
	Body         []byte
	Headers      map[string]string
	LatencyMS    int64
	ErrorClass   string
	ErrorMessage string
}

func (s *Service) doJSON(ctx context.Context, client *http.Client, method string, endpoint string, apiKey string, body []byte, accept string) httpProbe {
	headers := map[string]string{
		"Accept":        accept,
		"Authorization": "Bearer " + apiKey,
	}
	if body != nil {
		headers["Content-Type"] = "application/json"
	}
	return s.doJSONWithHeaders(ctx, client, method, endpoint, body, headers, apiKey)
}

func (s *Service) probeGatewayRoot(ctx context.Context, client *http.Client, baseURL string) httpProbe {
	return s.doJSONWithHeaders(ctx, client, http.MethodGet, buildGatewayRootURL(baseURL), nil, map[string]string{"Accept": "application/json"}, "")
}

func (s *Service) doJSONWithHeaders(ctx context.Context, client *http.Client, method string, endpoint string, body []byte, headers map[string]string, secret string) httpProbe {
	started := s.currentTime()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return httpProbe{ErrorClass: "request_build_error", ErrorMessage: sanitizeMessage(err.Error(), secret)}
	}
	for key, value := range headers {
		if strings.TrimSpace(key) == "" || value == "" {
			continue
		}
		req.Header.Set(key, value)
	}
	if client == nil {
		client = s.clientForRun(checkRunOptions{})
	}
	resp, err := client.Do(req)
	if err != nil {
		errorClass := errorClassForRequestError(ctx, err)
		return httpProbe{
			LatencyMS:    int64(s.currentTime().Sub(started) / time.Millisecond),
			ErrorClass:   errorClass,
			ErrorMessage: sanitizeMessage(err.Error(), secret),
		}
	}
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, maxProbeBodyBytes))
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxProbeBodyBytes))
	result := httpProbe{
		StatusCode: resp.StatusCode,
		Body:       bodyBytes,
		Headers:    selectedResponseHeaders(resp.Header),
		LatencyMS:  int64(s.currentTime().Sub(started) / time.Millisecond),
	}
	if readErr != nil {
		result.ErrorClass = "read_error"
		result.ErrorMessage = sanitizeMessage(readErr.Error(), secret)
		return result
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errorMessage := upstreamErrorMessage(bodyBytes)
		result.ErrorClass = errorClassForStatusAndMessage(resp.StatusCode, errorMessage)
		result.ErrorMessage = sanitizeMessage(errorMessage, secret)
	}
	return result
}

func errorClassForRequestError(ctx context.Context, err error) string {
	if err == nil {
		return ""
	}
	if ctx != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			if ctxErr == context.DeadlineExceeded {
				return "context_deadline_exceeded"
			}
			return "request_canceled"
		}
	}
	if timeoutErr, ok := err.(net.Error); ok && timeoutErr.Timeout() {
		message := strings.ToLower(err.Error())
		if strings.Contains(message, "response headers") {
			return "response_header_timeout"
		}
		return "network_timeout"
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "timeout awaiting response headers") {
		return "response_header_timeout"
	}
	return "network_error"
}

func selectedResponseHeaders(headers http.Header) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	keys := []string{
		"content-type",
		"access-control-expose-headers",
		"request-id",
		"x-request-id",
		"x-should-retry",
		"x-oneapi-request-id",
		"x-upstream-request-id",
		"x-litellm-call-id",
		"x-litellm-model-id",
		"x-litellm-version",
		"x-codex-beta-features",
		"x-codex-turn-state",
		"x-codex-turn-metadata",
		"x-codex-window-id",
		"x-client-request-id",
		"x-responsesapi-include-timing-metrics",
		"openai-model",
		"x-reasoning-included",
		"x-models-etag",
		"thread-id",
		"session-id",
		"session_id",
		"conversation_id",
		"originator",
		"chatgpt-account-id",
		"openai-beta",
		"x-grok-conv-id",
		"x-idempotency-key",
		"x-goog-api-client",
		"x-ratelimit-limit-requests",
		"x-ratelimit-remaining-requests",
		"anthropic-ratelimit-requests-limit",
		"anthropic-ratelimit-requests-remaining",
		"x-amzn-requestid",
		"x-amzn-trace-id",
		"x-goog-request-id",
		"x-cloud-trace-context",
		"x-kiro-upstream",
		"x-provider",
		"x-upstream",
		"x-upstream-provider",
		"x-relay-provider",
		"x-proxy-provider",
		"x-model-provider",
		"x-request-provider",
		"x-openrouter-provider",
		"x-cpa-version",
		"x-cpa-commit",
		"x-cpa-build-date",
		"x-cpa-support-plugin",
		"x-cpa-home-version",
		"x-cpa-home-build-date",
		"x-server-version",
		"x-server-build-date",
		"x-new-api-version",
		"x-translate-id",
		"auth-version",
		"specific_channel_version",
		"x-accel-buffering",
		"x-frame-options",
		"x-content-type-options",
		"referrer-policy",
		"retry-after",
		"server",
		"via",
		"cf-ray",
	}
	out := make(map[string]string)
	for _, key := range keys {
		if value := strings.TrimSpace(headers.Get(key)); value != "" {
			out[key] = sanitizeMessage(value, "")
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func fingerprintValuesFromHTTPProbes(probes ...httpProbe) []string {
	values := make([]string, 0, len(probes))
	for _, probe := range probes {
		if len(probe.Body) > 0 {
			limit := len(probe.Body)
			if limit > maxFingerprintBytes {
				limit = maxFingerprintBytes
			}
			if value := strings.TrimSpace(string(probe.Body[:limit])); value != "" {
				values = append(values, value)
			}
		}
		if value := strings.TrimSpace(probe.ErrorClass); value != "" {
			values = append(values, value)
		}
		if value := strings.TrimSpace(probe.ErrorMessage); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func channelFromProbe(provider string, host string, officialHost bool, probe httpProbe) string {
	return channelFromHeaders(provider, host, officialHost, probe.Headers)
}

func channelFromStreamProbe(provider string, host string, officialHost bool, headers map[string]string) string {
	return channelFromHeaders(provider, host, officialHost, headers)
}

func channelFromHeaders(provider string, host string, officialHost bool, headers map[string]string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if strings.Contains(host, "bedrock") || headerExists(headers, "x-amzn-requestid", "x-amzn-trace-id") {
		return "aws-bedrock"
	}
	if strings.Contains(host, "googleapis.com") || strings.Contains(host, "vertex") || headerExists(headers, "x-goog-request-id", "x-cloud-trace-context") {
		return "vertex"
	}
	if provider == ProviderAnthropic {
		if officialHost || strings.EqualFold(host, "api.anthropic.com") {
			return "anthropic"
		}
		return "anthropic-compatible"
	}
	if provider == ProviderGemini {
		if officialHost || strings.EqualFold(host, "generativelanguage.googleapis.com") {
			return "gemini"
		}
		return "gemini-compatible"
	}
	if provider == ProviderOpenAI {
		if officialHost || strings.EqualFold(host, "api.openai.com") {
			return "openai"
		}
		return "openai-compatible"
	}
	return "compatible"
}

func headerExists(headers map[string]string, keys ...string) bool {
	for _, key := range keys {
		if strings.TrimSpace(headers[strings.ToLower(key)]) != "" {
			return true
		}
	}
	return false
}

func errorClassForStatus(status int) string {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return "credential_invalid"
	case status == http.StatusTooManyRequests:
		return "upstream_rate_limited"
	case status >= 500:
		return "upstream_5xx"
	case status >= 400:
		return "request_error"
	default:
		return ""
	}
}

func errorClassForStatusAndMessage(status int, message string) string {
	if isAccountBalanceInsufficientMessage(message) {
		return errorClassAccountBalanceInsufficient
	}
	return errorClassForStatus(status)
}

func isAccountBalanceInsufficientMessage(message string) bool {
	value := strings.ToLower(strings.TrimSpace(message))
	if value == "" {
		return false
	}
	hasInsufficient := strings.Contains(value, "insufficient") || strings.Contains(value, "exceeded")
	if !hasInsufficient {
		return false
	}
	return strings.Contains(value, "balance") ||
		strings.Contains(value, "quota") ||
		strings.Contains(value, "credit") ||
		strings.Contains(value, "billing") ||
		strings.Contains(value, "fund")
}

func upstreamErrorMessage(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	for _, path := range []string{"error.message", "message", "response.error.message"} {
		if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); value != "" {
			return value
		}
	}
	return string(body)
}

func sanitizeMessage(message string, apiKey string) string {
	value := strings.TrimSpace(message)
	if value == "" {
		return ""
	}
	if apiKey != "" {
		value = strings.ReplaceAll(value, apiKey, "[redacted]")
	}
	if len(value) > maxErrorMessageLen {
		value = value[:maxErrorMessageLen]
	}
	return value
}
