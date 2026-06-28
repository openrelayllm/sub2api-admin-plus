package antigravity

import "github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels"

type Detector struct{}

func (Detector) Detect(ctx channels.Context) []string {
	return channels.SignalIfContains(ctx, "antigravity",
		"antigravity",
		// Antigravity 专用路由（sub2api / CLIProxyAPI 暴露）
		"/antigravity/models",
		"/antigravity/v1/messages",
		"/antigravity/v1beta/models",
		"/antigravity/callback",
		"antigravity-auth-url",
		// Antigravity 客户端身份（上游伪装头，若回显可见）
		"antigravity/cli",
	)
}
