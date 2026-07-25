package purity

import (
	"bufio"
	"bytes"
	"context"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/signature"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"io"
	"net/http"
	"strings"
	"time"
)

func (s *Service) probeClaudeMessages(ctx context.Context, client *http.Client, baseURL string, apiKey string, model string, probeCtx claudeProbeContext) httpProbe {
	return s.doJSONWithHeaders(ctx, client, http.MethodPost, buildOpenAIEndpointURL(baseURL, "/v1/messages"), claudeToolProbePayload(model, probeCtx), claudeHeaders("application/json", apiKey, probeCtx), apiKey)
}

func (s *Service) probeClaudeMultimodal(ctx context.Context, client *http.Client, baseURL string, apiKey string, model string, probeCtx claudeProbeContext) httpProbe {
	return s.doJSONWithHeaders(ctx, client, http.MethodPost, buildOpenAIEndpointURL(baseURL, "/v1/messages"), claudeMultimodalProbePayload(model, probeCtx), claudeHeaders("application/json", apiKey, probeCtx), apiKey)
}

func (s *Service) probeClaudeAudit(ctx context.Context, client *http.Client, baseURL string, apiKey string, model string, round int, auditNonce string, probeCtx claudeProbeContext, history []claudeAuditTurn) httpProbe {
	return s.doJSONWithHeaders(ctx, client, http.MethodPost, buildOpenAIEndpointURL(baseURL, "/v1/messages"), claudeAuditProbePayload(model, round, auditNonce, probeCtx, history), claudeHeaders("application/json", apiKey, probeCtx), apiKey)
}

func (s *Service) probeClaudeInvalidThinkingSignature(ctx context.Context, client *http.Client, baseURL string, apiKey string, model string, probeCtx claudeProbeContext) httpProbe {
	return s.doJSONWithHeaders(ctx, client, http.MethodPost, buildOpenAIEndpointURL(baseURL, "/v1/messages"), claudeInvalidThinkingSignaturePayload(model, probeCtx), claudeHeaders("application/json", apiKey, probeCtx), apiKey)
}

func (s *Service) probeClaudeThinking(ctx context.Context, client *http.Client, baseURL string, apiKey string, model string, probeCtx claudeProbeContext) httpProbe {
	return s.doJSONWithHeaders(ctx, client, http.MethodPost, buildOpenAIEndpointURL(baseURL, "/v1/messages"), claudeThinkingProbePayload(model, probeCtx, false), claudeHeaders("application/json", apiKey, probeCtx), apiKey)
}

func (s *Service) probeClaudeThinkingBudgetViolation(ctx context.Context, client *http.Client, baseURL string, apiKey string, model string, probeCtx claudeProbeContext) httpProbe {
	return s.doJSONWithHeaders(ctx, client, http.MethodPost, buildOpenAIEndpointURL(baseURL, "/v1/messages"), claudeThinkingBudgetViolationPayload(model, probeCtx), claudeHeaders("application/json", apiKey, probeCtx), apiKey)
}

func (s *Service) probeClaudeCacheControlOverflow(ctx context.Context, client *http.Client, baseURL string, apiKey string, model string, probeCtx claudeProbeContext) httpProbe {
	return s.doJSONWithHeaders(ctx, client, http.MethodPost, buildOpenAIEndpointURL(baseURL, "/v1/messages"), claudeCacheControlOverflowPayload(model, probeCtx), claudeHeaders("application/json", apiKey, probeCtx), apiKey)
}

func claudeHeaders(accept string, apiKey string, probeCtx claudeProbeContext) map[string]string {
	headers := map[string]string{
		"User-Agent":                                claudeCodeProbeUserAgent,
		"X-Stainless-Lang":                          "js",
		"X-Stainless-Package-Version":               "0.94.0",
		"X-Stainless-OS":                            "Linux",
		"X-Stainless-Arch":                          "arm64",
		"X-Stainless-Runtime":                       "node",
		"X-Stainless-Runtime-Version":               "v24.3.0",
		"X-Stainless-Retry-Count":                   "0",
		"X-Stainless-Timeout":                       "600",
		"X-App":                                     "cli",
		"Anthropic-Dangerous-Direct-Browser-Access": "true",
		"X-Claude-Code-Session-Id":                  probeCtx.sessionID,
		"x-client-request-id":                       uuid.NewString(),
		"Accept":                                    accept,
		"Content-Type":                              "application/json",
		"x-api-key":                                 apiKey,
		"anthropic-version":                         anthropicVersion,
		"anthropic-beta":                            claudeAPIKeyBetaHeader,
	}
	return headers
}

type claudeStreamProbe struct {
	StatusCode             int
	Headers                map[string]string
	FirstTokenMS           int64
	TotalLatencyMS         int64
	SeenData               bool
	SeenMessageStart       bool
	SeenContentBlockStart  bool
	SeenDelta              bool
	SeenMessageDelta       bool
	SeenMessageStop        bool
	SeenToolUse            bool
	SeenThinkingDelta      bool
	SeenSignatureDelta     bool
	SignatureAfterThinking bool
	SignatureFound         int
	SignatureParseErrors   int
	SignatureFingerprints  []signature.Fingerprint
	ErrorClass             string
	ErrorMessage           string
}

func (s *Service) probeClaudeStream(ctx context.Context, client *http.Client, baseURL string, apiKey string, model string, probeCtx claudeProbeContext) claudeStreamProbe {
	return s.probeClaudeStreamWithPayload(ctx, client, baseURL, apiKey, claudeStreamProbePayload(model, probeCtx), probeCtx)
}

func (s *Service) probeClaudeThinkingStream(ctx context.Context, client *http.Client, baseURL string, apiKey string, model string, probeCtx claudeProbeContext) claudeStreamProbe {
	return s.probeClaudeStreamWithPayload(ctx, client, baseURL, apiKey, claudeThinkingProbePayload(model, probeCtx, true), probeCtx)
}

func (s *Service) probeClaudeStreamWithPayload(ctx context.Context, client *http.Client, baseURL string, apiKey string, body []byte, probeCtx claudeProbeContext) claudeStreamProbe {
	started := s.currentTime()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildOpenAIEndpointURL(baseURL, "/v1/messages"), bytes.NewReader(body))
	if err != nil {
		return claudeStreamProbe{ErrorClass: "request_build_error", ErrorMessage: sanitizeMessage(err.Error(), apiKey)}
	}
	for key, value := range claudeHeaders("text/event-stream", apiKey, probeCtx) {
		req.Header.Set(key, value)
	}
	if client == nil {
		client = s.clientForRun(checkRunOptions{})
	}
	resp, err := client.Do(req)
	if err != nil {
		return claudeStreamProbe{
			TotalLatencyMS: int64(s.currentTime().Sub(started) / time.Millisecond),
			ErrorClass:     "network_error",
			ErrorMessage:   sanitizeMessage(err.Error(), apiKey),
		}
	}
	defer func() { _ = resp.Body.Close() }()

	result := claudeStreamProbe{StatusCode: resp.StatusCode, Headers: selectedResponseHeaders(resp.Header)}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxProbeBodyBytes))
		result.TotalLatencyMS = int64(s.currentTime().Sub(started) / time.Millisecond)
		errorMessage := upstreamErrorMessage(bodyBytes)
		result.ErrorClass = errorClassForStatusAndMessage(resp.StatusCode, errorMessage)
		result.ErrorMessage = sanitizeMessage(errorMessage, apiKey)
		return result
	}
	readClaudeStream(resp.Body, started, s.currentTime, &result, apiKey)
	result.TotalLatencyMS = int64(s.currentTime().Sub(started) / time.Millisecond)
	return result
}

func readClaudeStream(body io.Reader, started time.Time, now func() time.Time, result *claudeStreamProbe, apiKey string) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	signatureParts := make(map[int]*strings.Builder)
	finishedSignatures := make(map[int]struct{})
	finishSignature := func(index int) {
		if _, done := finishedSignatures[index]; done {
			return
		}
		builder := signatureParts[index]
		if builder == nil || builder.Len() == 0 {
			return
		}
		finishedSignatures[index] = struct{}{}
		result.SignatureFound++
		fingerprint, err := signature.Analyze(builder.String())
		if err != nil {
			result.SignatureParseErrors++
			return
		}
		for _, existing := range result.SignatureFingerprints {
			if existing.DedupHash == fingerprint.DedupHash {
				return
			}
		}
		result.SignatureFingerprints = append(result.SignatureFingerprints, fingerprint)
	}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		result.SeenData = true
		eventType := strings.TrimSpace(gjson.Get(data, "type").String())
		blockIndex := int(gjson.Get(data, "index").Int())
		switch eventType {
		case "message_start":
			result.SeenMessageStart = true
		case "content_block_start":
			result.SeenContentBlockStart = true
			if gjson.Get(data, "content_block.type").String() == "tool_use" {
				result.SeenToolUse = true
			}
		case "content_block_delta":
			deltaType := gjson.Get(data, "delta.type").String()
			if deltaType == "text_delta" || deltaType == "input_json_delta" || deltaType == "thinking_delta" {
				result.SeenDelta = true
				if deltaType == "thinking_delta" {
					result.SeenThinkingDelta = true
				}
				if result.FirstTokenMS == 0 {
					result.FirstTokenMS = int64(now().Sub(started) / time.Millisecond)
				}
			}
			if deltaType == "signature_delta" {
				result.SeenSignatureDelta = true
				result.SignatureAfterThinking = result.SignatureAfterThinking || result.SeenThinkingDelta
				part := gjson.Get(data, "delta.signature").String()
				if part != "" {
					builder := signatureParts[blockIndex]
					if builder == nil {
						builder = &strings.Builder{}
						signatureParts[blockIndex] = builder
					}
					if builder.Len()+len(part) <= 128*1024 {
						_, _ = builder.WriteString(part)
					} else {
						result.SignatureParseErrors++
						finishedSignatures[blockIndex] = struct{}{}
					}
				}
			}
		case "content_block_stop":
			finishSignature(blockIndex)
		case "message_delta":
			result.SeenMessageDelta = true
		case "message_stop":
			result.SeenMessageStop = true
		case "error":
			result.ErrorClass = "response_failed"
			result.ErrorMessage = sanitizeMessage(upstreamErrorMessage([]byte(data)), apiKey)
		}
	}
	if err := scanner.Err(); err != nil {
		result.ErrorClass = "stream_error"
		result.ErrorMessage = sanitizeMessage(err.Error(), apiKey)
	}
	for index := range signatureParts {
		finishSignature(index)
	}
}
