package gemini

import "github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels"

type Detector struct{}

func (Detector) Detect(ctx channels.Context) []string {
	signals := make([]string, 0, 3)
	if found := channels.SignalIfContains(ctx, "gemini",
		"gemini",
		// Gemini 原生协议标记（/v1beta/models 响应特征）
		"supportedgenerationmethods",
	); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfContains(ctx, "aistudio", "aistudio", "ai-studio"); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfContains(ctx, "vertex",
		"vertex",
		"aiplatform.googleapis.com",
		"vertexai.googleapis.com",
		"x-goog-request-id",
		"x-cloud-trace-context",
		"x-goog-api-client",
	); len(found) > 0 {
		signals = append(signals, found...)
	}
	return signals
}
