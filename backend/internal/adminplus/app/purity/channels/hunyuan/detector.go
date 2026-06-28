package hunyuan

import "github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels"

type Detector struct{}

func (Detector) Detect(ctx channels.Context) []string {
	return channels.SignalIfContains(ctx, "hunyuan", "hunyuan", "hunyuan-a13b", "hunyuan-2.0", "hy3", "tokenhub", "lkeap", "tencent", "tencentcloudapi.com")
}
