package attribution

import "strings"

const (
	ChannelKindOfficialNative = "official_native"
	ChannelKindOfficialCloud  = "official_cloud"
	ChannelKindAggregator     = "aggregator"
)

type endpointProfile struct {
	Host    string
	Channel string
	Code    string
	Kind    string
}

// These endpoints are stable language-model and compatibility channel bases
// also used by new-api adaptors. Task-only, media-only and user-defined hosts
// are excluded because they cannot be attributed reliably by host alone.
var endpointProfiles = []endpointProfile{
	{Host: "api.anthropic.com", Channel: "anthropic_native", Code: "official_anthropic_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "api.openai.com", Channel: "openai_native", Code: "official_openai_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "generativelanguage.googleapis.com", Channel: "google_ai_studio", Code: "official_google_ai_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "dashscope.aliyuncs.com", Channel: "alibaba_bailian", Code: "official_alibaba_bailian_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "aip.baidubce.com", Channel: "baidu_wenxin", Code: "official_baidu_wenxin_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "qianfan.baidubce.com", Channel: "baidu_qianfan", Code: "official_baidu_qianfan_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "api.360.cn", Channel: "ai360", Code: "official_ai360_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "open.bigmodel.cn", Channel: "zhipu_bigmodel", Code: "official_zhipu_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "hunyuan.tencentcloudapi.com", Channel: "tencent_hunyuan", Code: "official_tencent_hunyuan_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "api.moonshot.cn", Channel: "moonshot", Code: "official_moonshot_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "api.perplexity.ai", Channel: "perplexity", Code: "official_perplexity_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "api.lingyiwanwu.com", Channel: "yi", Code: "official_yi_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "api.cohere.ai", Channel: "cohere", Code: "official_cohere_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "api.minimax.chat", Channel: "minimax", Code: "official_minimax_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "api.siliconflow.cn", Channel: "siliconflow", Code: "official_siliconflow_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "api.mistral.ai", Channel: "mistral", Code: "official_mistral_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "api.deepseek.com", Channel: "deepseek", Code: "official_deepseek_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "ark.cn-beijing.volces.com", Channel: "volcengine_ark", Code: "official_volcengine_ark_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "api.x.ai", Channel: "xai", Code: "official_xai_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "api.z.ai", Channel: "zai_coding", Code: "official_zai_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "api.kimi.com", Channel: "kimi_coding", Code: "official_kimi_coding_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "chatgpt.com", Channel: "openai_codex_subscription", Code: "official_openai_codex_endpoint", Kind: ChannelKindOfficialNative},
	{Host: "openrouter.ai", Channel: "openrouter", Code: "official_openrouter_endpoint", Kind: ChannelKindAggregator},
	{Host: "api.cloudflare.com", Channel: "cloudflare_workers_ai", Code: "official_cloudflare_workers_ai_endpoint", Kind: ChannelKindOfficialCloud},
	{Host: "api.dify.ai", Channel: "dify", Code: "official_dify_endpoint", Kind: ChannelKindAggregator},
	{Host: "api.coze.cn", Channel: "coze", Code: "official_coze_endpoint", Kind: ChannelKindAggregator},
	{Host: "fastgpt.run", Channel: "fastgpt", Code: "official_fastgpt_endpoint", Kind: ChannelKindAggregator},
	{Host: "llm.submodel.ai", Channel: "submodel", Code: "official_submodel_endpoint", Kind: ChannelKindAggregator},
	{Host: "api.openai-sb.com", Channel: "openai_sb", Code: "known_openai_sb_endpoint", Kind: ChannelKindAggregator},
	{Host: "api.openaimax.com", Channel: "openaimax", Code: "known_openaimax_endpoint", Kind: ChannelKindAggregator},
	{Host: "api.ohmygpt.com", Channel: "ohmygpt", Code: "known_ohmygpt_endpoint", Kind: ChannelKindAggregator},
	{Host: "api.caipacity.com", Channel: "caipacity", Code: "known_caipacity_endpoint", Kind: ChannelKindAggregator},
	{Host: "api.aiproxy.io", Channel: "aiproxy", Code: "known_aiproxy_endpoint", Kind: ChannelKindAggregator},
	{Host: "api.api2gpt.com", Channel: "api2gpt", Code: "known_api2gpt_endpoint", Kind: ChannelKindAggregator},
	{Host: "api.aigc2d.com", Channel: "aigc2d", Code: "known_aigc2d_endpoint", Kind: ChannelKindAggregator},
}

func matchEndpointProfile(host string) (endpointProfile, bool) {
	host = normalizedHost(host)
	for _, profile := range endpointProfiles {
		if host == profile.Host {
			return profile, true
		}
	}
	switch {
	case strings.HasSuffix(host, ".openai.azure.com"),
		strings.HasSuffix(host, ".services.ai.azure.com"),
		strings.HasSuffix(host, ".cognitiveservices.azure.com"):
		return endpointProfile{Channel: "azure_openai", Code: "official_azure_openai_endpoint", Kind: ChannelKindOfficialCloud}, true
	case strings.Contains(host, "bedrock") && strings.HasSuffix(host, ".amazonaws.com"):
		return endpointProfile{Channel: "aws_bedrock", Code: "official_aws_bedrock_endpoint", Kind: ChannelKindOfficialCloud}, true
	case strings.HasSuffix(host, "-aiplatform.googleapis.com"),
		strings.HasSuffix(host, ".aiplatform.googleapis.com"),
		strings.HasSuffix(host, ".vertexai.googleapis.com"),
		host == "aiplatform.googleapis.com":
		return endpointProfile{Channel: "google_vertex", Code: "official_google_vertex_endpoint", Kind: ChannelKindOfficialCloud}, true
	default:
		return endpointProfile{}, false
	}
}

func ChannelKind(channel string) string {
	channel = strings.ToLower(strings.TrimSpace(channel))
	for _, profile := range endpointProfiles {
		if profile.Channel == channel {
			return profile.Kind
		}
	}
	switch channel {
	case "azure_openai", "aws_bedrock", "google_vertex":
		return ChannelKindOfficialCloud
	default:
		return ""
	}
}

func normalizedHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if strings.HasPrefix(host, "[") {
		if end := strings.Index(host, "]"); end >= 0 {
			return strings.Trim(host[1:end], ".")
		}
	}
	return strings.Trim(strings.Split(host, ":")[0], ".")
}
