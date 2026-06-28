package sub2api

import "github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels"

type Detector struct{}

func (Detector) Detect(ctx channels.Context) []string {
	signals := make([]string, 0, 3)
	brand := channels.ContainsAny(ctx,
		"sub2api",
		"sub2-api",
		"proxyai.best",
	)
	header := channels.ContainsAny(ctx,
		"x-client-request-id",
	)
	route := channels.ContainsAny(ctx,
		"/setup/status",
		"/api/event_logging/batch",
		"/api/v1/settings/public",
		"/api/v1/auth/oauth/linuxdo",
		"/backend-api/codex/responses",
		"/v1beta/models",
		"/antigravity/v1",
	)
	protocolRoute := countTrue(
		channels.ContainsAny(ctx, "/v1/messages"),
		channels.ContainsAny(ctx, "/v1/responses", "/responses"),
		channels.ContainsAny(ctx, "/v1/chat/completions", "/chat/completions"),
		channels.ContainsAny(ctx, "/v1beta/models"),
	) >= 2
	authError := channels.ContainsAny(ctx,
		"api_key_required",
		"authorization header (bearer scheme), x-api-key header, or x-goog-api-key header",
		"api key is not assigned to any group",
		// 中英混排错误体（code 英文 + message 中文，sub2api 业务文案）
		"api key 所属分组已删除",
		"api key 额度已用完",
		"api key 已过期",
		"api_key_in_query_deprecated",
	)
	// 强独立信号：本地预热拦截响应（mock id 几乎不可能误判）
	mockIntercept := channels.ContainsAny(ctx,
		"msg_mock_warmup",
		"msg_mock_suggestion",
	)
	// 强独立信号：协议门控 404 文案（sub2api 平台路由特有）
	gatingError := channels.ContainsAny(ctx,
		"is not supported for grok groups",
		"embeddings api is not supported for this platform",
		"images api is not supported for this platform",
		"token counting is not supported for this platform",
	)
	// 弱信号：未鉴权配置泄露，仅作组合证据（turnstile/needs_setup 非 sub2api 独有）
	configLeak := channels.ContainsAny(ctx,
		"turnstile_site_key",
		"needs_setup",
	)
	// 仅作发现别名/降级用的兼容网关内置模型清单。
	// 注意：claude-fable-5 已是真实官方模型，不能单独作为 sub2api 强证据。
	modelList := channels.ContainsAny(ctx,
		"codex-auto-review",
		"gpt-5.3-codex-spark",
		"models/gemini-3.1-pro-preview-customtools",
	)
	if brand || authError || mockIntercept || gatingError || modelList ||
		(header && (route || protocolRoute)) ||
		countTrue(header, route, protocolRoute, authError, modelList, configLeak) >= 2 {
		signals = append(signals, "sub2api")
	}
	if found := channels.SignalIfContains(ctx, "sub2api-model-mapping",
		"model_mapping_chain",
		"compact_model_mapping",
		`"model_mapping"`,
		`"requested_model"`,
		`"upstream_model"`,
	); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfContains(ctx, "sub2api-protocol-bridge",
		"target_provider",
		"detected_provider",
		"original_provider",
		"provider_mismatch",
		"protocol_model_vendor_mismatch",
		`"upstream_provider"`,
	); len(found) > 0 {
		signals = append(signals, found...)
	}
	return signals
}

func countTrue(values ...bool) int {
	count := 0
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}
