package purity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"strings"
	"time"
)

const (
	defaultClaudeModel       = "claude-opus-4-8"
	anthropicVersion         = "2023-06-01"
	claudeCodeProbeVersion   = "2.1.161"
	claudeFingerprintSalt    = "59cf53e54c78"
	claudeCodeProbeUserAgent = "claude-cli/2.1.161 (external, cli)"
	claudeCodeSystemPrompt   = "You are Claude Code, Anthropic's official CLI for Claude."
	claudeAPIKeyBetaHeader   = "claude-code-20250219,interleaved-thinking-2025-05-14,fine-grained-tool-streaming-2025-05-14"
)

type claudeProbeContext struct {
	deviceID  string
	sessionID string
}

type claudeAuditTurn struct {
	Round         int
	UserText      string
	AssistantText string
}

func newClaudeProbeContext() claudeProbeContext {
	deviceBytes := make([]byte, 32)
	if _, err := rand.Read(deviceBytes); err != nil {
		fallback := sha256.Sum256([]byte(uuid.NewString()))
		copy(deviceBytes, fallback[:])
	}
	return claudeProbeContext{
		deviceID:  hex.EncodeToString(deviceBytes),
		sessionID: uuid.NewString(),
	}
}

func (probe claudeProbeContext) metadata() map[string]any {
	userID, _ := json.Marshal(map[string]string{
		"device_id":    probe.deviceID,
		"account_uuid": "",
		"session_id":   probe.sessionID,
	})
	return map[string]any{"user_id": string(userID)}
}

func (s *Service) runClaudeCheck(ctx context.Context, in PublicCheckInput, emit PublicCheckEventSink, options checkRunOptions) (*PublicReport, error) {
	apiKey := strings.TrimSpace(in.APIKey)
	if apiKey == "" {
		return nil, infraerrors.BadRequest("PURITY_API_KEY_REQUIRED", "api key is required")
	}
	if len(apiKey) > maxAPIKeyLength {
		return nil, infraerrors.BadRequest("PURITY_API_KEY_TOO_LARGE", "api key is too large")
	}
	if options.EnforceRateLimit {
		if err := s.enforceRateLimit(in.ClientIP, apiKey); err != nil {
			return nil, err
		}
	}

	model := strings.TrimSpace(in.ModelID)
	if model == "" {
		model = defaultClaudeModel
	}
	baseURL, host, officialHost, err := s.normalizeAnthropicBaseURL(in.APIBaseURL, options.AllowPrivateHosts)
	if err != nil {
		return nil, err
	}
	client := s.clientForRun(options)
	probeCtx := newClaudeProbeContext()

	startedAt := s.currentTime()
	checkCtx, cancel := context.WithTimeout(ctx, defaultCheckTimeout)
	defer cancel()

	checkedAt := startedAt.UTC()
	report := &PublicReport{
		Provider:        ProviderAnthropic,
		ReportID:        newReportID(ProviderAnthropic, host, model, checkedAt),
		AccessMode:      normalizeAccessMode(options.AccessMode),
		BillingMode:     normalizeBillingMode(options.BillingMode),
		APIBaseHost:     host,
		ModelID:         model,
		CheckTokenUsage: !in.SkipTokenAudit,
		ExpectedModel:   model,
		Status:          RunStatusRunning,
		Verdict:         VerdictUnknown,
		Validations:     []ValidationResult{},
		Checks:          []CheckResult{},
		CheckedAt:       checkedAt,
		Metrics:         PublicCheckMetrics{},
	}
	emitPublicCheckEvent(emit, PublicCheckEvent{
		Type:     PublicCheckEventStarted,
		ReportID: report.ReportID,
		Status:   report.Status,
		Report:   clonePublicReport(report),
	})

	emitProgress(report, emit, 1, "tag")
	baseURLCheck := buildClaudeBaseURLCheck(host, officialHost)
	appendAndEmitChecks(report, emit, baseURLCheck)
	gatewayProbe := httpProbe{}
	if !officialHost && shouldProbeGatewayRoot(host) {
		gatewayProbe = s.probeGatewayRoot(checkCtx, client, baseURL)
	}

	emitProgress(report, emit, 2, "structure")
	messagesProbe := s.probeClaudeMessages(checkCtx, client, baseURL, apiKey, model, probeCtx)
	report.Metrics.MessagesLatencyMS = messagesProbe.LatencyMS
	report.Metrics.Usage = parseClaudeUsage(messagesProbe.Body)
	report.ResponseModel = strings.TrimSpace(gjson.GetBytes(messagesProbe.Body, "model").String())
	report.NonStreamChannel = channelFromProbe(ProviderAnthropic, host, officialHost, messagesProbe)
	schemaCheck := buildClaudeMessagesSchemaCheck(messagesProbe, apiKey)
	toolCheck := buildClaudeToolUseCheck(messagesProbe)
	usageCheck := buildClaudeUsageCheck(report.Metrics.Usage, messagesProbe)
	appendAndEmitChecks(report, emit, schemaCheck, toolCheck, usageCheck)
	upsertAndEmitValidation(report, emit, validationFromExecutedChecks("llm_fingerprint", "LLM 指纹验证", []CheckResult{baseURLCheck, schemaCheck}))
	upsertAndEmitValidation(report, emit, validationFromExecutedChecks("schema_integrity", "结构完整性", []CheckResult{schemaCheck}))
	emitMetrics(report, emit)

	if shouldStopAfterProbe(messagesProbe) {
		markReportProbeError(report, messagesProbe, "基础连接或鉴权未通过")
		appendAndEmitChecks(report, emit, skippedClaudeCoreChecks("基础连接或鉴权未通过，后续 Claude 探测未执行")...)
		upsertAndEmitValidation(report, emit, skippedValidation("behavior", "行为验证", []string{"claude_tool_use", "claude_streaming"}, "基础连接或鉴权未通过，行为验证未执行"))
		upsertAndEmitValidation(report, emit, skippedValidation("multimodal", "多模态能力", []string{"claude_multimodal"}, "基础连接或鉴权未通过，多模态探测未执行"))
		upsertAndEmitValidation(report, emit, skippedValidation("token_audit", "Token 用量审计", []string{"token_audit"}, "基础连接或鉴权未通过，Token 用量审计未执行"))
		report.Metrics.ErrorClass, report.Metrics.ErrorMessage = firstProbeError(report.Checks)
		report.Metrics.LatencyMS = int64(s.currentTime().Sub(startedAt) / time.Millisecond)
		report.HasVertex = hasVertexFingerprint(report.APIBaseHost, messagesProbe.Headers)
		report.IsKiro = hasKiroFingerprint(report.APIBaseHost, messagesProbe.Headers)
		report.WrapperSignals = wrapperFingerprintSignalsForReportWithValues(report, fingerprintValuesFromHTTPProbes(gatewayProbe, messagesProbe), gatewayProbe.Headers, messagesProbe.Headers)
		appendAndEmitModelIdentity(report, emit)
		appendAndEmitWrapperFingerprint(report, emit)
		s.finalizeAndSave(ctx, report, baseURL)
		emitFinalReport(report, emit)
		return report, nil
	}

	emitProgress(report, emit, 3, "behavior")
	streamProbe := s.probeClaudeStream(checkCtx, client, baseURL, apiKey, model, probeCtx)
	report.Metrics.StreamFirstTokenMS = streamProbe.FirstTokenMS
	report.Metrics.StreamTotalLatencyMS = streamProbe.TotalLatencyMS
	report.StreamChannel = firstNonEmptyString(channelFromStreamProbe(ProviderAnthropic, host, officialHost, streamProbe.Headers), report.NonStreamChannel)
	streamingCheck := buildClaudeStreamingCheck(streamProbe, apiKey)
	appendAndEmitChecks(report, emit, streamingCheck)
	report.Metrics.TokensPerSecond = tokensPerSecond(report.Metrics.Usage, streamProbe.TotalLatencyMS)
	upsertAndEmitValidation(report, emit, validationFromExecutedChecks("behavior", "行为验证", []CheckResult{toolCheck, streamingCheck}))
	emitMetrics(report, emit)

	emitProgress(report, emit, 4, "signature")
	signatureProbe := s.probeClaudeInvalidThinkingSignature(checkCtx, client, baseURL, apiKey, model, probeCtx)
	signatureCheck := buildClaudeThinkingSignatureCheck(signatureProbe, apiKey)
	budgetProbe := s.probeClaudeThinkingBudgetViolation(checkCtx, client, baseURL, apiKey, model, probeCtx)
	budgetCheck := buildClaudeThinkingBudgetCheck(budgetProbe, apiKey)
	cacheControlProbe := s.probeClaudeCacheControlOverflow(checkCtx, client, baseURL, apiKey, model, probeCtx)
	cacheControlCheck := buildClaudeCacheControlOverflowCheck(cacheControlProbe, apiKey)
	appendAndEmitChecks(report, emit, signatureCheck, budgetCheck, cacheControlCheck)
	upsertAndEmitValidation(report, emit, validationFromExecutedChecks("signature", "签名校验", []CheckResult{usageCheck, signatureCheck, budgetCheck, cacheControlCheck}))
	emitMetrics(report, emit)

	emitProgress(report, emit, 5, "multimodal")
	multimodalProbe := s.probeClaudeMultimodal(checkCtx, client, baseURL, apiKey, model, probeCtx)
	report.Metrics.MultimodalLatencyMS = multimodalProbe.LatencyMS
	multimodalCheck := buildClaudeMultimodalCheck(multimodalProbe, apiKey)
	appendAndEmitChecks(report, emit, multimodalCheck)
	upsertAndEmitValidation(report, emit, validationFromExecutedChecks("multimodal", "多模态能力", []CheckResult{multimodalCheck}))
	emitMetrics(report, emit)

	report.HasVertex = hasVertexFingerprint(report.APIBaseHost, messagesProbe.Headers, streamProbe.Headers)
	report.IsKiro = hasKiroFingerprint(report.APIBaseHost, messagesProbe.Headers, streamProbe.Headers)
	report.WrapperSignals = wrapperFingerprintSignalsForReportWithValues(report, fingerprintValuesFromHTTPProbes(gatewayProbe, messagesProbe, signatureProbe, budgetProbe, cacheControlProbe, multimodalProbe), gatewayProbe.Headers, messagesProbe.Headers, streamProbe.Headers, signatureProbe.Headers, budgetProbe.Headers, cacheControlProbe.Headers, multimodalProbe.Headers)
	appendAndEmitModelIdentity(report, emit)
	appendAndEmitWrapperFingerprint(report, emit)

	if in.SkipTokenAudit {
		tokenAuditCheck := CheckResult{ID: "token_audit", Name: "Token 用量审计", Status: CheckStatusWarn, Score: 0, MaxScore: 15, Message: "本次请求已关闭 Token 用量审计。", Details: map[string]any{"skipped": true}}
		appendAndEmitChecks(report, emit, tokenAuditCheck)
		upsertAndEmitValidation(report, emit, validationFromExecutedChecks("token_audit", "Token 用量审计", []CheckResult{tokenAuditCheck}))
	} else if messagesProbe.StatusCode >= 200 && messagesProbe.StatusCode < 300 {
		emitProgress(report, emit, 6, "token_audit")
		billingSnapshot := s.captureBillingUsageSnapshotForAudit(checkCtx, client, ProviderAnthropic, baseURL, apiKey, options)
		report.TokenAudit = s.runClaudeTokenAudit(checkCtx, client, baseURL, apiKey, model, probeCtx, func(sample TokenAuditSample) {
			report.TokenAuditPartial = upsertTokenAuditPartial(report.TokenAuditPartial, sample)
			report.TokenAuditProgress = fmt.Sprintf("%d/%d", len(report.TokenAuditPartial), tokenAuditSamples)
			emitPublicCheckEvent(emit, PublicCheckEvent{
				Type:               PublicCheckEventTokenAuditSample,
				ReportID:           report.ReportID,
				Status:             report.Status,
				Step:               report.Step,
				StepName:           report.StepName,
				Progress:           report.Progress,
				Scores:             cloneScores(report.Scores),
				Sample:             &sample,
				TokenAuditProgress: report.TokenAuditProgress,
				TokenAuditPartial:  append([]TokenAuditSample(nil), report.TokenAuditPartial...),
			})
		})
		s.applyTokenAuditBillingMultiplierForAudit(checkCtx, client, ProviderAnthropic, baseURL, apiKey, report.TokenAudit, billingSnapshot, options)
		tokenAuditCheck := buildTokenAuditCheck(report.TokenAudit)
		appendAndEmitChecks(report, emit, tokenAuditCheck)
		upsertAndEmitValidation(report, emit, validationFromExecutedChecks("token_audit", "Token 用量审计", []CheckResult{tokenAuditCheck}))
		emitPublicCheckEvent(emit, PublicCheckEvent{
			Type:               PublicCheckEventTokenAudit,
			ReportID:           report.ReportID,
			Status:             report.Status,
			Step:               report.Step,
			StepName:           report.StepName,
			Progress:           report.Progress,
			Scores:             cloneScores(report.Scores),
			TokenAudit:         report.TokenAudit,
			TokenAuditProgress: report.TokenAuditProgress,
			TokenAuditPartial:  append([]TokenAuditSample(nil), report.TokenAuditPartial...),
		})
	} else {
		tokenAuditCheck := failCheck("token_audit", "Token 用量审计", 15, "Messages 非流式探测未通过，Token 用量审计未执行。", nil)
		appendAndEmitChecks(report, emit, tokenAuditCheck)
		upsertAndEmitValidation(report, emit, validationFromExecutedChecks("token_audit", "Token 用量审计", []CheckResult{tokenAuditCheck}))
	}

	emitProgress(report, emit, 7, "evaluate")
	report.Metrics.LatencyMS = int64(s.currentTime().Sub(startedAt) / time.Millisecond)
	s.finalizeAndSave(ctx, report, baseURL)
	emitFinalReport(report, emit)
	return report, nil
}
