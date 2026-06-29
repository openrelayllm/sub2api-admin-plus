package purity

import (
	"context"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func (s *Service) runGeminiCheck(ctx context.Context, in PublicCheckInput, emit PublicCheckEventSink, options checkRunOptions) (*PublicReport, error) {
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
		model = defaultGeminiModel
	}
	baseURL, host, officialHost, err := s.normalizeGeminiBaseURL(in.APIBaseURL, options.AllowPrivateHosts)
	if err != nil {
		return nil, err
	}
	client := s.clientForRun(options)

	startedAt := s.currentTime()
	checkCtx, cancel := context.WithTimeout(ctx, defaultCheckTimeout)
	defer cancel()

	checkedAt := startedAt.UTC()
	report := &PublicReport{
		Provider:        ProviderGemini,
		ReportID:        newReportID(ProviderGemini, host, model, checkedAt),
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
	baseURLCheck := buildGeminiBaseURLCheck(host, officialHost)
	appendAndEmitChecks(report, emit, baseURLCheck)
	gatewayProbe := httpProbe{}
	if !officialHost && shouldProbeGatewayRoot(host) {
		gatewayProbe = s.probeGatewayRoot(checkCtx, client, baseURL)
	}

	modelsProbe := s.probeGeminiModels(checkCtx, client, baseURL, apiKey)
	report.Metrics.ModelsLatencyMS = modelsProbe.LatencyMS
	modelsCheck := buildGeminiModelsCheck(modelsProbe, model)
	appendAndEmitChecks(report, emit, modelsCheck)
	upsertAndEmitValidation(report, emit, buildLLMFingerprintValidation(baseURLCheck, modelsCheck))
	emitMetrics(report, emit)
	if shouldStopAfterProbe(modelsProbe) {
		markReportProbeError(report, modelsProbe, "基础连接或鉴权未通过")
		appendAndEmitChecks(report, emit, skippedGeminiCoreChecks("基础连接或鉴权未通过，后续 Gemini 探测未执行")...)
		upsertAndEmitValidation(report, emit, skippedValidation("schema_integrity", "结构完整性", []string{"responses_schema"}, "基础连接或鉴权未通过，结构完整性未执行"))
		upsertAndEmitValidation(report, emit, skippedValidation("behavior", "行为验证", []string{"tool_call", "streaming"}, "基础连接或鉴权未通过，行为验证未执行"))
		upsertAndEmitValidation(report, emit, skippedValidation("signature", "签名校验", []string{"usage"}, "基础连接或鉴权未通过，签名校验未执行"))
		upsertAndEmitValidation(report, emit, skippedValidation("multimodal", "多模态能力", []string{"multimodal"}, "基础连接或鉴权未通过，多模态探测未执行"))
		upsertAndEmitValidation(report, emit, skippedValidation("token_audit", "Token 用量审计", []string{"token_audit"}, "基础连接或鉴权未通过，Token 用量审计未执行"))
		report.Metrics.ErrorClass, report.Metrics.ErrorMessage = firstProbeError(report.Checks)
		report.Metrics.LatencyMS = int64(s.currentTime().Sub(startedAt) / time.Millisecond)
		report.HasVertex = hasVertexFingerprint(report.APIBaseHost, modelsProbe.Headers)
		report.WrapperSignals = wrapperFingerprintSignalsForReportWithValues(report, fingerprintValuesFromHTTPProbes(gatewayProbe, modelsProbe), gatewayProbe.Headers, modelsProbe.Headers)
		appendAndEmitModelIdentity(report, emit)
		appendAndEmitWrapperFingerprint(report, emit)
		s.finalizeAndSave(ctx, report, baseURL)
		emitFinalReport(report, emit)
		return report, nil
	}

	emitProgress(report, emit, 2, "structure")
	probeModel := model
	generateProbe := s.probeGeminiGenerateContent(checkCtx, client, baseURL, apiKey, probeModel)
	modelSelection := selectGeminiProbeModel(model, modelsProbe, generateProbe)
	if modelSelection.FallbackUsed {
		probeModel = modelSelection.Model
		generateProbe = s.probeGeminiGenerateContent(checkCtx, client, baseURL, apiKey, probeModel)
	}
	report.Metrics.GenerateContentLatencyMS = generateProbe.LatencyMS
	report.Metrics.ResponsesLatencyMS = generateProbe.LatencyMS
	report.Metrics.Usage = parseGeminiUsage(generateProbe.Body)
	report.ResponseModel, report.ResponseModelSource, report.responseBodyModel, report.responseHeaderModel = geminiResponseModelFromProbe(generateProbe, probeModel)
	report.NonStreamChannel = channelFromProbe(ProviderGemini, host, officialHost, generateProbe)
	schemaCheck := buildGeminiGenerateContentCheck(generateProbe, apiKey)
	annotateGeminiFallbackCheck(&schemaCheck, modelSelection)
	usageCheck := buildGeminiUsageCheck(report.Metrics.Usage, generateProbe)
	appendAndEmitChecks(report, emit, schemaCheck, usageCheck)
	upsertAndEmitValidation(report, emit, validationFromExecutedChecks("schema_integrity", "结构完整性", []CheckResult{schemaCheck}))
	upsertAndEmitValidation(report, emit, validationFromExecutedChecks("signature", "签名校验", []CheckResult{usageCheck}))
	emitMetrics(report, emit)

	emitProgress(report, emit, 3, "behavior")
	toolProbe := s.probeGeminiToolCall(checkCtx, client, baseURL, apiKey, probeModel)
	toolCheck := buildGeminiToolCallCheck(toolProbe, apiKey)
	appendAndEmitChecks(report, emit, toolCheck)
	streamProbe := s.probeGeminiStream(checkCtx, client, baseURL, apiKey, probeModel)
	report.Metrics.StreamFirstTokenMS = streamProbe.FirstTokenMS
	report.Metrics.StreamTotalLatencyMS = streamProbe.TotalLatencyMS
	report.StreamChannel = firstNonEmptyString(channelFromStreamProbe(ProviderGemini, host, officialHost, streamProbe.Headers), report.NonStreamChannel)
	streamingCheck := buildGeminiStreamingCheck(streamProbe, apiKey)
	appendAndEmitChecks(report, emit, streamingCheck)
	report.Metrics.TokensPerSecond = tokensPerSecond(report.Metrics.Usage, streamProbe.TotalLatencyMS)
	upsertAndEmitValidation(report, emit, validationFromExecutedChecks("behavior", "行为验证", []CheckResult{toolCheck, streamingCheck}))
	emitMetrics(report, emit)

	emitProgress(report, emit, 5, "multimodal")
	multimodalProbe := s.probeGeminiMultimodal(checkCtx, client, baseURL, apiKey, probeModel)
	report.Metrics.MultimodalLatencyMS = multimodalProbe.LatencyMS
	multimodalCheck := buildGeminiMultimodalCheck(multimodalProbe, apiKey)
	appendAndEmitChecks(report, emit, multimodalCheck)
	upsertAndEmitValidation(report, emit, validationFromExecutedChecks("multimodal", "多模态能力", []CheckResult{multimodalCheck}))
	emitMetrics(report, emit)

	report.HasVertex = hasVertexFingerprint(report.APIBaseHost, modelsProbe.Headers, generateProbe.Headers, streamProbe.Headers)
	report.WrapperSignals = wrapperFingerprintSignalsForReportWithValues(report, fingerprintValuesFromHTTPProbes(gatewayProbe, modelsProbe, generateProbe, multimodalProbe), gatewayProbe.Headers, modelsProbe.Headers, generateProbe.Headers, streamProbe.Headers, multimodalProbe.Headers)
	appendAndEmitModelIdentity(report, emit)
	appendAndEmitWrapperFingerprint(report, emit)

	if in.SkipTokenAudit {
		tokenAuditCheck := CheckResult{ID: "token_audit", Name: "Token 用量审计", Status: CheckStatusWarn, Score: 0, MaxScore: 15, Message: "本次请求已关闭 Token 用量审计。", Details: map[string]any{"skipped": true}}
		appendAndEmitChecks(report, emit, tokenAuditCheck)
		upsertAndEmitValidation(report, emit, validationFromExecutedChecks("token_audit", "Token 用量审计", []CheckResult{tokenAuditCheck}))
	} else if generateProbe.StatusCode >= 200 && generateProbe.StatusCode < 300 {
		emitProgress(report, emit, 6, "token_audit")
		billingSnapshot := s.captureBillingUsageSnapshotForAudit(checkCtx, client, ProviderGemini, baseURL, apiKey, options)
		report.TokenAudit = s.runGeminiTokenAudit(checkCtx, client, baseURL, apiKey, probeModel, func(sample TokenAuditSample) {
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
		s.applyTokenAuditBillingMultiplierForAudit(checkCtx, client, ProviderGemini, baseURL, apiKey, report.TokenAudit, billingSnapshot, options)
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
		tokenAuditCheck := failCheck("token_audit", "Token 用量审计", 15, "GenerateContent 非流式探测未通过，Token 用量审计未执行。", nil)
		appendAndEmitChecks(report, emit, tokenAuditCheck)
		upsertAndEmitValidation(report, emit, validationFromExecutedChecks("token_audit", "Token 用量审计", []CheckResult{tokenAuditCheck}))
	}

	emitProgress(report, emit, 7, "evaluate")
	report.Metrics.LatencyMS = int64(s.currentTime().Sub(startedAt) / time.Millisecond)
	s.finalizeAndSave(ctx, report, baseURL)
	emitFinalReport(report, emit)
	return report, nil
}
