package doubao

import "github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels"

type Detector struct{}

func (Detector) Detect(ctx channels.Context) []string {
	return channels.SignalIfContains(ctx, "doubao", "doubao", "doubao-seed", "seed-2-1", "seed-2.1", "seed-2-0", "seed-2.0", "volcengine", "ark.cn-beijing.volces.com", "volces.com", "byteplus", "bytedance")
}
