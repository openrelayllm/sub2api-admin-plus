package mimo

import "github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels"

type Detector struct{}

func (Detector) Detect(ctx channels.Context) []string {
	return channels.SignalIfContains(ctx, "mimo", "mimo", "mimo-v2", "mimo-v2.5", "mimo-code", "mi-moment", "xiaomimimo", "mimo.mi.com", "xiaomi")
}
