package ports

import (
	"context"
	"time"

	balancesapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/balances"
	healthapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/health"
	promotionsapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/promotions"
)

var _ ProviderAdapter = (*stubProviderAdapter)(nil)

type stubProviderAdapter struct{}

func (s *stubProviderAdapter) Identity() ProviderIdentity {
	return ProviderIdentity{SupplierID: 1, Kind: ProviderKindSub2API}
}

func (s *stubProviderAdapter) FetchRateCatalog(_ context.Context, _ FetchContext) ([]ProviderRateEntry, error) {
	return nil, nil
}

func (s *stubProviderAdapter) FetchBalance(_ context.Context, _ FetchContext) (*balancesapp.RecordSnapshotInput, error) {
	return &balancesapp.RecordSnapshotInput{SupplierID: 1}, nil
}

func (s *stubProviderAdapter) FetchPromotions(_ context.Context, _ FetchContext) ([]promotionsapp.RecordPromotionInput, error) {
	return nil, nil
}

func (s *stubProviderAdapter) FetchHealthSample(_ context.Context, _ FetchContext) (*healthapp.RecordSampleInput, error) {
	capturedAt := time.Now()
	return &healthapp.RecordSampleInput{SupplierID: 1, CapturedAt: &capturedAt}, nil
}

func (s *stubProviderAdapter) ExportBills(_ context.Context, _ BillExportRequest) (*BillExportResult, error) {
	return nil, nil
}
