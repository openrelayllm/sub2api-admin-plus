package actions

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	suppliersapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/suppliers"
	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type SupplierSignal struct {
	SupplierID         int64
	Name               string
	RuntimeStatus      adminplusdomain.SupplierRuntimeStatus
	HealthStatus       adminplusdomain.SupplierHealthStatus
	BalanceCents       int64
	Currency           string
	EffectiveCostCents int64
}

type GenerateInput struct {
	Suppliers       []SupplierSignal
	BalanceEvents   []*adminplusdomain.BalanceEvent
	HealthEvents    []*adminplusdomain.HealthEvent
	KanbanEvents    []*adminplusdomain.KanbanEvent
	MinProfitMargin float64
}

type GenerateResult struct {
	Items []*adminplusdomain.ActionRecommendation `json:"items"`
	Total int                                     `json:"total"`
}

type RecommendationFilter struct {
	SupplierID int64
	Status     adminplusdomain.ActionStatus
	Severity   adminplusdomain.ActionSeverity
	Type       adminplusdomain.ActionType
	Limit      int
}

type ExecuteInput struct {
	OperatorUserID int64
	RequestPayload map[string]any
}

type Repository interface {
	CreateRecommendation(ctx context.Context, action *adminplusdomain.ActionRecommendation) (*adminplusdomain.ActionRecommendation, error)
	GetRecommendation(ctx context.Context, id int64) (*adminplusdomain.ActionRecommendation, error)
	ListRecommendations(ctx context.Context, filter RecommendationFilter) ([]*adminplusdomain.ActionRecommendation, error)
	UpdateRecommendationStatus(ctx context.Context, id int64, status adminplusdomain.ActionStatus) (*adminplusdomain.ActionRecommendation, error)
	CreateExecution(ctx context.Context, execution *adminplusdomain.ActionExecution) (*adminplusdomain.ActionExecution, error)
	ListExecutions(ctx context.Context, recommendationID int64, limit int) ([]*adminplusdomain.ActionExecution, error)
}

type SupplierStatusUpdater interface {
	UpdateStatus(ctx context.Context, id int64, in suppliersapp.UpdateSupplierStatusInput) (*adminplusdomain.Supplier, error)
}

type Service struct {
	repo            Repository
	supplierUpdater SupplierStatusUpdater
	now             func() time.Time
}

func NewService(repo Repository) *Service {
	return NewServiceWithDependencies(repo, nil)
}

func NewServiceWithDependencies(repo Repository, supplierUpdater SupplierStatusUpdater) *Service {
	return &Service{
		repo:            repo,
		supplierUpdater: supplierUpdater,
		now:             time.Now,
	}
}

func NewRuleService() *Service {
	return NewService(nil)
}

func (s *Service) Generate(ctx context.Context, in GenerateInput) (*GenerateResult, error) {
	if len(in.Suppliers) == 0 && len(in.BalanceEvents) == 0 && len(in.HealthEvents) == 0 && len(in.KanbanEvents) == 0 {
		return nil, badRequest("ACTION_SIGNALS_REQUIRED", "action signals are required")
	}
	now := s.now().UTC()
	suppliers := indexSuppliers(in.Suppliers)
	bestCandidate := findBestSwitchCandidate(in.Suppliers)
	items := make([]*adminplusdomain.ActionRecommendation, 0)
	items = append(items, s.actionsFromSuppliers(now, in.Suppliers, bestCandidate)...)
	items = append(items, s.actionsFromBalanceEvents(now, suppliers, in.BalanceEvents, bestCandidate)...)
	items = append(items, s.actionsFromHealthEvents(now, suppliers, in.HealthEvents, bestCandidate)...)
	items = append(items, s.actionsFromKanbanEvents(now, suppliers, in.KanbanEvents, bestCandidate)...)
	sort.SliceStable(items, func(i, j int) bool {
		return severityRank(items[i].Severity) > severityRank(items[j].Severity)
	})
	if s.repo != nil {
		stored := make([]*adminplusdomain.ActionRecommendation, 0, len(items))
		for _, item := range items {
			created, err := s.repo.CreateRecommendation(ctx, item)
			if err != nil {
				return nil, err
			}
			stored = append(stored, created)
		}
		items = stored
	}
	return &GenerateResult{Items: items, Total: len(items)}, nil
}

func (s *Service) ListRecommendations(ctx context.Context, filter RecommendationFilter) ([]*adminplusdomain.ActionRecommendation, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("action repository is not configured")
	}
	if filter.SupplierID < 0 {
		return nil, badRequest("ACTION_SUPPLIER_ID_INVALID", "invalid supplier id")
	}
	if filter.Status != "" && !filter.Status.Valid() {
		return nil, badRequest("ACTION_STATUS_INVALID", "invalid action status")
	}
	if filter.Severity != "" && !filter.Severity.Valid() {
		return nil, badRequest("ACTION_SEVERITY_INVALID", "invalid action severity")
	}
	if filter.Type != "" && !filter.Type.Valid() {
		return nil, badRequest("ACTION_TYPE_INVALID", "invalid action type")
	}
	filter.Limit = normalizeLimit(filter.Limit)
	return s.repo.ListRecommendations(ctx, filter)
}

func (s *Service) UpdateRecommendationStatus(ctx context.Context, id int64, status adminplusdomain.ActionStatus) (*adminplusdomain.ActionRecommendation, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("action repository is not configured")
	}
	if id <= 0 {
		return nil, badRequest("ACTION_RECOMMENDATION_ID_INVALID", "invalid action recommendation id")
	}
	if !status.Valid() {
		return nil, badRequest("ACTION_STATUS_INVALID", "invalid action status")
	}
	return s.repo.UpdateRecommendationStatus(ctx, id, status)
}

func (s *Service) ExecuteApprovedRecommendation(ctx context.Context, id int64, in ExecuteInput) (*adminplusdomain.ActionExecution, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("action repository is not configured")
	}
	if id <= 0 {
		return nil, badRequest("ACTION_RECOMMENDATION_ID_INVALID", "invalid action recommendation id")
	}
	action, err := s.repo.GetRecommendation(ctx, id)
	if err != nil {
		return nil, err
	}
	if action.Status != adminplusdomain.ActionStatusApproved {
		return nil, infraerrors.New(http.StatusConflict, "ACTION_RECOMMENDATION_NOT_APPROVED", "action recommendation must be approved before execution")
	}
	execution := s.executionFromRecommendation(ctx, action, in, s.now().UTC())
	created, err := s.repo.CreateExecution(ctx, execution)
	if err != nil {
		return nil, err
	}
	if created.Status == adminplusdomain.ActionExecutionStatusSucceeded {
		if _, err := s.repo.UpdateRecommendationStatus(ctx, id, adminplusdomain.ActionStatusExecuted); err != nil {
			return nil, err
		}
	}
	return created, nil
}

func (s *Service) ListExecutions(ctx context.Context, recommendationID int64, limit int) ([]*adminplusdomain.ActionExecution, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("action repository is not configured")
	}
	if recommendationID <= 0 {
		return nil, badRequest("ACTION_RECOMMENDATION_ID_INVALID", "invalid action recommendation id")
	}
	return s.repo.ListExecutions(ctx, recommendationID, normalizeLimit(limit))
}

func (s *Service) actionsFromSuppliers(now time.Time, suppliers []SupplierSignal, bestCandidate *SupplierSignal) []*adminplusdomain.ActionRecommendation {
	items := make([]*adminplusdomain.ActionRecommendation, 0)
	for _, supplier := range suppliers {
		if supplier.SupplierID <= 0 {
			continue
		}
		if supplier.RuntimeStatus == adminplusdomain.SupplierRuntimeStatusActive && supplier.BalanceCents <= 0 {
			items = append(items, newAction(now, supplier.SupplierID, nil, adminplusdomain.ActionTypePauseSupplier, adminplusdomain.ActionSeverityCritical, "active_supplier_depleted", "Pause depleted active supplier", "Active supplier has no balance and must not receive traffic.", "prevent failed upstream calls", []string{"balance_cents=0"}))
			if bestCandidate != nil {
				items = append(items, newAction(now, supplier.SupplierID, &bestCandidate.SupplierID, adminplusdomain.ActionTypeSwitchSupplier, adminplusdomain.ActionSeverityCritical, "switch_from_depleted_supplier", "Switch away from depleted supplier", "A cheaper or available candidate exists while the active supplier is depleted.", "restore traffic with available balance", []string{"balance_cents=0", "candidate_available"}))
			}
		}
		if supplier.HealthStatus == adminplusdomain.SupplierHealthStatusCredentialInvalid {
			items = append(items, newAction(now, supplier.SupplierID, nil, adminplusdomain.ActionTypeReviewCredential, adminplusdomain.ActionSeverityCritical, "credential_invalid", "Review supplier credential", "Supplier credential is invalid and browser/API collection may fail.", "restore monitoring and routing eligibility", []string{"credential_invalid"}))
		}
	}
	return items
}

func (s *Service) actionsFromBalanceEvents(now time.Time, suppliers map[int64]SupplierSignal, events []*adminplusdomain.BalanceEvent, bestCandidate *SupplierSignal) []*adminplusdomain.ActionRecommendation {
	items := make([]*adminplusdomain.ActionRecommendation, 0, len(events))
	for _, event := range events {
		if event == nil || event.Status != adminplusdomain.BalanceEventStatusOpen {
			continue
		}
		supplier := suppliers[event.SupplierID]
		switch event.Type {
		case adminplusdomain.BalanceEventTypeDepleted:
			items = append(items, newAction(now, event.SupplierID, nil, adminplusdomain.ActionTypeRechargeSupplier, adminplusdomain.ActionSeverityCritical, "supplier_balance_depleted", "Recharge depleted supplier", "Supplier balance is depleted. It may still be monitored, but must not be selected for switching.", "restore candidate eligibility after recharge", []string{"balance_depleted"}))
			if bestCandidate != nil && supplier.RuntimeStatus == adminplusdomain.SupplierRuntimeStatusActive {
				items = append(items, newAction(now, event.SupplierID, &bestCandidate.SupplierID, adminplusdomain.ActionTypeSwitchSupplier, adminplusdomain.ActionSeverityCritical, "switch_from_depleted_supplier", "Switch to available supplier", "Active supplier balance is depleted and another eligible supplier is available.", "keep customer traffic stable", []string{"balance_depleted", "candidate_available"}))
			}
		case adminplusdomain.BalanceEventTypeLowBalance:
			items = append(items, newAction(now, event.SupplierID, nil, adminplusdomain.ActionTypeRechargeSupplier, adminplusdomain.ActionSeverityWarning, "supplier_balance_low", "Recharge low-balance supplier", "Supplier balance is below configured threshold.", "avoid emergency traffic switch", []string{"balance_low"}))
		}
	}
	return items
}

func (s *Service) actionsFromHealthEvents(now time.Time, suppliers map[int64]SupplierSignal, events []*adminplusdomain.HealthEvent, bestCandidate *SupplierSignal) []*adminplusdomain.ActionRecommendation {
	items := make([]*adminplusdomain.ActionRecommendation, 0, len(events))
	for _, event := range events {
		if event == nil || event.Status != adminplusdomain.HealthEventStatusOpen {
			continue
		}
		supplier := suppliers[event.SupplierID]
		switch event.Type {
		case adminplusdomain.HealthEventTypeRequestError:
			items = append(items, newAction(now, event.SupplierID, nil, adminplusdomain.ActionTypePauseSupplier, adminplusdomain.ActionSeverityCritical, "supplier_request_errors", "Pause failing supplier", "Supplier returned request errors and may impact customer traffic.", "stop routing traffic to failing node", []string{"request_error"}))
			if bestCandidate != nil && supplier.RuntimeStatus == adminplusdomain.SupplierRuntimeStatusActive {
				items = append(items, newAction(now, event.SupplierID, &bestCandidate.SupplierID, adminplusdomain.ActionTypeSwitchSupplier, adminplusdomain.ActionSeverityCritical, "switch_from_failing_supplier", "Switch from failing supplier", "Active supplier is failing and another eligible supplier is available.", "restore stable API responses", []string{"request_error", "candidate_available"}))
			}
		case adminplusdomain.HealthEventTypeSlowFirstToken, adminplusdomain.HealthEventTypeSlowTotal, adminplusdomain.HealthEventTypeConcurrencyFull:
			items = append(items, newAction(now, event.SupplierID, nil, adminplusdomain.ActionTypeDegradeSupplier, adminplusdomain.ActionSeverityWarning, "supplier_performance_degraded", "Degrade supplier weight", "Supplier performance is degraded or concurrency is saturated.", "reduce slow responses while preserving fallback capacity", []string{string(event.Type)}))
		}
	}
	return items
}

func (s *Service) actionsFromKanbanEvents(now time.Time, suppliers map[int64]SupplierSignal, events []*adminplusdomain.KanbanEvent, bestCandidate *SupplierSignal) []*adminplusdomain.ActionRecommendation {
	items := make([]*adminplusdomain.ActionRecommendation, 0, len(events))
	for _, event := range events {
		if event == nil || event.Status != "open" {
			continue
		}
		supplierID := supplierIDFromKanbanEvent(event)
		severity := actionSeverityFromKanbanEvent(event.Severity)
		signals := kanbanEventSignals(event)
		supplier := suppliers[supplierID]
		switch event.EventType {
		case "supply_quality_risk":
			if supplierID <= 0 {
				items = append(items, newAction(now, 0, nil, adminplusdomain.ActionTypeInvestigateProfit, severity, "kanban_supply_quality_risk", "Review supply quality risk", eventDescription(event, "Supply quality risk needs manual review."), "prevent low-quality supply from entering production", signals))
				continue
			}
			actionType := adminplusdomain.ActionTypeDegradeSupplier
			reason := "kanban_supply_quality_risk"
			title := "Degrade risky supplier"
			impact := "reduce exposure to low-quality supply while keeping fallback capacity"
			if severity == adminplusdomain.ActionSeverityCritical {
				actionType = adminplusdomain.ActionTypePauseSupplier
				reason = "kanban_supply_quality_blocked"
				title = "Pause blocked supplier"
				impact = "stop routing production traffic to failed quality source"
			}
			items = append(items, newAction(now, supplierID, nil, actionType, severity, reason, title, eventDescription(event, "Supply quality is below production standard."), impact, signals))
			items = append(items, switchActionFromKanban(now, supplier, supplierID, bestCandidate, severity, "switch_from_supply_quality_risk", "Switch from risky supplier", eventDescription(event, "Active supplier has quality risk and another candidate is available."), "restore stable traffic with a safer supplier", signals)...)
		case "acceptance_risk":
			if supplierID <= 0 {
				items = append(items, newAction(now, 0, nil, adminplusdomain.ActionTypeReviewCredential, severity, "kanban_acceptance_risk", "Review acceptance risk", eventDescription(event, "Acceptance failed or is not production-ready."), "keep unaccepted supply out of production candidates", signals))
				continue
			}
			actionType := adminplusdomain.ActionTypeReviewCredential
			reason := "kanban_acceptance_risk"
			title := "Review supplier acceptance"
			impact := "keep unaccepted supply out of production candidates"
			if severity == adminplusdomain.ActionSeverityCritical && supplier.RuntimeStatus == adminplusdomain.SupplierRuntimeStatusActive {
				actionType = adminplusdomain.ActionTypePauseSupplier
				reason = "kanban_acceptance_blocked_active_supplier"
				title = "Pause supplier with failed acceptance"
				impact = "stop routing production traffic to a supplier that failed acceptance"
			}
			items = append(items, newAction(now, supplierID, nil, actionType, severity, reason, title, eventDescription(event, "Acceptance failed or is not production-ready."), impact, signals))
			items = append(items, switchActionFromKanban(now, supplier, supplierID, bestCandidate, severity, "switch_from_acceptance_risk", "Switch from failed-acceptance supplier", eventDescription(event, "Active supplier failed acceptance and another candidate is available."), "restore traffic with an accepted supplier candidate", signals)...)
		case "cache_efficiency_risk":
			if supplierID > 0 {
				items = append(items, newAction(now, supplierID, nil, adminplusdomain.ActionTypeDegradeSupplier, severity, "kanban_cache_efficiency_risk", "Degrade low-cache supplier", eventDescription(event, "Cache efficiency risk increases real cost."), "reduce repeated-input cost caused by low cache hit rate", signals))
			} else {
				items = append(items, newAction(now, 0, nil, adminplusdomain.ActionTypeInvestigateProfit, severity, "kanban_cache_efficiency_risk", "Review cache-adjusted cost", eventDescription(event, "Cache efficiency risk increases real cost."), "correct pricing and routing assumptions for low cache hit rate", signals))
			}
		case "market_price_drop", "market_price_anomaly", "market_model_added", "market_model_removed", "market_promotion", "unprofitable_model", "pricing_recommendation":
			items = append(items, newAction(now, 0, nil, adminplusdomain.ActionTypeInvestigateProfit, severity, "kanban_pricing_review", "Review model pricing", eventDescription(event, "Market pricing or margin signal needs review."), "avoid following unsustainable prices without quality and cache evidence", signals))
		}
	}
	return items
}

func indexSuppliers(items []SupplierSignal) map[int64]SupplierSignal {
	out := make(map[int64]SupplierSignal, len(items))
	for _, item := range items {
		if item.SupplierID > 0 {
			out[item.SupplierID] = item
		}
	}
	return out
}

func findBestSwitchCandidate(items []SupplierSignal) *SupplierSignal {
	var best *SupplierSignal
	for i := range items {
		item := items[i]
		if !adminplusdomain.CanUseSupplierForSwitching(item.RuntimeStatus, item.BalanceCents) {
			continue
		}
		if item.HealthStatus != "" && item.HealthStatus != adminplusdomain.SupplierHealthStatusNormal {
			continue
		}
		if best == nil || item.EffectiveCostCents < best.EffectiveCostCents {
			cp := item
			best = &cp
		}
	}
	return best
}

func switchActionFromKanban(now time.Time, supplier SupplierSignal, supplierID int64, bestCandidate *SupplierSignal, severity adminplusdomain.ActionSeverity, reason string, title string, description string, impact string, signals []string) []*adminplusdomain.ActionRecommendation {
	if bestCandidate == nil || bestCandidate.SupplierID <= 0 || bestCandidate.SupplierID == supplierID {
		return nil
	}
	if supplier.RuntimeStatus != adminplusdomain.SupplierRuntimeStatusActive {
		return nil
	}
	if severity != adminplusdomain.ActionSeverityCritical {
		return nil
	}
	return []*adminplusdomain.ActionRecommendation{
		newAction(now, supplierID, &bestCandidate.SupplierID, adminplusdomain.ActionTypeSwitchSupplier, severity, reason, title, description, impact, append(normalizeSignals(signals), "candidate_available")),
	}
}

func supplierIDFromKanbanEvent(event *adminplusdomain.KanbanEvent) int64 {
	if event == nil {
		return 0
	}
	if event.SourceType == "supplier" && event.SourceID > 0 {
		return event.SourceID
	}
	if event.Payload != nil {
		return int64FromPayload(event.Payload, "supplier_id")
	}
	return 0
}

func int64FromPayload(payload map[string]any, key string) int64 {
	switch value := payload[key].(type) {
	case int64:
		return value
	case int:
		return int64(value)
	case float64:
		return int64(value)
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func actionSeverityFromKanbanEvent(severity string) adminplusdomain.ActionSeverity {
	switch adminplusdomain.ActionSeverity(strings.ToLower(strings.TrimSpace(severity))) {
	case adminplusdomain.ActionSeverityCritical:
		return adminplusdomain.ActionSeverityCritical
	case adminplusdomain.ActionSeverityWarning:
		return adminplusdomain.ActionSeverityWarning
	default:
		return adminplusdomain.ActionSeverityInfo
	}
}

func kanbanEventSignals(event *adminplusdomain.KanbanEvent) []string {
	if event == nil {
		return nil
	}
	signals := []string{"kanban_event", "kanban_event_id=" + strconv.FormatInt(event.ID, 10)}
	if event.EventType != "" {
		signals = append(signals, "kanban_event_type="+event.EventType)
	}
	if event.Model != "" {
		signals = append(signals, "model="+event.Model)
	}
	if event.RelatedSnapshotType != "" {
		signals = append(signals, "snapshot_type="+event.RelatedSnapshotType)
	}
	return signals
}

func eventDescription(event *adminplusdomain.KanbanEvent, fallback string) string {
	if event == nil {
		return fallback
	}
	parts := make([]string, 0, 3)
	if strings.TrimSpace(event.Title) != "" {
		parts = append(parts, strings.TrimSpace(event.Title))
	}
	if strings.TrimSpace(event.Description) != "" {
		parts = append(parts, strings.TrimSpace(event.Description))
	}
	if strings.TrimSpace(event.Recommendation) != "" {
		parts = append(parts, strings.TrimSpace(event.Recommendation))
	}
	if len(parts) == 0 {
		return fallback
	}
	return strings.Join(parts, " ")
}

func newAction(now time.Time, supplierID int64, targetSupplierID *int64, actionType adminplusdomain.ActionType, severity adminplusdomain.ActionSeverity, reason string, title string, description string, impact string, signals []string) *adminplusdomain.ActionRecommendation {
	return &adminplusdomain.ActionRecommendation{
		SupplierID:       supplierID,
		TargetSupplierID: cloneInt64(targetSupplierID),
		Type:             actionType,
		Severity:         severity,
		Status:           adminplusdomain.ActionStatusOpen,
		ReasonCode:       reason,
		Title:            title,
		Description:      description,
		ExpectedImpact:   impact,
		RequiresApproval: true,
		Signals:          normalizeSignals(signals),
		CreatedAt:        now,
	}
}

func (s *Service) executionFromRecommendation(ctx context.Context, action *adminplusdomain.ActionRecommendation, in ExecuteInput, now time.Time) *adminplusdomain.ActionExecution {
	execution := &adminplusdomain.ActionExecution{
		RecommendationID: action.ID,
		ActionType:       action.Type,
		SupplierID:       action.SupplierID,
		TargetSupplierID: cloneInt64(action.TargetSupplierID),
		RequestPayload:   clonePayload(in.RequestPayload),
		OperatorUserID:   in.OperatorUserID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	switch action.Type {
	case adminplusdomain.ActionTypeInvestigateProfit, adminplusdomain.ActionTypeReviewCredential:
		execution.Status = adminplusdomain.ActionExecutionStatusSucceeded
		execution.ResponsePayload = map[string]any{
			"mode":        "manual_workflow_recorded",
			"action_type": string(action.Type),
			"reason_code": action.ReasonCode,
		}
	case adminplusdomain.ActionTypePauseSupplier:
		s.executeSupplierStatusUpdate(ctx, execution, adminplusdomain.SupplierRuntimeStatusDisabled, adminplusdomain.SupplierHealthStatusPaused)
	case adminplusdomain.ActionTypeDegradeSupplier:
		s.executeSupplierStatusUpdate(ctx, execution, adminplusdomain.SupplierRuntimeStatusMonitorOnly, adminplusdomain.SupplierHealthStatusNormal)
	default:
		markUnsupportedExecution(execution, action.ReasonCode)
	}
	return execution
}

func (s *Service) executeSupplierStatusUpdate(ctx context.Context, execution *adminplusdomain.ActionExecution, runtimeStatus adminplusdomain.SupplierRuntimeStatus, healthStatus adminplusdomain.SupplierHealthStatus) {
	if execution.SupplierID <= 0 {
		execution.Status = adminplusdomain.ActionExecutionStatusFailed
		execution.ErrorMessage = "supplier_id is required for supplier status action"
		execution.ResponsePayload = map[string]any{
			"mode":        "supplier_status_update",
			"action_type": string(execution.ActionType),
			"error":       "supplier_id_required",
		}
		return
	}
	if s == nil || s.supplierUpdater == nil {
		markUnsupportedExecution(execution, "supplier_status_updater_missing")
		return
	}
	updated, err := s.supplierUpdater.UpdateStatus(ctx, execution.SupplierID, suppliersapp.UpdateSupplierStatusInput{
		RuntimeStatus: runtimeStatus,
		HealthStatus:  healthStatus,
	})
	if err != nil {
		execution.Status = adminplusdomain.ActionExecutionStatusFailed
		execution.ErrorMessage = err.Error()
		execution.ResponsePayload = map[string]any{
			"mode":                  "supplier_status_update",
			"action_type":           string(execution.ActionType),
			"target_runtime_status": string(runtimeStatus),
			"target_health_status":  string(healthStatus),
		}
		return
	}
	execution.Status = adminplusdomain.ActionExecutionStatusSucceeded
	execution.ResponsePayload = map[string]any{
		"mode":                  "supplier_status_update",
		"action_type":           string(execution.ActionType),
		"supplier_id":           updated.ID,
		"runtime_status":        string(updated.RuntimeStatus),
		"health_status":         string(updated.HealthStatus),
		"target_runtime_status": string(runtimeStatus),
		"target_health_status":  string(healthStatus),
	}
}

func markUnsupportedExecution(execution *adminplusdomain.ActionExecution, reasonCode string) {
	execution.Status = adminplusdomain.ActionExecutionStatusUnsupported
	execution.ErrorMessage = "automatic execution for this action type is not enabled; keep manual approval and execute in the owning system"
	execution.ResponsePayload = map[string]any{
		"mode":        "unsupported_without_routing_executor",
		"action_type": string(execution.ActionType),
		"reason_code": reasonCode,
	}
}

func clonePayload(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func normalizeSignals(signals []string) []string {
	out := make([]string, 0, len(signals))
	for _, signal := range signals {
		v := strings.TrimSpace(signal)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func cloneInt64(in *int64) *int64 {
	if in == nil {
		return nil
	}
	v := *in
	return &v
}

func severityRank(severity adminplusdomain.ActionSeverity) int {
	switch severity {
	case adminplusdomain.ActionSeverityCritical:
		return 3
	case adminplusdomain.ActionSeverityWarning:
		return 2
	case adminplusdomain.ActionSeverityInfo:
		return 1
	default:
		return 0
	}
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 200
	}
	if limit > 1000 {
		return 1000
	}
	return limit
}

func badRequest(reason string, message string) error {
	return infraerrors.New(http.StatusBadRequest, reason, message)
}

func internalError(message string) error {
	return infraerrors.New(http.StatusInternalServerError, "ADMIN_PLUS_INTERNAL_ERROR", message)
}
