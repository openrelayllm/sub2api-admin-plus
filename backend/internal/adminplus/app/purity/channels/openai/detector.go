package openai

import "github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels"

type Detector struct{}

func (Detector) Detect(ctx channels.Context) []string {
	signals := make([]string, 0, 11)
	if found := channels.SignalIfContains(ctx, "openrouter", "openrouter"); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfContains(ctx, "litellm", "litellm", "x-litellm", "llm-proxy"); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfContains(ctx, "api2d", "api2d"); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfContains(ctx, "aiproxy", "aiproxy", "ai-proxy"); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfContains(ctx, "openmodel", "openmodel"); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfContains(ctx, "pptoken", "pptoken"); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfContains(ctx, "pateway", "pateway"); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfContains(ctx, "suixiang", "suixiang", "sui-xiang"); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfContains(ctx, "ohmygpt", "ohmygpt", "oh-my-gpt"); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfContains(ctx, "openai-compatible", "openai-compatible", "openai-compatibility"); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfHeaderContains(ctx, "codex", "codex", "x-codex", "codex-tui", "codex_cli_rs", "originator:codex", "responses_websockets", "chatgpt-account-id", "thread-id", "session_id", "conversation_id"); len(found) > 0 {
		signals = append(signals, found...)
	}
	return signals
}
