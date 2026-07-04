package kanban

import (
	"context"
	"strings"

	schedulerapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/scheduler"
	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
)

type AcceptanceEvidenceScheduler interface {
	EnqueueRun(ctx context.Context, in schedulerapp.RunInput) (*adminplusdomain.SchedulerRunSummary, error)
}

type AcceptanceEvidenceRunReader interface {
	GetRunDetail(ctx context.Context, runID string) (*adminplusdomain.SchedulerRunDetail, error)
}

type acceptanceEvidenceRunObserverSetter interface {
	WithRunStatusObserver(observer schedulerapp.RunStatusObserver) *schedulerapp.Service
}

func (s *Service) enqueueAcceptanceEvidenceTasks(ctx context.Context, supplyType string, supplierID int64, accountID int64, model string) (*adminplusdomain.SchedulerRunSummary, error) {
	if s == nil || s.evidenceScheduler == nil {
		return nil, internalError("acceptance evidence scheduler is not configured")
	}
	if supplierID <= 0 {
		return nil, badRequest("KANBAN_ACCEPTANCE_SUPPLIER_REQUIRED", "supplier_id is required to enqueue acceptance evidence tasks")
	}
	return s.evidenceScheduler.EnqueueRun(ctx, schedulerapp.RunInput{
		Mode:       "kanban_acceptance",
		SupplierID: supplierID,
		TaskTypes: []adminplusdomain.ExtensionTaskType{
			adminplusdomain.ExtensionTaskTypeFetchHealth,
			adminplusdomain.ExtensionTaskTypeCheckChannels,
			adminplusdomain.ExtensionTaskTypeRunPurityCheck,
			adminplusdomain.ExtensionTaskTypeFetchUsageCosts,
			adminplusdomain.ExtensionTaskTypeFetchBalance,
		},
		Request: map[string]any{
			"supply_type":              normalizeSupplyType(supplyType),
			"model":                    strings.TrimSpace(model),
			"local_sub2api_account_id": accountID,
		},
		WindowMinutes: 5,
	})
}
