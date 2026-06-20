package ports

import (
	"context"
	"time"

	balancesapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/balances"
	healthapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/health"
	promotionsapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/promotions"
	ratesapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/rates"
)

type ProviderKind string

const (
	ProviderKindSub2API     ProviderKind = "sub2api"
	ProviderKindNewAPI      ProviderKind = "new_api"
	ProviderKindSourceLLM   ProviderKind = "source_llm"
	ProviderKindBrowserOnly ProviderKind = "browser_only"
	ProviderKindCustom      ProviderKind = "custom"
)

type ProviderIdentity struct {
	SupplierID int64
	Kind       ProviderKind
	Name       string
	BaseURL    string
}

type FetchContext struct {
	SupplierID int64
	CapturedAt time.Time
	TraceID    string
}

type BillExportRequest struct {
	SupplierID int64
	StartedAt  time.Time
	EndedAt    time.Time
	Format     string
}

type BillExportResult struct {
	SupplierID int64
	FileName   string
	MimeType   string
	Content    []byte
	ExportedAt time.Time
}

type ProviderAdapter interface {
	Identity() ProviderIdentity
	FetchRateCatalog(ctx context.Context, fetch FetchContext) ([]ratesapp.RateEntryInput, error)
	FetchBalance(ctx context.Context, fetch FetchContext) (*balancesapp.RecordSnapshotInput, error)
	FetchPromotions(ctx context.Context, fetch FetchContext) ([]promotionsapp.RecordPromotionInput, error)
	FetchHealthSample(ctx context.Context, fetch FetchContext) (*healthapp.RecordSampleInput, error)
	ExportBills(ctx context.Context, request BillExportRequest) (*BillExportResult, error)
}
