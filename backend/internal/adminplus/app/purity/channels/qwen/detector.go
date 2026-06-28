package qwen

import "github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels"

type Detector struct{}

func (Detector) Detect(ctx channels.Context) []string {
	return channels.SignalIfContains(ctx, "qwen", "qwen", "qwen3.", "qwen3-", "qwen-plus", "qwen-max", "qwen-coder", "qwen-vl", "dashscope", "tongyi", "bailian", "aliyun", "aliyuncs.com")
}
