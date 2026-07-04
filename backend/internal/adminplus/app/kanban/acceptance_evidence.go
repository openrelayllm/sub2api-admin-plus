package kanban

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
)

const acceptanceEvidenceGenerator = "acceptance_from_scheduler_evidence_v1"

func (s *Service) OnSchedulerRunStatusRefreshed(ctx context.Context, runID string) error {
	_, err := s.refreshAcceptanceReportFromEvidenceRun(ctx, runID, true)
	return err
}

func (s *Service) RefreshAcceptanceReportFromEvidenceRun(ctx context.Context, runID string) (*adminplusdomain.AcceptanceReport, error) {
	return s.refreshAcceptanceReportFromEvidenceRun(ctx, runID, false)
}

func (s *Service) refreshAcceptanceReportFromEvidenceRun(ctx context.Context, runID string, ignoreNotReady bool) (*adminplusdomain.AcceptanceReport, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("kanban service is not configured")
	}
	if s.evidenceRunReader == nil {
		return nil, internalError("acceptance evidence run reader is not configured")
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, badRequest("KANBAN_ACCEPTANCE_EVIDENCE_RUN_ID_REQUIRED", "scheduler run id is required")
	}
	detail, err := s.evidenceRunReader.GetRunDetail(ctx, runID)
	if err != nil {
		return nil, err
	}
	if !isAcceptanceEvidenceRun(detail) {
		if ignoreNotReady {
			return nil, nil
		}
		return nil, badRequest("KANBAN_ACCEPTANCE_EVIDENCE_RUN_INVALID", "scheduler run is not a kanban acceptance evidence run")
	}
	if !acceptanceEvidenceRunFinished(detail) {
		if ignoreNotReady {
			return nil, nil
		}
		return nil, badRequest("KANBAN_ACCEPTANCE_EVIDENCE_RUN_NOT_FINISHED", "scheduler run has unfinished evidence steps")
	}
	evidence, err := newAcceptanceSchedulerEvidence(detail, s.now().UTC())
	if err != nil {
		return nil, err
	}
	return s.recordAcceptanceSchedulerEvidence(ctx, evidence)
}

func (s *Service) recordAcceptanceSchedulerEvidence(ctx context.Context, evidence acceptanceSchedulerEvidence) (*adminplusdomain.AcceptanceReport, error) {
	quality, err := s.RecordSupplyQuality(ctx, evidence.supplyQualityInput())
	if err != nil {
		return nil, err
	}
	var cache *adminplusdomain.CacheEfficiencySnapshot
	if cacheInput, ok := evidence.cacheEfficiencyInput(); ok {
		cache, err = s.RecordCacheEfficiency(ctx, cacheInput)
		if err != nil {
			return nil, err
		}
	}
	reportInput := evidence.acceptanceReportInput(quality, cache)
	return s.RecordAcceptanceReport(ctx, reportInput)
}

type acceptanceSchedulerEvidence struct {
	run        adminplusdomain.SchedulerRunSummary
	steps      []adminplusdomain.SchedulerStepRecord
	stepsByKey map[adminplusdomain.ExtensionTaskType]*adminplusdomain.SchedulerStepRecord
	supplyType string
	supplierID int64
	accountID  int64
	model      string
	observedAt time.Time
}

func newAcceptanceSchedulerEvidence(detail *adminplusdomain.SchedulerRunDetail, fallbackNow time.Time) (acceptanceSchedulerEvidence, error) {
	if detail == nil {
		return acceptanceSchedulerEvidence{}, badRequest("KANBAN_ACCEPTANCE_EVIDENCE_RUN_INVALID", "scheduler run detail is empty")
	}
	evidence := acceptanceSchedulerEvidence{
		run:        detail.Run,
		steps:      detail.Steps,
		stepsByKey: map[adminplusdomain.ExtensionTaskType]*adminplusdomain.SchedulerStepRecord{},
		supplyType: "supplier",
		observedAt: fallbackNow,
	}
	if detail.Run.FinishedAt != nil && !detail.Run.FinishedAt.IsZero() {
		evidence.observedAt = detail.Run.FinishedAt.UTC()
	}
	evidence.supplyType = firstNonEmptyString(
		acceptanceStringValue(detail.Run.RequestSnapshot, "supply_type"),
		evidence.supplyType,
	)
	evidence.model = acceptanceStringValue(detail.Run.RequestSnapshot, "model")
	evidence.accountID = acceptanceInt64Value(detail.Run.RequestSnapshot, "local_sub2api_account_id")
	supplierSet := map[int64]struct{}{}
	for idx := range detail.Steps {
		step := &detail.Steps[idx]
		if step.SupplierID > 0 {
			supplierSet[step.SupplierID] = struct{}{}
			if evidence.supplierID == 0 {
				evidence.supplierID = step.SupplierID
			}
		}
		if evidence.model == "" {
			evidence.model = firstNonEmptyString(
				acceptanceStringValue(step.RequestSnapshot, "model"),
				acceptanceStringValue(step.ResultSnapshot, "model"),
			)
		}
		if evidence.accountID == 0 {
			evidence.accountID = firstPositiveID(
				acceptanceInt64Value(step.RequestSnapshot, "local_sub2api_account_id"),
				acceptanceInt64Value(step.ResultSnapshot, "local_sub2api_account_id"),
			)
		}
		current := evidence.stepsByKey[step.TaskType]
		if current == nil || step.ID > current.ID {
			evidence.stepsByKey[step.TaskType] = step
		}
	}
	if len(supplierSet) > 1 {
		return acceptanceSchedulerEvidence{}, badRequest("KANBAN_ACCEPTANCE_EVIDENCE_SUPPLIER_AMBIGUOUS", "scheduler run contains more than one supplier")
	}
	if evidence.supplierID == 0 && evidence.accountID == 0 && evidence.model == "" {
		return acceptanceSchedulerEvidence{}, badRequest("KANBAN_ACCEPTANCE_TARGET_REQUIRED", "scheduler run has no supplier, account or model evidence")
	}
	evidence.supplyType = normalizeSupplyType(evidence.supplyType)
	return evidence, nil
}

func (e acceptanceSchedulerEvidence) supplyQualityInput() SupplyQualityInput {
	availabilityRatio, errorRatio := e.stepRatios()
	cacheRatio, _ := e.cacheHitRatio()
	observedAt := e.observedAt
	return SupplyQualityInput{
		SupplyType:            e.supplyType,
		SupplierID:            e.supplierID,
		LocalSub2APIAccountID: e.accountID,
		Model:                 e.model,
		AvailabilityRatio:     availabilityRatio,
		ErrorRatio:            errorRatio,
		CacheHitRatio:         cacheRatio,
		PurityScore:           e.purityScore(),
		UsageTrustScore:       acceptanceScoreFromStatus(e.usageMeteringStatus()),
		BalanceRiskScore:      acceptanceRiskScoreFromStatus(e.balanceStatus()),
		ConcurrencyScore:      acceptanceScoreFromStatus(e.concurrencyStatus()),
		Notes:                 fmt.Sprintf("由调度验收 run %s 自动回填。", e.run.ID),
		ObservedAt:            &observedAt,
		RawPayload:            e.rawPayload(),
	}
}

func (e acceptanceSchedulerEvidence) cacheEfficiencyInput() (CacheEfficiencyInput, bool) {
	inputTokens, cachedTokens, outputTokens, ok := e.purityUsageTokens()
	if !ok || e.model == "" {
		return CacheEfficiencyInput{}, false
	}
	ratio, hasRatio := acceptanceCacheHitRatio(inputTokens, cachedTokens)
	if !hasRatio {
		return CacheEfficiencyInput{}, false
	}
	observedAt := e.observedAt
	duplicateInputTokens := inputTokens - cachedTokens
	if duplicateInputTokens < 0 {
		duplicateInputTokens = 0
	}
	return CacheEfficiencyInput{
		SupplyType:            e.supplyType,
		SupplierID:            e.supplierID,
		LocalSub2APIAccountID: e.accountID,
		Model:                 e.model,
		RoutingStrategy:       firstNonEmptyString(e.requestValue("routing_strategy"), "unknown"),
		StickyScope:           firstNonEmptyString(e.requestValue("sticky_scope"), "none"),
		SampleRequests:        e.puritySampleRequests(),
		CacheReadTokens:       cachedTokens,
		InputTokens:           inputTokens,
		OutputTokens:          outputTokens,
		CacheHitRatio:         &ratio,
		DuplicateInputTokens:  duplicateInputTokens,
		Notes:                 fmt.Sprintf("由调度验收 run %s 的纯度检测 usage 自动回填；路由策略需独立缓存审计确认。", e.run.ID),
		ObservedAt:            &observedAt,
		RawPayload:            e.rawPayload(),
	}, true
}

func (e acceptanceSchedulerEvidence) acceptanceReportInput(quality *adminplusdomain.SupplyQualitySnapshot, cache *adminplusdomain.CacheEfficiencySnapshot) AcceptanceReportInput {
	observedAt := e.observedAt
	input := AcceptanceReportInput{
		SupplyType:            e.supplyType,
		SupplierID:            e.supplierID,
		LocalSub2APIAccountID: e.accountID,
		Model:                 e.model,
		ConnectivityStatus:    e.connectivityStatus(),
		ModelListStatus:       e.modelListStatus(),
		PurityStatus:          e.purityStatus(),
		TrialCallStatus:       e.trialCallStatus(),
		UsageMeteringStatus:   e.usageMeteringStatus(),
		CacheAuditStatus:      e.cacheAuditStatus(),
		BalanceStatus:         e.balanceStatus(),
		ConcurrencyStatus:     e.concurrencyStatus(),
		ReportPayload:         e.rawPayload(),
		ObservedAt:            &observedAt,
	}
	input.ReportPayload["quality_snapshot_id"] = snapshotID(quality)
	input.ReportPayload["cache_snapshot_id"] = snapshotID(cache)
	failureReasons := make([]string, 0)
	collectAcceptanceFailureReasons(&failureReasons, input)
	input.FailureReason = strings.Join(failureReasons, "; ")
	return input
}

func (e acceptanceSchedulerEvidence) connectivityStatus() string {
	return acceptanceStatusFromStep(e.stepsByKey[adminplusdomain.ExtensionTaskTypeFetchHealth])
}

func (e acceptanceSchedulerEvidence) modelListStatus() string {
	return acceptanceStatusFromStep(e.stepsByKey[adminplusdomain.ExtensionTaskTypeCheckChannels])
}

func (e acceptanceSchedulerEvidence) trialCallStatus() string {
	return acceptanceStatusFromStep(e.stepsByKey[adminplusdomain.ExtensionTaskTypeCheckChannels])
}

func (e acceptanceSchedulerEvidence) balanceStatus() string {
	return acceptanceStatusFromStep(e.stepsByKey[adminplusdomain.ExtensionTaskTypeFetchBalance])
}

func (e acceptanceSchedulerEvidence) concurrencyStatus() string {
	return acceptanceStatusFromStep(e.stepsByKey[adminplusdomain.ExtensionTaskTypeCheckChannels])
}

func (e acceptanceSchedulerEvidence) purityStatus() string {
	step := e.stepsByKey[adminplusdomain.ExtensionTaskTypeRunPurityCheck]
	if step == nil {
		return "unknown"
	}
	if status := acceptanceStatusFromStep(step); status == "fail" || status == "warn" {
		return status
	}
	if score, ok := acceptanceFloat64Value(step.ResultSnapshot, "score"); ok && score > 0 {
		return statusFromScore(&score, 50, 80)
	}
	for _, value := range []string{
		acceptanceStringValue(step.ResultSnapshot, "verdict"),
		acceptanceStringValue(step.ResultSnapshot, "status"),
	} {
		switch acceptanceSemanticStatus(value) {
		case "pass", "warn", "fail":
			return acceptanceSemanticStatus(value)
		}
	}
	return acceptanceStatusFromStep(step)
}

func (e acceptanceSchedulerEvidence) usageMeteringStatus() string {
	purityStep := e.stepsByKey[adminplusdomain.ExtensionTaskTypeRunPurityCheck]
	if purityStep != nil {
		audit := acceptanceSemanticStatus(acceptanceStringValue(purityStep.ResultSnapshot, "token_audit_status"))
		if audit == "pass" || audit == "warn" || audit == "fail" {
			return audit
		}
		if _, _, _, ok := e.purityUsageTokens(); ok {
			return "pass"
		}
	}
	return acceptanceStatusFromStep(e.stepsByKey[adminplusdomain.ExtensionTaskTypeFetchUsageCosts])
}

func (e acceptanceSchedulerEvidence) cacheAuditStatus() string {
	ratio, ok := e.cacheHitRatio()
	if !ok {
		return "unknown"
	}
	switch {
	case ratio < defaultCacheRiskHitRatio:
		return "fail"
	case ratio < 0.65:
		return "warn"
	default:
		return "pass"
	}
}

func (e acceptanceSchedulerEvidence) purityScore() float64 {
	step := e.stepsByKey[adminplusdomain.ExtensionTaskTypeRunPurityCheck]
	if step != nil {
		if score, ok := acceptanceFloat64Value(step.ResultSnapshot, "score"); ok && score >= 0 {
			if score > 100 {
				return 100
			}
			return score
		}
	}
	return acceptanceScoreFromStatus(e.purityStatus())
}

func (e acceptanceSchedulerEvidence) stepRatios() (float64, float64) {
	total := 0
	succeeded := 0
	failed := 0
	for _, step := range e.steps {
		if !acceptanceTerminalStepStatus(step.Status) {
			continue
		}
		total++
		switch step.Status {
		case "succeeded":
			succeeded++
		case "retryable_failed", "manual_required", "dead":
			failed++
		}
	}
	if total == 0 {
		return 0, 0
	}
	return float64(succeeded) / float64(total), float64(failed) / float64(total)
}

func (e acceptanceSchedulerEvidence) purityUsageTokens() (inputTokens int64, cachedTokens int64, outputTokens int64, ok bool) {
	step := e.stepsByKey[adminplusdomain.ExtensionTaskTypeRunPurityCheck]
	if step == nil || len(step.ResultSnapshot) == 0 {
		return 0, 0, 0, false
	}
	inputTokens = acceptanceInt64Value(step.ResultSnapshot, "input_tokens")
	cachedTokens = acceptanceInt64Value(step.ResultSnapshot, "cached_tokens")
	outputTokens = acceptanceInt64Value(step.ResultSnapshot, "output_tokens")
	return inputTokens, cachedTokens, outputTokens, inputTokens > 0 || cachedTokens > 0 || outputTokens > 0
}

func (e acceptanceSchedulerEvidence) cacheHitRatio() (float64, bool) {
	inputTokens, cachedTokens, _, ok := e.purityUsageTokens()
	if !ok {
		return 0, false
	}
	return acceptanceCacheHitRatio(inputTokens, cachedTokens)
}

func (e acceptanceSchedulerEvidence) puritySampleRequests() int {
	step := e.stepsByKey[adminplusdomain.ExtensionTaskTypeRunPurityCheck]
	if step == nil {
		return 1
	}
	total := acceptanceInt64Value(step.ResultSnapshot, "total")
	if total <= 0 {
		return 1
	}
	if total > 1000000 {
		return 1000000
	}
	return int(total)
}

func (e acceptanceSchedulerEvidence) requestValue(key string) string {
	if value := acceptanceStringValue(e.run.RequestSnapshot, key); value != "" {
		return value
	}
	for _, step := range e.steps {
		if value := acceptanceStringValue(step.RequestSnapshot, key); value != "" {
			return value
		}
	}
	return ""
}

func (e acceptanceSchedulerEvidence) rawPayload() map[string]any {
	payload := map[string]any{
		"generator":                        acceptanceEvidenceGenerator,
		"evidence_scheduler_run_id":        e.run.ID,
		"evidence_scheduler_status":        e.run.Status,
		"evidence_scheduler_trigger_type":  e.run.TriggerType,
		"evidence_scheduler_task_type":     e.run.TaskType,
		"evidence_scheduler_total_steps":   e.run.TotalSteps,
		"evidence_scheduler_failed_steps":  e.run.FailedSteps,
		"evidence_scheduler_skipped_steps": e.run.SkippedSteps,
		"evidence_scheduler_finished_at":   e.observedAt.UTC().Format(time.RFC3339),
		"step_statuses":                    e.stepStatuses(),
	}
	if step := e.stepsByKey[adminplusdomain.ExtensionTaskTypeRunPurityCheck]; step != nil {
		payload["purity_report_id"] = acceptanceStringValue(step.ResultSnapshot, "report_id")
		payload["purity_verdict"] = acceptanceStringValue(step.ResultSnapshot, "verdict")
		payload["token_audit_status"] = acceptanceStringValue(step.ResultSnapshot, "token_audit_status")
	}
	if ratio, ok := e.cacheHitRatio(); ok {
		payload["cache_hit_ratio"] = ratio
	}
	return payload
}

func (e acceptanceSchedulerEvidence) stepStatuses() map[string]string {
	out := make(map[string]string, len(e.steps))
	for _, step := range e.steps {
		out[string(step.TaskType)] = step.Status
	}
	return out
}

func isAcceptanceEvidenceRun(detail *adminplusdomain.SchedulerRunDetail) bool {
	if detail == nil {
		return false
	}
	return detail.Run.TriggerType == "kanban_acceptance" || strings.HasPrefix(detail.Run.ID, "kanban_acceptance-")
}

func acceptanceEvidenceRunFinished(detail *adminplusdomain.SchedulerRunDetail) bool {
	if detail == nil {
		return false
	}
	switch detail.Run.Status {
	case "succeeded", "partial_succeeded", "retryable_failed", "manual_required", "dead", "skipped", "cancelled":
		return true
	case "queued", "running":
		return false
	}
	for _, step := range detail.Steps {
		if !acceptanceTerminalStepStatus(step.Status) {
			return false
		}
	}
	return len(detail.Steps) > 0
}

func acceptanceTerminalStepStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "succeeded", "skipped", "cancelled", "retryable_failed", "manual_required", "dead":
		return true
	default:
		return false
	}
}

func acceptanceStatusFromStep(step *adminplusdomain.SchedulerStepRecord) string {
	if step == nil {
		return "unknown"
	}
	switch strings.ToLower(strings.TrimSpace(step.Status)) {
	case "succeeded":
		return "pass"
	case "cancelled":
		return "warn"
	case "retryable_failed", "manual_required", "dead":
		return "fail"
	default:
		return "unknown"
	}
}

func acceptanceSemanticStatus(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "pass", "passed", "ok", "success", "succeeded", "completed", "trusted", "clean", "official", "compatible":
		return "pass"
	case "warn", "warning", "partial", "suspect", "mixed", "degraded":
		return "warn"
	case "fail", "failed", "error", "invalid", "untrusted", "blocked", "mismatch", "fake":
		return "fail"
	default:
		return ""
	}
}

func acceptanceScoreFromStatus(status string) float64 {
	switch status {
	case "pass":
		return 90
	case "warn":
		return 60
	case "fail":
		return 20
	default:
		return 0
	}
}

func acceptanceRiskScoreFromStatus(status string) float64 {
	switch status {
	case "pass":
		return 10
	case "warn":
		return 50
	case "fail":
		return 90
	default:
		return 0
	}
}

func acceptanceCacheHitRatio(inputTokens int64, cachedTokens int64) (float64, bool) {
	if inputTokens <= 0 {
		return 0, false
	}
	ratio := float64(cachedTokens) / float64(inputTokens)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return ratio, true
}

func acceptanceInt64Value(snapshot map[string]any, key string) int64 {
	value := snapshot[key]
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case json.Number:
		parsed, _ := v.Int64()
		return parsed
	case string:
		parsed, _ := json.Number(strings.TrimSpace(v)).Int64()
		return parsed
	default:
		return 0
	}
}

func acceptanceFloat64Value(snapshot map[string]any, key string) (float64, bool) {
	value := snapshot[key]
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int64:
		return float64(v), true
	case int:
		return float64(v), true
	case json.Number:
		parsed, err := v.Float64()
		return parsed, err == nil
	case string:
		parsed, err := json.Number(strings.TrimSpace(v)).Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func acceptanceStringValue(snapshot map[string]any, key string) string {
	if len(snapshot) == 0 {
		return ""
	}
	value := snapshot[key]
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
