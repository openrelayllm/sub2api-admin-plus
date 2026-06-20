package rates

import (
	"context"
	"math"
	"net/http"
	"strings"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const defaultChangeThresholdPercent = 1.0

type RateEntryInput struct {
	Model       string
	BillingMode string
	PriceItem   string
	Unit        string
	Currency    string
	PriceMicros int64
	RawPayload  map[string]any
}

type RecordSnapshotInput struct {
	SupplierID       int64
	Source           string
	CapturedAt       *time.Time
	ThresholdPercent float64
	Entries          []RateEntryInput
}

type RecordSnapshotResult struct {
	Snapshots []*adminplusdomain.RateSnapshot    `json:"snapshots"`
	Events    []*adminplusdomain.RateChangeEvent `json:"events"`
}

type SnapshotFilter struct {
	SupplierID int64
	Model      string
	Limit      int
}

type EventFilter struct {
	SupplierID int64
	Status     adminplusdomain.RateChangeStatus
	Limit      int
}

type Repository interface {
	CreateSnapshot(ctx context.Context, snapshot *adminplusdomain.RateSnapshot) (*adminplusdomain.RateSnapshot, error)
	FindLatestComparableSnapshot(ctx context.Context, snapshot *adminplusdomain.RateSnapshot) (*adminplusdomain.RateSnapshot, error)
	CreateChangeEvent(ctx context.Context, event *adminplusdomain.RateChangeEvent) (*adminplusdomain.RateChangeEvent, error)
	ListSnapshots(ctx context.Context, filter SnapshotFilter) ([]*adminplusdomain.RateSnapshot, error)
	ListChangeEvents(ctx context.Context, filter EventFilter) ([]*adminplusdomain.RateChangeEvent, error)
	UpdateChangeEventStatus(ctx context.Context, id int64, status adminplusdomain.RateChangeStatus) (*adminplusdomain.RateChangeEvent, error)
}

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  time.Now,
	}
}

func (s *Service) RecordSnapshot(ctx context.Context, in RecordSnapshotInput) (*RecordSnapshotResult, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("rate service is not configured")
	}
	if in.SupplierID <= 0 {
		return nil, badRequest("RATE_SUPPLIER_ID_INVALID", "invalid supplier id")
	}
	if len(in.Entries) == 0 {
		return nil, badRequest("RATE_ENTRIES_REQUIRED", "rate entries are required")
	}
	if len(in.Entries) > 500 {
		return nil, badRequest("RATE_ENTRIES_TOO_MANY", "rate entries must be 500 or less")
	}
	source := normalizeSource(in.Source)
	threshold := in.ThresholdPercent
	if threshold <= 0 {
		threshold = defaultChangeThresholdPercent
	}
	capturedAt := s.now().UTC()
	if in.CapturedAt != nil {
		capturedAt = in.CapturedAt.UTC()
	}

	result := &RecordSnapshotResult{
		Snapshots: make([]*adminplusdomain.RateSnapshot, 0, len(in.Entries)),
		Events:    make([]*adminplusdomain.RateChangeEvent, 0),
	}
	for _, entry := range in.Entries {
		snapshot, err := buildSnapshot(in.SupplierID, source, capturedAt, entry)
		if err != nil {
			return nil, err
		}
		previous, err := s.repo.FindLatestComparableSnapshot(ctx, snapshot)
		if err != nil {
			return nil, err
		}
		created, err := s.repo.CreateSnapshot(ctx, snapshot)
		if err != nil {
			return nil, err
		}
		result.Snapshots = append(result.Snapshots, created)

		event := buildChangeEvent(created, previous, threshold)
		if event == nil {
			continue
		}
		createdEvent, err := s.repo.CreateChangeEvent(ctx, event)
		if err != nil {
			return nil, err
		}
		result.Events = append(result.Events, createdEvent)
	}
	return result, nil
}

func (s *Service) ListSnapshots(ctx context.Context, filter SnapshotFilter) ([]*adminplusdomain.RateSnapshot, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("rate service is not configured")
	}
	if filter.SupplierID < 0 {
		return nil, badRequest("RATE_SUPPLIER_ID_INVALID", "invalid supplier id")
	}
	filter.Model = strings.TrimSpace(filter.Model)
	filter.Limit = normalizeLimit(filter.Limit)
	return s.repo.ListSnapshots(ctx, filter)
}

func (s *Service) ListChangeEvents(ctx context.Context, filter EventFilter) ([]*adminplusdomain.RateChangeEvent, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("rate service is not configured")
	}
	if filter.SupplierID < 0 {
		return nil, badRequest("RATE_SUPPLIER_ID_INVALID", "invalid supplier id")
	}
	if filter.Status != "" && !filter.Status.Valid() {
		return nil, badRequest("RATE_EVENT_STATUS_INVALID", "invalid rate event status")
	}
	filter.Limit = normalizeLimit(filter.Limit)
	return s.repo.ListChangeEvents(ctx, filter)
}

func (s *Service) AcknowledgeChangeEvent(ctx context.Context, id int64) (*adminplusdomain.RateChangeEvent, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("rate service is not configured")
	}
	if id <= 0 {
		return nil, badRequest("RATE_EVENT_ID_INVALID", "invalid rate event id")
	}
	return s.repo.UpdateChangeEventStatus(ctx, id, adminplusdomain.RateChangeStatusAcknowledged)
}

func buildSnapshot(supplierID int64, source string, capturedAt time.Time, entry RateEntryInput) (*adminplusdomain.RateSnapshot, error) {
	model := strings.TrimSpace(entry.Model)
	if model == "" {
		return nil, badRequest("RATE_MODEL_REQUIRED", "rate model is required")
	}
	billingMode := strings.ToLower(strings.TrimSpace(entry.BillingMode))
	if billingMode == "" {
		return nil, badRequest("RATE_BILLING_MODE_REQUIRED", "rate billing mode is required")
	}
	priceItem := strings.ToLower(strings.TrimSpace(entry.PriceItem))
	if priceItem == "" {
		return nil, badRequest("RATE_PRICE_ITEM_REQUIRED", "rate price item is required")
	}
	unit := strings.ToLower(strings.TrimSpace(entry.Unit))
	if unit == "" {
		return nil, badRequest("RATE_UNIT_REQUIRED", "rate unit is required")
	}
	if entry.PriceMicros < 0 {
		return nil, badRequest("RATE_PRICE_INVALID", "rate price must be non-negative")
	}
	return &adminplusdomain.RateSnapshot{
		SupplierID:  supplierID,
		Source:      source,
		Model:       model,
		BillingMode: billingMode,
		PriceItem:   priceItem,
		Unit:        unit,
		Currency:    normalizeCurrency(entry.Currency),
		PriceMicros: entry.PriceMicros,
		RawPayload:  entry.RawPayload,
		CapturedAt:  capturedAt,
	}, nil
}

func buildChangeEvent(current, previous *adminplusdomain.RateSnapshot, thresholdPercent float64) *adminplusdomain.RateChangeEvent {
	if current == nil {
		return nil
	}
	event := &adminplusdomain.RateChangeEvent{
		SupplierID:       current.SupplierID,
		SnapshotID:       current.ID,
		Model:            current.Model,
		BillingMode:      current.BillingMode,
		PriceItem:        current.PriceItem,
		Unit:             current.Unit,
		Currency:         current.Currency,
		NewPriceMicros:   current.PriceMicros,
		ThresholdPercent: thresholdPercent,
		Status:           adminplusdomain.RateChangeStatusOpen,
	}
	if previous == nil {
		event.Direction = adminplusdomain.RateChangeDirectionNew
		event.ThresholdExceeded = true
		return event
	}
	if previous.PriceMicros == current.PriceMicros {
		return nil
	}
	oldPrice := previous.PriceMicros
	event.OldPriceMicros = &oldPrice
	if current.PriceMicros > previous.PriceMicros {
		event.Direction = adminplusdomain.RateChangeDirectionIncrease
	} else {
		event.Direction = adminplusdomain.RateChangeDirectionDecrease
	}
	if previous.PriceMicros != 0 {
		changePercent := (float64(current.PriceMicros-previous.PriceMicros) / math.Abs(float64(previous.PriceMicros))) * 100
		event.ChangePercent = &changePercent
		event.ThresholdExceeded = math.Abs(changePercent) >= thresholdPercent
	} else {
		event.ThresholdExceeded = true
	}
	return event
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
