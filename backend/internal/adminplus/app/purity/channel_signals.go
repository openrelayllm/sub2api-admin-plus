package purity

import (
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels/bedrock"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels/claude"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels/cliproxyapi"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels/deepseek"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels/doubao"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels/gemini"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels/glm"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels/hunyuan"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels/kimi"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels/mimo"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels/minimax"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels/newapi"
	channelsopenai "github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels/openai"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels/qwen"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels/sub2api"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/app/purity/channels/xai"
)

var channelSignalDetectors = []channels.Detector{
	claude.Detector{},
	cliproxyapi.Detector{},
	newapi.Detector{},
	sub2api.Detector{},
	channelsopenai.Detector{},
	gemini.Detector{},
	antigravity.Detector{},
	bedrock.Detector{},
	kimi.Detector{},
	xai.Detector{},
	deepseek.Detector{},
	qwen.Detector{},
	glm.Detector{},
	doubao.Detector{},
	minimax.Detector{},
	hunyuan.Detector{},
	mimo.Detector{},
}

func wrapperFingerprintSignals(host string, headerSets ...map[string]string) []string {
	ctx := channels.NewContext(host, headerSets...)
	return wrapperFingerprintSignalsFromContext(ctx)
}

func wrapperFingerprintSignalsForReport(report *PublicReport, headerSets ...map[string]string) []string {
	return wrapperFingerprintSignalsForReportWithValues(report, nil, headerSets...)
}

func wrapperFingerprintSignalsForReportWithValues(report *PublicReport, values []string, headerSets ...map[string]string) []string {
	if report == nil {
		ctx := channels.NewContext("", headerSets...)
		ctx = ctx.WithValues(values...)
		return wrapperFingerprintSignalsFromContext(ctx)
	}
	ctx := channels.NewContext(report.APIBaseHost, headerSets...)
	ctx = ctx.WithValues(report.ModelID, report.ExpectedModel, report.ResponseModel)
	ctx = ctx.WithValues(values...)
	return filterProviderNativeSignals(report.Provider, wrapperFingerprintSignalsFromContext(ctx))
}

func wrapperFingerprintSignalsFromContext(ctx channels.Context) []string {
	signals := make([]string, 0, len(channelSignalDetectors))
	for _, detector := range channelSignalDetectors {
		for _, signal := range detector.Detect(ctx) {
			signals = appendUniqueString(signals, signal)
		}
	}
	return signals
}

func filterProviderNativeSignals(provider string, signals []string) []string {
	if len(signals) == 0 {
		return signals
	}
	out := make([]string, 0, len(signals))
	for _, signal := range signals {
		if provider == ProviderGemini && (signal == "gemini" || signal == "aistudio") {
			continue
		}
		out = append(out, signal)
	}
	return out
}

func containsWrapperSignal(signals []string, target string) bool {
	return channels.ContainsSignal(signals, target)
}

func hasVertexFingerprint(host string, headerSets ...map[string]string) bool {
	return containsWrapperSignal(wrapperFingerprintSignals(host, headerSets...), "vertex")
}

func hasKiroFingerprint(host string, headerSets ...map[string]string) bool {
	return containsWrapperSignal(wrapperFingerprintSignals(host, headerSets...), "kiro")
}
