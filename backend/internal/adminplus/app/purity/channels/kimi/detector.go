package kimi

import "github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels"

type Detector struct{}

func (Detector) Detect(ctx channels.Context) []string {
	return channels.SignalIfContains(ctx, "kimi", "kimi", "kimi-k2", "k2.7", "moonshot", "moonshot.cn", "api.moonshot.cn", "kimi.ai", "platform.kimi")
}
