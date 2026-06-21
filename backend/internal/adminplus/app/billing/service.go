package billing

import (
	"context"
	"net/http"
	"strings"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/ports"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type ImportBillLineInput struct {
	SupplierID        int64
	Source            string
	ExternalBillID    string
	ExternalRequestID string
	APIKeyName        string
	Model             string
	Endpoint          string
	RequestType       string
	BillingMode       string
	ReasoningEffort   string
	Currency          string
	CostCents         int64
	InputTokens       int64
	OutputTokens      int64
	CacheReadTokens   int64
	TotalTokens       int64
	FirstTokenMS      int64
	DurationMS        int64
	UserAgent         string
	StartedAt         time.Time
	EndedAt           *time.Time
	RawPayload        map[string]any
}

type BillLineFilter struct {
	SupplierID int64
	Limit      int
}

type SyncFromSessionInput struct {
	SupplierID int64
	StartedAt  time.Time
	EndedAt    time.Time
}

type SyncFromSessionResult struct {
	SupplierID int64                               `json:"supplier_id"`
	SystemType string                              `json:"system_type"`
	Origin     string                              `json:"origin"`
	APIBaseURL string                              `json:"api_base_url"`
	SyncedAt   time.Time                           `json:"synced_at"`
	Total      int                                 `json:"total"`
	Items      []*adminplusdomain.SupplierBillLine `json:"items"`
}

type Repository interface {
	CreateBillLine(ctx context.Context, line *adminplusdomain.SupplierBillLine) (*adminplusdomain.SupplierBillLine, error)
	ListBillLines(ctx context.Context, filter BillLineFilter) ([]*adminplusdomain.SupplierBillLine, error)
}

type SessionReader interface {
	DecryptedProbeInput(ctx context.Context, supplierID int64) (ports.SessionProbeInput, error)
}

type Service struct {
	repo    Repository
	session SessionReader
	reader  ports.SessionBillingAdapter
	now     func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  time.Now,
	}
}

func NewServiceWithDependencies(repo Repository, session SessionReader, reader ports.SessionBillingAdapter) *Service {
	service := NewService(repo)
	service.session = session
	service.reader = reader
	return service
}

func (s *Service) ImportBillLines(ctx context.Context, lines []ImportBillLineInput) ([]*adminplusdomain.SupplierBillLine, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("billing service is not configured")
	}
	if len(lines) == 0 {
		return nil, badRequest("BILLING_LINES_REQUIRED", "billing lines are required")
	}
	if len(lines) > 1000 {
		return nil, badRequest("BILLING_LINES_TOO_MANY", "billing lines must be 1000 or less")
	}
	created := make([]*adminplusdomain.SupplierBillLine, 0, len(lines))
	for _, input := range lines {
		line, err := s.buildLine(input)
		if err != nil {
			return nil, err
		}
		stored, err := s.repo.CreateBillLine(ctx, line)
		if err != nil {
			return nil, err
		}
		created = append(created, stored)
	}
	return created, nil
}

func (s *Service) ListBillLines(ctx context.Context, filter BillLineFilter) ([]*adminplusdomain.SupplierBillLine, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("billing service is not configured")
	}
	if filter.SupplierID < 0 {
		return nil, badRequest("BILLING_SUPPLIER_ID_INVALID", "invalid supplier id")
	}
	filter.Limit = normalizeLimit(filter.Limit)
	return s.repo.ListBillLines(ctx, filter)
}

func (s *Service) SyncFromSession(ctx context.Context, in SyncFromSessionInput) (*SyncFromSessionResult, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("billing service is not configured")
	}
	if s.session == nil {
		return nil, internalError("supplier browser session service is not configured")
	}
	if s.reader == nil {
		return nil, internalError("supplier billing provider adapter is not configured")
	}
	if in.SupplierID <= 0 {
		return nil, badRequest("BILLING_SUPPLIER_ID_INVALID", "invalid supplier id")
	}
	if in.StartedAt.IsZero() || in.EndedAt.IsZero() || !in.StartedAt.Before(in.EndedAt) {
		return nil, badRequest("BILLING_TIME_RANGE_INVALID", "billing ended_at must be after started_at")
	}
	probeInput, err := s.session.DecryptedProbeInput(ctx, in.SupplierID)
	if err != nil {
		return nil, err
	}
	readResult, err := s.reader.ReadBilling(ctx, probeInput, ports.ReadBillingInput{
		SupplierID: in.SupplierID,
		StartedAt:  in.StartedAt.UTC(),
		EndedAt:    in.EndedAt.UTC(),
	})
	if err != nil {
		return nil, err
	}
	lines := make([]ImportBillLineInput, 0, len(readResult.Lines))
	for _, line := range readResult.Lines {
		lines = append(lines, ImportBillLineInput{
			SupplierID:        in.SupplierID,
			Source:            "provider_session",
			ExternalBillID:    line.ExternalBillID,
			ExternalRequestID: line.ExternalRequestID,
			APIKeyName:        line.APIKeyName,
			Model:             line.Model,
			Endpoint:          line.Endpoint,
			RequestType:       line.RequestType,
			BillingMode:       line.BillingMode,
			ReasoningEffort:   line.ReasoningEffort,
			Currency:          line.Currency,
			CostCents:         line.CostCents,
			InputTokens:       line.InputTokens,
			OutputTokens:      line.OutputTokens,
			CacheReadTokens:   line.CacheReadTokens,
			TotalTokens:       line.TotalTokens,
			FirstTokenMS:      line.FirstTokenMS,
			DurationMS:        line.DurationMS,
			UserAgent:         line.UserAgent,
			StartedAt:         line.StartedAt,
			EndedAt:           line.EndedAt,
			RawPayload:        line.RawPayload,
		})
	}
	items := make([]*adminplusdomain.SupplierBillLine, 0)
	if len(lines) > 0 {
		imported, err := s.ImportBillLines(ctx, lines)
		if err != nil {
			return nil, err
		}
		items = imported
	}
	syncedAt := readResult.CapturedAt.UTC()
	if syncedAt.IsZero() {
		syncedAt = s.now().UTC()
	}
	return &SyncFromSessionResult{
		SupplierID: in.SupplierID,
		SystemType: readResult.SystemType,
		Origin:     readResult.Origin,
		APIBaseURL: readResult.APIBaseURL,
		SyncedAt:   syncedAt,
		Total:      len(items),
		Items:      items,
	}, nil
}

func (s *Service) buildLine(in ImportBillLineInput) (*adminplusdomain.SupplierBillLine, error) {
	if in.SupplierID <= 0 {
		return nil, badRequest("BILLING_SUPPLIER_ID_INVALID", "invalid supplier id")
	}
	model := strings.TrimSpace(in.Model)
	if model == "" {
		return nil, badRequest("BILLING_MODEL_REQUIRED", "billing model is required")
	}
	if in.CostCents < 0 {
		return nil, badRequest("BILLING_COST_INVALID", "billing cost must be non-negative")
	}
	if in.InputTokens < 0 || in.OutputTokens < 0 || in.CacheReadTokens < 0 || in.TotalTokens < 0 {
		return nil, badRequest("BILLING_TOKENS_INVALID", "billing tokens must be non-negative")
	}
	if in.FirstTokenMS < 0 || in.DurationMS < 0 {
		return nil, badRequest("BILLING_LATENCY_INVALID", "billing latency must be non-negative")
	}
	if in.StartedAt.IsZero() {
		return nil, badRequest("BILLING_STARTED_AT_REQUIRED", "billing started_at is required")
	}
	if in.EndedAt != nil && in.EndedAt.Before(in.StartedAt) {
		return nil, badRequest("BILLING_TIME_RANGE_INVALID", "billing ended_at must be after started_at")
	}
	return &adminplusdomain.SupplierBillLine{
		SupplierID:        in.SupplierID,
		Source:            normalizeSource(in.Source),
		ExternalBillID:    trimLimit(in.ExternalBillID, 120),
		ExternalRequestID: trimLimit(in.ExternalRequestID, 160),
		APIKeyName:        trimLimit(in.APIKeyName, 160),
		Model:             model,
		Endpoint:          trimLimit(in.Endpoint, 160),
		RequestType:       trimLimit(in.RequestType, 80),
		BillingMode:       trimLimit(in.BillingMode, 60),
		ReasoningEffort:   trimLimit(in.ReasoningEffort, 60),
		Currency:          normalizeCurrency(in.Currency),
		CostCents:         in.CostCents,
		InputTokens:       in.InputTokens,
		OutputTokens:      in.OutputTokens,
		CacheReadTokens:   in.CacheReadTokens,
		TotalTokens:       normalizedTotalTokens(in),
		FirstTokenMS:      in.FirstTokenMS,
		DurationMS:        in.DurationMS,
		UserAgent:         trimLimit(in.UserAgent, 300),
		StartedAt:         in.StartedAt.UTC(),
		EndedAt:           cloneTime(in.EndedAt),
		RawPayload:        in.RawPayload,
		CreatedAt:         s.now().UTC(),
	}, nil
}

func normalizedTotalTokens(in ImportBillLineInput) int64 {
	if in.TotalTokens > 0 {
		return in.TotalTokens
	}
	return in.InputTokens + in.OutputTokens + in.CacheReadTokens
}

func normalizeSource(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == "" {
		return "manual"
	}
	if len(v) > 60 {
		return v[:60]
	}
	return v
}

func normalizeCurrency(value string) string {
	v := strings.ToUpper(strings.TrimSpace(value))
	if len(v) != 3 {
		return "USD"
	}
	return v
}

func trimLimit(value string, limit int) string {
	v := strings.TrimSpace(value)
	if len(v) <= limit {
		return v
	}
	return v[:limit]
}

func cloneTime(in *time.Time) *time.Time {
	if in == nil {
		return nil
	}
	v := in.UTC()
	return &v
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
