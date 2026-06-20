package health

import (
	"context"
	"net/http"
	"strings"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	defaultFirstTokenThresholdMS   = int64(3000)
	defaultTotalLatencyThresholdMS = int64(30000)
)

type RecordSampleInput struct {
	SupplierID                   int64
	Source                       string
	Model                        string
	FirstTokenLatencyMS          int64
	TotalLatencyMS               int64
	StatusCode                   int
	ErrorClass                   string
	ObservedConcurrency          int
	AvailableConcurrency         *int
	ConcurrencyLimit             *int
	FirstTokenThresholdMS        int64
	TotalLatencyThresholdMS      int64
	ConcurrencySaturationPercent float64
	RawPayload                   map[string]any
	CapturedAt                   *time.Time
}

type RecordSampleResult struct {
	Sample *adminplusdomain.HealthSample  `json:"sample"`
	Events []*adminplusdomain.HealthEvent `json:"events"`
}

type SampleFilter struct {
	SupplierID int64
	Model      string
	Limit      int
}

type EventFilter struct {
	SupplierID int64
	Status     adminplusdomain.HealthEventStatus
	Type       adminplusdomain.HealthEventType
	Limit      int
}

type Repository interface {
	CreateSample(ctx context.Context, sample *adminplusdomain.HealthSample) (*adminplusdomain.HealthSample, error)
	CreateEvent(ctx context.Context, event *adminplusdomain.HealthEvent) (*adminplusdomain.HealthEvent, error)
	ListSamples(ctx context.Context, filter SampleFilter) ([]*adminplusdomain.HealthSample, error)
	ListEvents(ctx context.Context, filter EventFilter) ([]*adminplusdomain.HealthEvent, error)
	UpdateEventStatus(ctx context.Context, id int64, status adminplusdomain.HealthEventStatus) (*adminplusdomain.HealthEvent, error)
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

func (s *Service) RecordSample(ctx context.Context, in RecordSampleInput) (*RecordSampleResult, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("health service is not configured")
	}
	sample, thresholds, err := s.buildSample(in)
	if err != nil {
		return nil, err
	}
	created, err := s.repo.CreateSample(ctx, sample)
	if err != nil {
		return nil, err
	}
	events := buildHealthEvents(created, thresholds)
	result := &RecordSampleResult{
		Sample: created,
		Events: make([]*adminplusdomain.HealthEvent, 0, len(events)),
	}
	for _, event := range events {
		createdEvent, err := s.repo.CreateEvent(ctx, event)
		if err != nil {
			return nil, err
		}
		result.Events = append(result.Events, createdEvent)
	}
	return result, nil
}

func (s *Service) ListSamples(ctx context.Context, filter SampleFilter) ([]*adminplusdomain.HealthSample, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("health service is not configured")
	}
	if filter.SupplierID < 0 {
		return nil, badRequest("HEALTH_SUPPLIER_ID_INVALID", "invalid supplier id")
	}
	filter.Model = strings.TrimSpace(filter.Model)
	filter.Limit = normalizeLimit(filter.Limit)
	return s.repo.ListSamples(ctx, filter)
}

func (s *Service) ListEvents(ctx context.Context, filter EventFilter) ([]*adminplusdomain.HealthEvent, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("health service is not configured")
	}
	if filter.SupplierID < 0 {
		return nil, badRequest("HEALTH_SUPPLIER_ID_INVALID", "invalid supplier id")
	}
	if filter.Status != "" && !filter.Status.Valid() {
		return nil, badRequest("HEALTH_EVENT_STATUS_INVALID", "invalid health event status")
	}
	filter.Limit = normalizeLimit(filter.Limit)
	return s.repo.ListEvents(ctx, filter)
}

func (s *Service) AcknowledgeEvent(ctx context.Context, id int64) (*adminplusdomain.HealthEvent, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("health service is not configured")
	}
	if id <= 0 {
		return nil, badRequest("HEALTH_EVENT_ID_INVALID", "invalid health event id")
	}
	return s.repo.UpdateEventStatus(ctx, id, adminplusdomain.HealthEventStatusAcknowledged)
}

type healthThresholds struct {
	firstTokenMS          int64
	totalLatencyMS        int64
	concurrencySaturation float64
}

func (s *Service) buildSample(in RecordSampleInput) (*adminplusdomain.HealthSample, healthThresholds, error) {
	if in.SupplierID <= 0 {
		return nil, healthThresholds{}, badRequest("HEALTH_SUPPLIER_ID_INVALID", "invalid supplier id")
	}
	model := strings.TrimSpace(in.Model)
	if model == "" {
		return nil, healthThresholds{}, badRequest("HEALTH_MODEL_REQUIRED", "health model is required")
	}
	if in.FirstTokenLatencyMS < 0 || in.TotalLatencyMS < 0 {
		return nil, healthThresholds{}, badRequest("HEALTH_LATENCY_INVALID", "latency must be non-negative")
	}
	if in.ObservedConcurrency < 0 {
		return nil, healthThresholds{}, badRequest("HEALTH_CONCURRENCY_INVALID", "observed concurrency must be non-negative")
	}
	if in.AvailableConcurrency != nil && *in.AvailableConcurrency < 0 {
		return nil, healthThresholds{}, badRequest("HEALTH_AVAILABLE_CONCURRENCY_INVALID", "available concurrency must be non-negative")
	}
	if in.ConcurrencyLimit != nil && *in.ConcurrencyLimit < 0 {
		return nil, healthThresholds{}, badRequest("HEALTH_CONCURRENCY_LIMIT_INVALID", "concurrency limit must be non-negative")
	}
	thresholds := healthThresholds{
		firstTokenMS:          in.FirstTokenThresholdMS,
		totalLatencyMS:        in.TotalLatencyThresholdMS,
		concurrencySaturation: in.ConcurrencySaturationPercent,
	}
	if thresholds.firstTokenMS <= 0 {
		thresholds.firstTokenMS = defaultFirstTokenThresholdMS
	}
	if thresholds.totalLatencyMS <= 0 {
		thresholds.totalLatencyMS = defaultTotalLatencyThresholdMS
	}
	if thresholds.concurrencySaturation <= 0 || thresholds.concurrencySaturation > 100 {
		thresholds.concurrencySaturation = 100
	}
	capturedAt := s.now().UTC()
	if in.CapturedAt != nil {
		capturedAt = in.CapturedAt.UTC()
	}
	return &adminplusdomain.HealthSample{
		SupplierID:           in.SupplierID,
		Source:               normalizeSource(in.Source),
		Model:                model,
		FirstTokenLatencyMS:  in.FirstTokenLatencyMS,
		TotalLatencyMS:       in.TotalLatencyMS,
		StatusCode:           in.StatusCode,
		ErrorClass:           trimLimit(in.ErrorClass, 80),
		ObservedConcurrency:  in.ObservedConcurrency,
		AvailableConcurrency: cloneInt(in.AvailableConcurrency),
		ConcurrencyLimit:     cloneInt(in.ConcurrencyLimit),
		RawPayload:           in.RawPayload,
		CapturedAt:           capturedAt,
	}, thresholds, nil
}

func buildHealthEvents(sample *adminplusdomain.HealthSample, thresholds healthThresholds) []*adminplusdomain.HealthEvent {
	if sample == nil {
		return nil
	}
	events := make([]*adminplusdomain.HealthEvent, 0, 4)
	if sample.FirstTokenLatencyMS > thresholds.firstTokenMS {
		events = append(events, newHealthEvent(sample, adminplusdomain.HealthEventTypeSlowFirstToken, sample.FirstTokenLatencyMS, thresholds.firstTokenMS))
	}
	if sample.TotalLatencyMS > thresholds.totalLatencyMS {
		events = append(events, newHealthEvent(sample, adminplusdomain.HealthEventTypeSlowTotal, sample.TotalLatencyMS, thresholds.totalLatencyMS))
	}
	if sample.StatusCode >= 400 || sample.ErrorClass != "" {
		observed := int64(sample.StatusCode)
		if observed == 0 {
			observed = 1
		}
		events = append(events, newHealthEvent(sample, adminplusdomain.HealthEventTypeRequestError, observed, 0))
	}
	if sample.ConcurrencyLimit != nil && *sample.ConcurrencyLimit > 0 {
		observedPercent := float64(sample.ObservedConcurrency) / float64(*sample.ConcurrencyLimit) * 100
		if observedPercent >= thresholds.concurrencySaturation {
			events = append(events, newHealthEvent(sample, adminplusdomain.HealthEventTypeConcurrencyFull, int64(sample.ObservedConcurrency), int64(*sample.ConcurrencyLimit)))
		}
	}
	return events
}

func newHealthEvent(sample *adminplusdomain.HealthSample, eventType adminplusdomain.HealthEventType, observed int64, threshold int64) *adminplusdomain.HealthEvent {
	return &adminplusdomain.HealthEvent{
		SupplierID:     sample.SupplierID,
		SampleID:       sample.ID,
		Type:           eventType,
		Model:          sample.Model,
		ObservedValue:  observed,
		ThresholdValue: threshold,
		StatusCode:     sample.StatusCode,
		ErrorClass:     sample.ErrorClass,
		Status:         adminplusdomain.HealthEventStatusOpen,
	}
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

func trimLimit(value string, limit int) string {
	v := strings.TrimSpace(value)
	if len(v) <= limit {
		return v
	}
	return v[:limit]
}

func cloneInt(in *int) *int {
	if in == nil {
		return nil
	}
	v := *in
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
