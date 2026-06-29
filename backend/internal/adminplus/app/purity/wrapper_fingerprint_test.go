package purity

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestWrapperFingerprintSignals_CoversCommonProxyProviders(t *testing.T) {
	signals := wrapperFingerprintSignals("https://api.proxyai.best", map[string]string{
		"x-cpa-support-plugin": "true",
		"x-relay-provider":     "antigravity",
		"x-provider":           "openai-compatible-kimi",
		"x-upstream-provider":  "xai-grok",
		"x-model-provider":     "gemini-aistudio-codex",
	})
	require.Contains(t, signals, "sub2api")
	require.Contains(t, signals, "cliproxyapi")
	require.Contains(t, signals, "antigravity")
	require.Contains(t, signals, "openai-compatible")
	require.Contains(t, signals, "kimi")
	require.Contains(t, signals, "xai")
	require.Contains(t, signals, "gemini")
	require.Contains(t, signals, "aistudio")
	require.Contains(t, signals, "codex")
}
func TestWrapperFingerprintSignals_CoversSub2APIBlackBoxFingerprints(t *testing.T) {
	report := &PublicReport{
		APIBaseHost:   "relay.example.com",
		Provider:      ProviderOpenAI,
		ModelID:       "gpt-5.5",
		ExpectedModel: "gpt-5.5",
		ResponseModel: "gpt-5.5",
	}
	signals := wrapperFingerprintSignalsForReportWithValues(report, []string{
		`{"error":"API_KEY_REQUIRED","message":"missing Authorization, x-api-key, or x-goog-api-key","routes":["/v1/messages","/backend-api/codex/responses","/antigravity/v1beta/models"]}`,
		`{"data":[{"id":"codex-auto-review"},{"id":"gpt-5.3-codex-spark"},{"id":"models/gemini-3.1-pro-preview-customtools"},{"id":"claude-fable-5"}]}`,
	}, map[string]string{
		"x-client-request-id": "req_123",
	})
	require.Contains(t, signals, "sub2api")
	require.NotContains(t, signals, "sub2api-model-mapping")
	require.NotContains(t, signals, "sub2api-protocol-bridge")
}
func TestWrapperFingerprintSignals_DoesNotFlagSub2APIWeakHeaderAlone(t *testing.T) {
	signals := wrapperFingerprintSignals("relay.example.com", map[string]string{
		"x-client-request-id": "req_123",
	})
	require.NotContains(t, signals, "sub2api")
}
func TestWrapperFingerprintSignals_CoversSub2APIModelMappingLeak(t *testing.T) {
	report := &PublicReport{
		APIBaseHost:   "relay.example.com",
		Provider:      ProviderAnthropic,
		ModelID:       "claude-opus-4-8",
		ExpectedModel: "claude-opus-4-8",
		ResponseModel: "claude-opus-4-8",
	}
	signals := wrapperFingerprintSignalsForReportWithValues(report, []string{
		`{"requested_model":"claude-opus-4-8","upstream_model":"gpt-5.4","model_mapping_chain":["account.model_mapping"]}`,
	}, nil)
	require.Contains(t, signals, "sub2api-model-mapping")

	report.WrapperSignals = signals
	require.True(t, hasWrapperObfuscationFingerprint(report))
	require.Equal(t, 55, wrapperPurityScoreCap(report))
}
func TestWrapperFingerprintSignals_CoversNewAPIHeaderFingerprint(t *testing.T) {
	signals := wrapperFingerprintSignals("relay.example.com", map[string]string{
		"x-new-api-version": "0.9.99",
	})
	require.Contains(t, signals, "new-api")
	require.NotContains(t, signals, "new-api-model-mapping")
}
func TestWrapperFingerprintSignals_CoversCLIProxyAPICodexAndSignatureBridge(t *testing.T) {
	report := &PublicReport{
		APIBaseHost:   "proxy.local",
		Provider:      ProviderOpenAI,
		ModelID:       "gpt-5.4",
		ExpectedModel: "gpt-5.4",
		ResponseModel: "gpt-5.4",
	}
	signals := wrapperFingerprintSignalsForReportWithValues(report, []string{
		`{"message":"CLI Proxy API Server","endpoints":["POST /v1/responses/compact","GET /v1beta/models"]}`,
		`{"oauth-model-alias":[{"name":"glm-5.2","alias":"claude-sonnet-latest","force-mapping":true}]}`,
		`data: {"type":"content_block_delta","delta":{"type":"signature_delta","signature":"claude#abc"}}`,
		`{"native_finish_reason":"MAX_TOKENS","choices":[{"delta":{"reasoning_content":"hidden"}}]}`,
	}, map[string]string{
		"access-control-expose-headers": "X-CPA-VERSION, X-CPA-COMMIT, X-CPA-SUPPORT-PLUGIN",
		"x-codex-turn-metadata":         `{"prompt_cache_key":"cache-1","turn_id":"turn-1"}`,
		"x-codex-window-id":             "cache-1:0",
		"openai-beta":                   "responses_websockets=2026-02-06",
		"originator":                    "codex-tui",
	})
	require.Contains(t, signals, "cliproxyapi")
	require.Contains(t, signals, "cliproxyapi-codex-direct")
	require.Contains(t, signals, "codex")
	require.Contains(t, signals, "cliproxyapi-codex-identity")
	require.Contains(t, signals, "cliproxyapi-model-mapping")
	require.Contains(t, signals, "cliproxyapi-signature-bridge")
}
func TestWrapperPurityScoreCap_DistinguishesTransparentRelayAndObfuscation(t *testing.T) {
	transparent := &PublicReport{
		Provider:       ProviderOpenAI,
		WrapperSignals: []string{"cliproxyapi", "new-api", "sub2api"},
	}
	require.False(t, hasWrapperObfuscationFingerprint(transparent))
	require.Equal(t, CheckStatusPass, buildWrapperFingerprintCheck(transparent).Status)
	require.Equal(t, 100, wrapperPurityScoreCap(transparent))
	require.Contains(t, summaryForReport(transparent), "透明中转/兼容网关")
	require.Contains(t, summaryForReport(transparent), "未显示模型或协议混淆")

	obfuscated := &PublicReport{
		Provider:       ProviderOpenAI,
		WrapperSignals: []string{"cliproxyapi", "cliproxyapi-model-mapping"},
	}
	require.True(t, hasWrapperObfuscationFingerprint(obfuscated))
	require.Equal(t, CheckStatusFail, buildWrapperFingerprintCheck(obfuscated).Status)
	require.Equal(t, 55, wrapperPurityScoreCap(obfuscated))
	require.Contains(t, summaryForReport(obfuscated), "模型或协议混淆风险")
}
func TestWrapperFingerprintSignals_CoversNewAPIErrorBodyFingerprints(t *testing.T) {
	report := &PublicReport{
		APIBaseHost:   "relay.example.com",
		Provider:      ProviderOpenAI,
		ModelID:       "gpt-5.5",
		ExpectedModel: "gpt-5.5",
		ResponseModel: "gpt-5.5",
	}
	// 错误体独有 error.type 命名空间 + request id 后缀 + 分组（distributor）文案
	signals := wrapperFingerprintSignalsForReportWithValues(report, []string{
		`{"error":{"message":"分组 default 下模型 gpt-5.5 无可用渠道（distributor） (request id: 2026xxxx)","type":"new_api_error","code":"model_not_found"}}`,
	}, nil)
	require.Contains(t, signals, "new-api")
	require.NotContains(t, signals, "new-api-model-mapping")
}
func TestWrapperFingerprintSignals_CoversNewAPIFixedConstantHeaders(t *testing.T) {
	// auth-version / specific_channel_version 固定常量值
	signals := wrapperFingerprintSignals("relay.example.com", map[string]string{
		"auth-version": "864b7076dbcd0a3c01b5520316720ebf",
	})
	require.Contains(t, signals, "new-api")
}
func TestWrapperFingerprintSignals_CoversCLIProxyAPIUnauthFingerprints(t *testing.T) {
	report := &PublicReport{
		APIBaseHost:   "proxy.example.com",
		Provider:      ProviderOpenAI,
		ModelID:       "gpt-5.4",
		ExpectedModel: "gpt-5.4",
		ResponseModel: "gpt-5.4",
	}
	// 无鉴权可观测：OAuth 回调固定 HTML + 管理面文案 + 全局 CORS 新头名
	signals := wrapperFingerprintSignalsForReportWithValues(report, []string{
		`<html><head><title>Authentication successful</title></head><body><h1>Authentication successful!</h1><script>setTimeout(function(){window.close();},5000)</script></body></html>`,
		`{"error":"IP banned due to too many failed attempts. Try again in 5m"}`,
	}, map[string]string{
		"access-control-expose-headers": "X-CPA-VERSION, X-SERVER-VERSION, X-SERVER-BUILD-DATE",
	})
	require.Contains(t, signals, "cliproxyapi")
}
func TestWrapperFingerprintSignals_CoversSub2APIStrongProbes(t *testing.T) {
	report := &PublicReport{
		APIBaseHost:   "relay.example.com",
		Provider:      ProviderAnthropic,
		ModelID:       "claude-opus-4-8",
		ExpectedModel: "claude-opus-4-8",
		ResponseModel: "claude-opus-4-8",
	}
	// 预热拦截 mock id（强独立信号）
	mock := wrapperFingerprintSignalsForReportWithValues(report, []string{
		`event: message_start\ndata: {"type":"message_start","message":{"id":"msg_mock_warmup","usage":{"input_tokens":10}}}`,
	}, nil)
	require.Contains(t, mock, "sub2api")

	// 协议门控 404 文案（强独立信号）
	gating := wrapperFingerprintSignalsForReportWithValues(report, []string{
		`{"error":{"type":"not_found_error","message":"Token counting is not supported for this platform"}}`,
	}, nil)
	require.Contains(t, gating, "sub2api")
}
func TestWrapperFingerprintSignals_DoesNotFlagFableModelOrConfigLeakAlone(t *testing.T) {
	// claude-fable-5 现为真实官方模型，单独出现不得判 sub2api
	fable := wrapperFingerprintSignals("api.anthropic.com", map[string]string{
		"content-type": "application/json",
	})
	require.NotContains(t, fable, "sub2api")
	report := &PublicReport{APIBaseHost: "api.anthropic.com", Provider: ProviderAnthropic}
	fableBody := wrapperFingerprintSignalsForReportWithValues(report, []string{
		`{"data":[{"id":"claude-fable-5","type":"model","display_name":"Claude Fable 5"}]}`,
	}, nil)
	require.NotContains(t, fableBody, "sub2api")

	// turnstile/needs_setup 单条弱信号不得单独触发
	weak := wrapperFingerprintSignals("relay.example.com", map[string]string{
		"x-client-request-id": "req_1",
	})
	require.NotContains(t, weak, "sub2api")
}
func TestWrapperFingerprintSignals_CoversAntigravityAndGeminiRoutes(t *testing.T) {
	// Antigravity 专用路由 + 客户端 UA（不依赖 host 含 antigravity）
	report := &PublicReport{APIBaseHost: "relay.example.com", Provider: ProviderAnthropic}
	ag := wrapperFingerprintSignalsForReportWithValues(report, []string{
		`{"routes":["/antigravity/v1/messages","/antigravity/v1beta/models"]}`,
	}, map[string]string{
		"x-goog-api-client": "antigravity/cli 1.0 darwin/arm64",
	})
	require.Contains(t, ag, "antigravity")

	// Gemini 原生协议标记 + Vertex Google 客户端头
	gem := wrapperFingerprintSignals("relay.example.com", map[string]string{
		"x-goog-api-client": "google-api-nodejs-client/10.3.0",
	})
	require.Contains(t, gem, "vertex")
	gemBody := wrapperFingerprintSignalsForReportWithValues(
		&PublicReport{APIBaseHost: "relay.example.com", Provider: ProviderOpenAI},
		[]string{`{"models":[{"name":"models/gemini-3-flash","supportedGenerationMethods":["generateContent"]}]}`},
		nil,
	)
	require.Contains(t, gemBody, "gemini")
	nativeGeminiBody := wrapperFingerprintSignalsForReportWithValues(
		&PublicReport{APIBaseHost: "relay.example.com", Provider: ProviderGemini},
		[]string{`{"models":[{"name":"models/gemini-3-flash","supportedGenerationMethods":["generateContent"]}]}`},
		nil,
	)
	require.NotContains(t, nativeGeminiBody, "gemini")
}
func TestWrapperFingerprintSignals_DoesNotFlagOfficialCodexModelNameAlone(t *testing.T) {
	report := &PublicReport{
		APIBaseHost:   "api.openai.com",
		Provider:      ProviderOpenAI,
		ModelID:       "gpt-5-codex",
		ExpectedModel: "gpt-5-codex",
		ResponseModel: "gpt-5-codex",
	}
	signals := wrapperFingerprintSignalsForReport(report)
	require.NotContains(t, signals, "codex")
}
func TestWrapperFingerprintSignals_CoversMainstreamRelayBrands(t *testing.T) {
	signals := wrapperFingerprintSignals("api.pptoken.org", map[string]string{
		"x-litellm-version": "1.70.0",
		"server":            "LiteLLM",
		"x-provider":        "api2d-aiproxy-openmodel-pateway-suixiang-ohmygpt",
	})
	require.Contains(t, signals, "litellm")
	require.Contains(t, signals, "api2d")
	require.Contains(t, signals, "aiproxy")
	require.Contains(t, signals, "openmodel")
	require.Contains(t, signals, "pptoken")
	require.Contains(t, signals, "pateway")
	require.Contains(t, signals, "suixiang")
	require.Contains(t, signals, "ohmygpt")
}
func TestWrapperFingerprintSignals_CoversChineseOpenModelChannels(t *testing.T) {
	report := &PublicReport{
		APIBaseHost:   "ark.cn-beijing.volces.com",
		ModelID:       "qwen3.7-max",
		ExpectedModel: "glm-5.2",
		ResponseModel: "doubao-seed-2-1-pro-260628",
	}
	signals := wrapperFingerprintSignalsForReport(report, map[string]string{
		"x-provider":          "MiniMax-M2.7-highspeed",
		"x-upstream-provider": "hy3-preview",
		"x-model-provider":    "kimi-k2.7-code-highspeed mimo-v2.5-pro",
	})
	require.Contains(t, signals, "qwen")
	require.Contains(t, signals, "glm")
	require.Contains(t, signals, "doubao")
	require.Contains(t, signals, "minimax")
	require.Contains(t, signals, "hunyuan")
	require.Contains(t, signals, "kimi")
	require.Contains(t, signals, "mimo")
}
func TestWrapperFingerprintSignals_CoversDeepSeekChannel(t *testing.T) {
	signals := wrapperFingerprintSignals("api.deepseek.com", map[string]string{
		"x-upstream-provider": "deepseek-v4-pro",
	})
	require.Contains(t, signals, "deepseek")
}
func TestWrapperFingerprintSignals_CoversCloudProviderChannels(t *testing.T) {
	signals := wrapperFingerprintSignals("us-central1-aiplatform.googleapis.com", map[string]string{
		"x-goog-request-id": "goog-req",
		"x-amzn-requestid":  "aws-req",
	})
	require.Contains(t, signals, "vertex")
	require.Contains(t, signals, "bedrock")
}
