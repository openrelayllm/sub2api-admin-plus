package purity

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOfficialPurityScore_TokenAuditWarnUsesReservedWeight(t *testing.T) {
	report := &PublicReport{
		Provider: ProviderOpenAI,
		Status:   RunStatusDone,
		Checks: []CheckResult{
			passCheck("base_url", "API Base 域名", 20, "ok", nil),
			{ID: "token_audit", Name: "Token 用量审计", Status: CheckStatusWarn, Score: 8, MaxScore: 15, Message: "Responses 状态链路不完整"},
		},
		Validations: []ValidationResult{
			{ID: "llm_fingerprint", Status: CheckStatusPass},
			{ID: "schema_integrity", Status: CheckStatusPass},
			{ID: "behavior", Status: CheckStatusPass},
			{ID: "signature", Status: CheckStatusPass},
			{ID: "multimodal", Status: CheckStatusPass},
			{ID: "token_audit", Status: CheckStatusWarn},
		},
		TokenAudit: &TokenAuditReport{Status: CheckStatusWarn, Summary: "Responses 状态链路不完整", SampleCount: tokenAuditSamples},
	}

	finalizeReport(report)

	require.Equal(t, 95, report.OfficialScore)
	require.Equal(t, 95, report.Score)
}

func TestOfficialPurityScore_SkippedTokenAuditDoesNotLowerScore(t *testing.T) {
	report := &PublicReport{
		Provider: ProviderGemini,
		Status:   RunStatusDone,
		Checks: []CheckResult{
			passCheck("base_url", "API Base 域名", 20, "ok", nil),
			{ID: "token_audit", Name: "Token 用量审计", Status: CheckStatusWarn, Score: 0, MaxScore: 0, Message: "skipped", Details: map[string]any{"skipped": true}},
		},
		Validations: []ValidationResult{
			{ID: "llm_fingerprint", Status: CheckStatusPass},
			{ID: "schema_integrity", Status: CheckStatusPass},
			{ID: "behavior", Status: CheckStatusPass},
			{ID: "signature", Status: CheckStatusPass},
			{ID: "multimodal", Status: CheckStatusPass},
			{ID: "token_audit", Status: CheckStatusWarn},
		},
	}

	finalizeReport(report)

	require.Equal(t, 100, report.OfficialScore)
	require.Equal(t, 100, report.Score)
}

func TestPublicReportCCTestCompatFields(t *testing.T) {
	report := &PublicReport{
		Provider:         ProviderOpenAI,
		ReportID:         "report-compat",
		APIBaseHost:      "api.proxyai.best",
		ModelID:          "gpt-5.4",
		ExpectedModel:    "gpt-5.4",
		ResponseModel:    "gpt-5.4",
		Status:           RunStatusRunning,
		Step:             6,
		StepName:         "token_audit",
		Progress:         0.86,
		Verdict:          VerdictOpenAICompatible,
		StreamChannel:    "openai_compatible",
		NonStreamChannel: "openai_compatible",
		HasVertex:        true,
		IsKiro:           true,
		WrapperSignals:   []string{"vertex", "kiro"},
		Checks: []CheckResult{
			passCheck("base_url", "API Base 域名", 20, "ok", nil),
			passCheck("responses_schema", "Responses 非流式结构", 20, "ok", nil),
			passCheck("tool_call", "强制工具调用", 20, "ok", nil),
			passCheck("usage", "Usage 计量", 10, "ok", nil),
			passCheck("streaming", "Responses 流式事件", 15, "ok", nil),
			{ID: "responses_store_include", Name: "Responses store/include", Status: CheckStatusPass, Score: 0, MaxScore: 0, Message: "ok"},
			passCheck("multimodal", "多模态输入", 10, "ok", nil),
			passCheck("token_audit", "Token 用量审计", 15, "ok", nil),
		},
		Metrics: PublicCheckMetrics{
			LatencyMS:            321,
			StreamFirstTokenMS:   45,
			StreamTotalLatencyMS: 180,
			ModelsLatencyMS:      12,
			ResponsesLatencyMS:   98,
			MultimodalLatencyMS:  110,
			TokensPerSecond:      18.75,
			Usage:                &TokenUsage{InputTokens: 1200, OutputTokens: 88, TotalTokens: 1288, CachedTokens: 1024},
		},
		CheckedAt: time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC),
	}
	finalizeReport(report)

	raw, err := json.Marshal(report)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	require.Equal(t, "token_audit", payload["stepName"])
	require.Equal(t, float64(100), payload["total"])
	require.Equal(t, "official", payload["verdictKey"])
	require.Equal(t, "gpt-5.4", payload["expectedModel"])
	require.Equal(t, "gpt-5.4", payload["responseModel"])
	require.Equal(t, "openai_compatible", payload["streamChannel"])
	require.Equal(t, "openai_compatible", payload["nonStreamChannel"])
	require.Equal(t, true, payload["hasVertex"])
	require.Equal(t, true, payload["isKiro"])
	require.Equal(t, []any{"vertex", "kiro"}, payload["wrapperSignals"])

	metrics, ok := payload["metrics"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(321), metrics["latencyMs"])
	require.Equal(t, float64(45), metrics["ttfbMs"])
	require.Equal(t, 18.75, metrics["tokensPerSec"])
	require.Equal(t, float64(1200), metrics["inputTokens"])
	require.Equal(t, float64(88), metrics["outputTokens"])

	var emitted PublicCheckEvent
	emitPublicCheckEvent(func(event PublicCheckEvent) {
		emitted = event
	}, PublicCheckEvent{
		Type:     PublicCheckEventProgress,
		ReportID: report.ReportID,
		StepName: report.StepName,
		Metrics:  &report.Metrics,
		Report:   report,
	})
	eventRaw, err := json.Marshal(emitted)
	require.NoError(t, err)
	var eventPayload map[string]any
	require.NoError(t, json.Unmarshal(eventRaw, &eventPayload))
	require.Equal(t, "token_audit", eventPayload["stepName"])
	eventMetrics, ok := eventPayload["metrics"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(321), eventMetrics["latencyMs"])
	require.Equal(t, float64(45), eventMetrics["ttfbMs"])
	require.Equal(t, 18.75, eventMetrics["tokensPerSec"])
}
