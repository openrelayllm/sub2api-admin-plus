package bedrock

import "github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels"

type Detector struct{}

func (Detector) Detect(ctx channels.Context) []string {
	return channels.SignalIfContains(ctx, "bedrock", "bedrock", "x-amzn-requestid", "x-amzn-trace-id")
}
