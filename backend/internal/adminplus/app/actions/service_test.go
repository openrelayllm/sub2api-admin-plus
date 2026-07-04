package actions

import (
	"context"
	"testing"

	suppliersapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/suppliers"
	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/stretchr/testify/require"
)

func TestServiceGenerateSwitchesFromDepletedActiveSupplierToEligibleCandidate(t *testing.T) {
	svc := NewRuleService()

	result, err := svc.Generate(context.Background(), GenerateInput{
		Suppliers: []SupplierSignal{
			{
				SupplierID:         1,
				Name:               "active",
				RuntimeStatus:      adminplusdomain.SupplierRuntimeStatusActive,
				HealthStatus:       adminplusdomain.SupplierHealthStatusNormal,
				BalanceCents:       0,
				EffectiveCostCents: 100,
			},
			{
				SupplierID:         2,
				Name:               "candidate",
				RuntimeStatus:      adminplusdomain.SupplierRuntimeStatusCandidate,
				HealthStatus:       adminplusdomain.SupplierHealthStatusNormal,
				BalanceCents:       5000,
				EffectiveCostCents: 80,
			},
		},
	})

	require.NoError(t, err)
	requireAction(t, result.Items, adminplusdomain.ActionTypePauseSupplier, "active_supplier_depleted")
	switchAction := requireAction(t, result.Items, adminplusdomain.ActionTypeSwitchSupplier, "switch_from_depleted_supplier")
	require.NotNil(t, switchAction.TargetSupplierID)
	require.Equal(t, int64(2), *switchAction.TargetSupplierID)
}

func TestServiceGeneratePausesAndSwitchesFromFailingActiveSupplier(t *testing.T) {
	svc := NewRuleService()

	result, err := svc.Generate(context.Background(), GenerateInput{
		Suppliers: []SupplierSignal{
			{
				SupplierID:         1,
				RuntimeStatus:      adminplusdomain.SupplierRuntimeStatusActive,
				HealthStatus:       adminplusdomain.SupplierHealthStatusNormal,
				BalanceCents:       5000,
				EffectiveCostCents: 100,
			},
			{
				SupplierID:         2,
				RuntimeStatus:      adminplusdomain.SupplierRuntimeStatusCandidate,
				HealthStatus:       adminplusdomain.SupplierHealthStatusNormal,
				BalanceCents:       5000,
				EffectiveCostCents: 90,
			},
		},
		HealthEvents: []*adminplusdomain.HealthEvent{
			{
				SupplierID: 1,
				Type:       adminplusdomain.HealthEventTypeRequestError,
				Status:     adminplusdomain.HealthEventStatusOpen,
			},
		},
	})

	require.NoError(t, err)
	requireAction(t, result.Items, adminplusdomain.ActionTypePauseSupplier, "supplier_request_errors")
	requireAction(t, result.Items, adminplusdomain.ActionTypeSwitchSupplier, "switch_from_failing_supplier")
}

func TestServiceGenerateDegradesSlowOrSaturatedSupplier(t *testing.T) {
	svc := NewRuleService()

	result, err := svc.Generate(context.Background(), GenerateInput{
		Suppliers: []SupplierSignal{
			{
				SupplierID:    1,
				RuntimeStatus: adminplusdomain.SupplierRuntimeStatusActive,
				HealthStatus:  adminplusdomain.SupplierHealthStatusNormal,
				BalanceCents:  5000,
			},
		},
		HealthEvents: []*adminplusdomain.HealthEvent{
			{
				SupplierID: 1,
				Type:       adminplusdomain.HealthEventTypeSlowFirstToken,
				Status:     adminplusdomain.HealthEventStatusOpen,
			},
		},
	})

	require.NoError(t, err)
	requireAction(t, result.Items, adminplusdomain.ActionTypeDegradeSupplier, "supplier_performance_degraded")
}

func TestServiceGeneratePausesAndSwitchesFromCriticalKanbanSupplierRisk(t *testing.T) {
	svc := NewRuleService()

	result, err := svc.Generate(context.Background(), GenerateInput{
		Suppliers: []SupplierSignal{
			{
				SupplierID:         1,
				RuntimeStatus:      adminplusdomain.SupplierRuntimeStatusActive,
				HealthStatus:       adminplusdomain.SupplierHealthStatusNormal,
				BalanceCents:       5000,
				EffectiveCostCents: 100,
			},
			{
				SupplierID:         2,
				RuntimeStatus:      adminplusdomain.SupplierRuntimeStatusCandidate,
				HealthStatus:       adminplusdomain.SupplierHealthStatusNormal,
				BalanceCents:       5000,
				EffectiveCostCents: 90,
			},
		},
		KanbanEvents: []*adminplusdomain.KanbanEvent{
			{
				ID:         31,
				EventType:  "supply_quality_risk",
				Severity:   "critical",
				Status:     "open",
				Model:      "gpt-risk",
				SourceType: "supplier",
				SourceID:   1,
				Title:      "供应质量风险",
			},
		},
	})

	require.NoError(t, err)
	requireAction(t, result.Items, adminplusdomain.ActionTypePauseSupplier, "kanban_supply_quality_blocked")
	switchAction := requireAction(t, result.Items, adminplusdomain.ActionTypeSwitchSupplier, "switch_from_supply_quality_risk")
	require.NotNil(t, switchAction.TargetSupplierID)
	require.Equal(t, int64(2), *switchAction.TargetSupplierID)
	require.Contains(t, switchAction.Signals, "kanban_event_id=31")
}

func TestServiceGeneratePricingReviewFromKanbanEventWithoutSuppliers(t *testing.T) {
	svc := NewRuleService()

	result, err := svc.Generate(context.Background(), GenerateInput{
		KanbanEvents: []*adminplusdomain.KanbanEvent{
			{
				ID:        41,
				EventType: "market_price_drop",
				Severity:  "warning",
				Status:    "open",
				Model:     "gpt-price",
				Title:     "市场价格下降",
			},
		},
	})

	require.NoError(t, err)
	action := requireAction(t, result.Items, adminplusdomain.ActionTypeInvestigateProfit, "kanban_pricing_review")
	require.Equal(t, int64(0), action.SupplierID)
	require.Contains(t, action.Signals, "model=gpt-price")
}

func TestServiceExecuteApprovedRecommendationRecordsSucceededReceipt(t *testing.T) {
	repo := newFakeActionRepository(&adminplusdomain.ActionRecommendation{
		ID:         1,
		Type:       adminplusdomain.ActionTypeReviewCredential,
		Status:     adminplusdomain.ActionStatusApproved,
		ReasonCode: "credential_invalid",
	})
	svc := NewService(repo)

	execution, err := svc.ExecuteApprovedRecommendation(context.Background(), 1, ExecuteInput{
		OperatorUserID: 99,
		RequestPayload: map[string]any{"note": "reviewed"},
	})

	require.NoError(t, err)
	require.Equal(t, adminplusdomain.ActionExecutionStatusSucceeded, execution.Status)
	require.Equal(t, int64(99), execution.OperatorUserID)
	require.Equal(t, adminplusdomain.ActionStatusExecuted, repo.actions[1].Status)
	require.Len(t, repo.executions, 1)
}

func TestServiceExecuteApprovedRecommendationKeepsUnsupportedRoutingActionApproved(t *testing.T) {
	repo := newFakeActionRepository(&adminplusdomain.ActionRecommendation{
		ID:         2,
		Type:       adminplusdomain.ActionTypeSwitchSupplier,
		Status:     adminplusdomain.ActionStatusApproved,
		SupplierID: 7,
	})
	svc := NewService(repo)

	execution, err := svc.ExecuteApprovedRecommendation(context.Background(), 2, ExecuteInput{})

	require.NoError(t, err)
	require.Equal(t, adminplusdomain.ActionExecutionStatusUnsupported, execution.Status)
	require.Equal(t, adminplusdomain.ActionStatusApproved, repo.actions[2].Status)
	require.Contains(t, execution.ResponsePayload["mode"], "unsupported")
}

func TestServiceExecuteApprovedRecommendationUpdatesSupplierStatus(t *testing.T) {
	cases := []struct {
		name              string
		actionType        adminplusdomain.ActionType
		wantRuntimeStatus adminplusdomain.SupplierRuntimeStatus
		wantHealthStatus  adminplusdomain.SupplierHealthStatus
	}{
		{
			name:              "pause",
			actionType:        adminplusdomain.ActionTypePauseSupplier,
			wantRuntimeStatus: adminplusdomain.SupplierRuntimeStatusDisabled,
			wantHealthStatus:  adminplusdomain.SupplierHealthStatusPaused,
		},
		{
			name:              "degrade",
			actionType:        adminplusdomain.ActionTypeDegradeSupplier,
			wantRuntimeStatus: adminplusdomain.SupplierRuntimeStatusMonitorOnly,
			wantHealthStatus:  adminplusdomain.SupplierHealthStatusNormal,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeActionRepository(&adminplusdomain.ActionRecommendation{
				ID:         3,
				Type:       tc.actionType,
				Status:     adminplusdomain.ActionStatusApproved,
				SupplierID: 7,
				ReasonCode: string(tc.actionType),
			})
			updater := &fakeSupplierStatusUpdater{}
			svc := NewServiceWithDependencies(repo, updater)

			execution, err := svc.ExecuteApprovedRecommendation(context.Background(), 3, ExecuteInput{})

			require.NoError(t, err)
			require.Equal(t, adminplusdomain.ActionExecutionStatusSucceeded, execution.Status)
			require.Equal(t, 1, updater.calls)
			require.Equal(t, int64(7), updater.id)
			require.Equal(t, tc.wantRuntimeStatus, updater.input.RuntimeStatus)
			require.Equal(t, tc.wantHealthStatus, updater.input.HealthStatus)
			require.Equal(t, adminplusdomain.ActionStatusExecuted, repo.actions[3].Status)
			require.Equal(t, "supplier_status_update", execution.ResponsePayload["mode"])
		})
	}
}

func requireAction(t *testing.T, items []*adminplusdomain.ActionRecommendation, actionType adminplusdomain.ActionType, reason string) *adminplusdomain.ActionRecommendation {
	t.Helper()
	for _, item := range items {
		if item.Type == actionType && item.ReasonCode == reason {
			return item
		}
	}
	require.Failf(t, "missing action", "type=%s reason=%s items=%v", actionType, reason, items)
	return nil
}

type fakeActionRepository struct {
	actions    map[int64]*adminplusdomain.ActionRecommendation
	executions []*adminplusdomain.ActionExecution
}

func newFakeActionRepository(actions ...*adminplusdomain.ActionRecommendation) *fakeActionRepository {
	repo := &fakeActionRepository{actions: map[int64]*adminplusdomain.ActionRecommendation{}}
	for _, action := range actions {
		repo.actions[action.ID] = action
	}
	return repo
}

func (r *fakeActionRepository) CreateRecommendation(_ context.Context, action *adminplusdomain.ActionRecommendation) (*adminplusdomain.ActionRecommendation, error) {
	if action.ID == 0 {
		action.ID = int64(len(r.actions) + 1)
	}
	r.actions[action.ID] = action
	return action, nil
}

func (r *fakeActionRepository) GetRecommendation(_ context.Context, id int64) (*adminplusdomain.ActionRecommendation, error) {
	return r.actions[id], nil
}

func (r *fakeActionRepository) ListRecommendations(_ context.Context, _ RecommendationFilter) ([]*adminplusdomain.ActionRecommendation, error) {
	items := make([]*adminplusdomain.ActionRecommendation, 0, len(r.actions))
	for _, action := range r.actions {
		items = append(items, action)
	}
	return items, nil
}

func (r *fakeActionRepository) UpdateRecommendationStatus(_ context.Context, id int64, status adminplusdomain.ActionStatus) (*adminplusdomain.ActionRecommendation, error) {
	action := r.actions[id]
	action.Status = status
	return action, nil
}

func (r *fakeActionRepository) CreateExecution(_ context.Context, execution *adminplusdomain.ActionExecution) (*adminplusdomain.ActionExecution, error) {
	if execution.ID == 0 {
		execution.ID = int64(len(r.executions) + 1)
	}
	r.executions = append(r.executions, execution)
	return execution, nil
}

func (r *fakeActionRepository) ListExecutions(_ context.Context, recommendationID int64, _ int) ([]*adminplusdomain.ActionExecution, error) {
	items := make([]*adminplusdomain.ActionExecution, 0, len(r.executions))
	for _, execution := range r.executions {
		if execution.RecommendationID == recommendationID {
			items = append(items, execution)
		}
	}
	return items, nil
}

type fakeSupplierStatusUpdater struct {
	calls int
	id    int64
	input suppliersapp.UpdateSupplierStatusInput
}

func (u *fakeSupplierStatusUpdater) UpdateStatus(_ context.Context, id int64, in suppliersapp.UpdateSupplierStatusInput) (*adminplusdomain.Supplier, error) {
	u.calls++
	u.id = id
	u.input = in
	return &adminplusdomain.Supplier{
		ID:            id,
		RuntimeStatus: in.RuntimeStatus,
		HealthStatus:  in.HealthStatus,
	}, nil
}
