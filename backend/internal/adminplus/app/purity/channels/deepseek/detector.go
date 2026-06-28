package deepseek

import "github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels"

type Detector struct{}

func (Detector) Detect(ctx channels.Context) []string {
	return channels.SignalIfContains(ctx, "deepseek", "deepseek", "deepseek-v4", "deepseek-v3.2", "deepseek-chat", "deepseek-reasoner", "deepseek.com", "deepseek-ai")
}
