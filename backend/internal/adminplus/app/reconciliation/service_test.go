package reconciliation

import (
	"context"
	"testing"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/stretchr/testify/require"
)

func TestServiceRunMatchesByExternalRequestIDAndCalculatesProfit(t *testing.T) {
	svc := NewService()
	startedAt := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)

	result, err := svc.Run(context.Background(), RunInput{
		SupplierBills: []*adminplusdomain.SupplierBillLine{
			{
				ID:                11,
				SupplierID:        7,
				ExternalRequestID: "req-1",
				Model:             "gpt-4o-mini",
				Currency:          "USD",
				CostCents:         100,
				StartedAt:         startedAt,
			},
		},
		LocalUsages: []*adminplusdomain.LocalUsageLine{
			{
				ID:                21,
				ExternalRequestID: "req-1",
				Model:             "gpt-4o-mini",
				Currency:          "USD",
				RevenueCents:      150,
				StartedAt:         startedAt,
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, result.Lines, 1)
	line := result.Lines[0]
	require.Equal(t, adminplusdomain.ReconciliationStatusMatched, line.Status)
	require.Equal(t, int64(100), line.CostCents)
	require.Equal(t, int64(150), line.RevenueCents)
	require.Equal(t, int64(50), line.ProfitCents)
	require.NotNil(t, line.ProfitMargin)
	require.InDelta(t, 0.3333, *line.ProfitMargin, 0.001)
	require.Equal(t, int64(50), result.Summary.ProfitCents)
}

func TestServiceRunMatchesByModelAndTimeWindow(t *testing.T) {
	svc := NewService()
	startedAt := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)

	result, err := svc.Run(context.Background(), RunInput{
		TimeTolerance: 2 * time.Minute,
		SupplierBills: []*adminplusdomain.SupplierBillLine{
			{
				ID:         11,
				SupplierID: 7,
				Model:      "claude-sonnet-4",
				Currency:   "USD",
				CostCents:  200,
				StartedAt:  startedAt,
			},
		},
		LocalUsages: []*adminplusdomain.LocalUsageLine{
			{
				ID:           21,
				Model:        "claude-sonnet-4",
				Currency:     "USD",
				RevenueCents: 320,
				StartedAt:    startedAt.Add(time.Minute),
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, result.Lines, 1)
	require.Equal(t, adminplusdomain.ReconciliationStatusMatched, result.Lines[0].Status)
}

func TestServiceRunEmitsSupplierOnlyAndLocalOnlyLines(t *testing.T) {
	svc := NewService()
	startedAt := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)

	result, err := svc.Run(context.Background(), RunInput{
		SupplierBills: []*adminplusdomain.SupplierBillLine{
			{ID: 11, SupplierID: 7, Model: "gpt-4o-mini", Currency: "USD", CostCents: 100, StartedAt: startedAt},
		},
		LocalUsages: []*adminplusdomain.LocalUsageLine{
			{ID: 21, Model: "gemini-2.5-pro", Currency: "USD", RevenueCents: 200, StartedAt: startedAt},
		},
	})

	require.NoError(t, err)
	require.Len(t, result.Lines, 2)
	require.Equal(t, adminplusdomain.ReconciliationStatusSupplierOnly, result.Lines[0].Status)
	require.Equal(t, adminplusdomain.ReconciliationStatusLocalOnly, result.Lines[1].Status)
	require.Equal(t, int64(100), result.Summary.CostCents)
	require.Equal(t, int64(200), result.Summary.RevenueCents)
	require.Equal(t, int64(100), result.Summary.ProfitCents)
}

func TestServiceRunMarksCurrencyMismatch(t *testing.T) {
	svc := NewService()
	startedAt := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)

	result, err := svc.Run(context.Background(), RunInput{
		SupplierBills: []*adminplusdomain.SupplierBillLine{
			{ID: 11, SupplierID: 7, ExternalRequestID: "req-1", Model: "gpt-4o-mini", Currency: "USD", CostCents: 100, StartedAt: startedAt},
		},
		LocalUsages: []*adminplusdomain.LocalUsageLine{
			{ID: 21, ExternalRequestID: "req-1", Model: "gpt-4o-mini", Currency: "CNY", RevenueCents: 200, StartedAt: startedAt},
		},
	})

	require.NoError(t, err)
	require.Equal(t, adminplusdomain.ReconciliationStatusCurrencyMismatch, result.Lines[0].Status)
}

func TestServiceRunMarksRevenueBelowCostAsCostMismatch(t *testing.T) {
	svc := NewService()
	startedAt := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)

	result, err := svc.Run(context.Background(), RunInput{
		SupplierBills: []*adminplusdomain.SupplierBillLine{
			{ID: 11, SupplierID: 7, ExternalRequestID: "req-1", Model: "gpt-4o-mini", Currency: "USD", CostCents: 200, StartedAt: startedAt},
		},
		LocalUsages: []*adminplusdomain.LocalUsageLine{
			{ID: 21, ExternalRequestID: "req-1", Model: "gpt-4o-mini", Currency: "USD", RevenueCents: 150, StartedAt: startedAt},
		},
	})

	require.NoError(t, err)
	require.Equal(t, adminplusdomain.ReconciliationStatusCostMismatch, result.Lines[0].Status)
	require.Equal(t, int64(-50), result.Lines[0].ProfitCents)
}
