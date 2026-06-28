package cliproxyapi

import "github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels"

type Detector struct{}

func (Detector) Detect(ctx channels.Context) []string {
	signals := make([]string, 0, 5)
	if found := channels.SignalIfContains(ctx, "cliproxyapi",
		"cliproxy",
		"cli-proxy",
		"cli proxy api",
		"cli proxy api server",
		"router-for-me",
		"cpa-support-plugin",
		"x-cpa-",
		"access-control-expose-headers:x-cpa-version",
		"access-control-expose-headers:x-cpa-commit",
		"access-control-expose-headers:x-cpa-support-plugin",
		"x-cpa-version",
		"x-cpa-commit",
		"x-cpa-build-date",
		"x-cpa-home-version",
		"x-cpa-home-build-date",
		// 新增：CORS Expose-Headers 新发现头名 + server/home 版本头
		"x-server-version",
		"x-server-build-date",
		// 新增：根响应 / 健康检查精确 JSON
		`"cli proxy api server"`,
		`"post /v1/chat/completions","post /v1/completions","get /v1/models"`,
		// 新增：OAuth 回调固定 HTML（4 路径免鉴权）
		"authentication successful!",
		"window.close()",
		// 新增：管理面精确文案
		"remote management disabled",
		"ip banned due to too many failed attempts",
		"missing management key",
		"invalid management key",
		"invalid_yaml",
		"home control center",
		"management.html",
		"v0/management",
	); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfContains(ctx, "cliproxyapi-codex-direct",
		"v1/responses/compact",
		"responses/compact",
		"codex-tui/0.135.0",
		"responses_websockets=2026-02-06",
	); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfHeaderContains(ctx, "codex",
		"codex",
		"x-codex",
		"codex-tui",
		"codex_cli_rs",
		"originator:codex",
		"responses_websockets",
		"chatgpt-account-id",
		"thread-id",
		"session_id",
		"session-id",
		"conversation_id",
	); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfContains(ctx, "cliproxyapi-codex-identity",
		"x-codex-turn-metadata",
		"x-codex-window-id",
		"x-codex-installation-id",
		"prompt_cache_key",
		"thread-id",
		"session_id",
		"session-id",
		"conversation_id",
		"client_metadata.x-codex-turn-metadata",
	); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfContains(ctx, "cliproxyapi-model-mapping",
		"force-mapping",
		"forcemapping",
		"model-aliases",
		"oauth-model-alias",
		"openai-compatibility",
		"requested_model",
		"upstream_model",
		"mapping_chain",
	); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfContains(ctx, "cliproxyapi-signature-bridge",
		"skip_thought_signature_validator",
		"signature_delta",
		"thinking_delta",
		"encrypted_content",
		"reasoning_content",
		"reasoning.encrypted_content",
		"response.reasoning_summary_text.delta",
		"native_finish_reason",
		"gemini#",
		"claude#",
		"gpt#",
		"target_provider",
		"detected_provider",
	); len(found) > 0 {
		signals = append(signals, found...)
	}
	return signals
}
