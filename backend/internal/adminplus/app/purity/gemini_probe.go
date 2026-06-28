package purity

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

const defaultGeminiModel = "gemini-3-pro-preview"

func (s *Service) probeGeminiModels(ctx context.Context, client *http.Client, baseURL string, apiKey string) httpProbe {
	return s.doGeminiJSON(ctx, client, http.MethodGet, buildGeminiModelsURL(baseURL), apiKey, nil, "application/json")
}

func (s *Service) probeGeminiGenerateContent(ctx context.Context, client *http.Client, baseURL string, apiKey string, model string) httpProbe {
	return s.doGeminiJSON(ctx, client, http.MethodPost, buildGeminiGenerateURL(baseURL, model, "generateContent", false), apiKey, geminiToolProbePayload(), "application/json")
}

func (s *Service) probeGeminiMultimodal(ctx context.Context, client *http.Client, baseURL string, apiKey string, model string) httpProbe {
	return s.doGeminiJSON(ctx, client, http.MethodPost, buildGeminiGenerateURL(baseURL, model, "generateContent", false), apiKey, geminiMultimodalProbePayload(), "application/json")
}

func (s *Service) doGeminiJSON(ctx context.Context, client *http.Client, method string, endpoint string, apiKey string, body []byte, accept string) httpProbe {
	headers := map[string]string{
		"Accept":         accept,
		"x-goog-api-key": apiKey,
	}
	if body != nil {
		headers["Content-Type"] = "application/json"
	}
	return s.doJSONWithHeaders(ctx, client, method, endpoint, body, headers, apiKey)
}

func (s *Service) probeGeminiStream(ctx context.Context, client *http.Client, baseURL string, apiKey string, model string) streamProbe {
	started := s.currentTime()
	body := geminiTextProbePayload("Return exactly: ok")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildGeminiGenerateURL(baseURL, model, "streamGenerateContent", true), bytes.NewReader(body))
	if err != nil {
		return streamProbe{ErrorClass: "request_build_error", ErrorMessage: sanitizeMessage(err.Error(), apiKey)}
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)
	if client == nil {
		client = s.clientForRun(checkRunOptions{})
	}
	resp, err := client.Do(req)
	if err != nil {
		return streamProbe{
			TotalLatencyMS: int64(s.currentTime().Sub(started) / time.Millisecond),
			ErrorClass:     "network_error",
			ErrorMessage:   sanitizeMessage(err.Error(), apiKey),
		}
	}
	defer func() { _ = resp.Body.Close() }()

	result := streamProbe{StatusCode: resp.StatusCode, Headers: selectedResponseHeaders(resp.Header)}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxProbeBodyBytes))
		result.TotalLatencyMS = int64(s.currentTime().Sub(started) / time.Millisecond)
		errorMessage := upstreamErrorMessage(bodyBytes)
		result.ErrorClass = errorClassForStatusAndMessage(resp.StatusCode, errorMessage)
		result.ErrorMessage = sanitizeMessage(errorMessage, apiKey)
		return result
	}
	readGeminiStream(resp.Body, started, s.currentTime, &result, apiKey)
	result.TotalLatencyMS = int64(s.currentTime().Sub(started) / time.Millisecond)
	return result
}

func readGeminiStream(body io.Reader, started time.Time, now func() time.Time, result *streamProbe, apiKey string) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		result.SeenData = true
		if data == "[DONE]" {
			result.SeenDone = true
			continue
		}
		if strings.TrimSpace(gjson.Get(data, "candidates.0.content.parts.0.text").String()) != "" {
			result.SeenDelta = true
			if result.FirstTokenMS == 0 {
				result.FirstTokenMS = int64(now().Sub(started) / time.Millisecond)
			}
		}
		if strings.TrimSpace(gjson.Get(data, "candidates.0.finishReason").String()) != "" || gjson.Get(data, "usageMetadata").Exists() {
			result.SeenCompleted = true
		}
		if gjson.Get(data, "error").Exists() {
			result.ErrorClass = "response_failed"
			result.ErrorMessage = sanitizeMessage(upstreamErrorMessage([]byte(data)), apiKey)
		}
	}
	if err := scanner.Err(); err != nil {
		result.ErrorClass = "stream_error"
		result.ErrorMessage = sanitizeMessage(err.Error(), apiKey)
	}
}

func geminiToolProbePayload() []byte {
	body, _ := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]any{
					{"text": "Call the probe_ping function with ok=true to acknowledge readiness. You must use the tool."},
				},
			},
		},
		"tools": []map[string]any{
			{
				"functionDeclarations": []map[string]any{
					{
						"name":        "probe_ping",
						"description": "Capability probe. Call to acknowledge.",
						"parameters": map[string]any{
							"type": "OBJECT",
							"properties": map[string]any{
								"ok": map[string]any{"type": "BOOLEAN"},
							},
							"required": []string{"ok"},
						},
					},
				},
			},
		},
		"toolConfig": map[string]any{
			"functionCallingConfig": map[string]any{
				"mode":                 "ANY",
				"allowedFunctionNames": []string{"probe_ping"},
			},
		},
		"generationConfig": map[string]any{
			"maxOutputTokens": 128,
		},
	})
	return body
}

func geminiTextProbePayload(prompt string) []byte {
	body, _ := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]any{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]any{
			"maxOutputTokens": 16,
		},
	})
	return body
}

func geminiMultimodalProbePayload() []byte {
	body, _ := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]any{
					{"text": "Read the attached image and return exactly: ok"},
					{"inlineData": map[string]any{"mimeType": "image/png", "data": probePNGBase64}},
				},
			},
		},
		"generationConfig": map[string]any{
			"maxOutputTokens": 16,
		},
	})
	return body
}

func parseGeminiUsage(body []byte) *TokenUsage {
	usage := gjson.GetBytes(body, "usageMetadata")
	if !usage.Exists() || !usage.IsObject() {
		return nil
	}
	input := usage.Get("promptTokenCount").Int()
	output := usage.Get("candidatesTokenCount").Int()
	total := usage.Get("totalTokenCount").Int()
	cached := usage.Get("cachedContentTokenCount").Int()
	thoughts := usage.Get("thoughtsTokenCount").Int()
	toolUse := usage.Get("toolUsePromptTokenCount").Int()
	return &TokenUsage{
		InputTokens:              input,
		OutputTokens:             output,
		TotalTokens:              total,
		CachedTokens:             cached,
		CachedTokensFieldPresent: usagePathExists(usage, "cachedContentTokenCount"),
		ReasoningTokens:          thoughts + toolUse,
	}
}
