package newapi

import "github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels"

type Detector struct{}

func (Detector) Detect(ctx channels.Context) []string {
	signals := make([]string, 0, 2)
	if found := channels.SignalIfContains(ctx, "new-api",
		"new-api",
		"newapi",
		"oneapi",
		"one-api",
		"x-oneapi-request-id",
		"x-new-api-version",
		"x-translate-id",
		// 固定常量头（new-api 家族独有）
		"864b7076dbcd0a3c01b5520316720ebf",         // auth-version
		"701e3ae1dc3f7975556d354e0675168d004891c8", // specific_channel_version
		// 错误体强指纹：独有 error.type 命名空间 + 分组文案 + 未实现端点
		"new_api_error",
		"（distributor）",
		"(distributor)",
		"api_not_implemented",
		"api not implemented",
		"github.com/quantumnous/new-api",
		"calciumion/new-api",
	); len(found) > 0 {
		signals = append(signals, found...)
	}
	if found := channels.SignalIfContains(ctx, "new-api-model-mapping",
		"channel_model_mapping",
		"contextkeychannelmodelmapping",
		"upstream_model_name",
		`"upstream_model"`,
	); len(found) > 0 {
		signals = append(signals, found...)
	}
	return signals
}
