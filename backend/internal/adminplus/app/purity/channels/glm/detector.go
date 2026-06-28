package glm

import "github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels"

type Detector struct{}

func (Detector) Detect(ctx channels.Context) []string {
	return channels.SignalIfContains(ctx, "glm", "glm", "glm-5", "glm-4.7", "chatglm", "zhipu", "zhipuai", "z.ai", "z-ai", "bigmodel", "bigmodel.cn")
}
