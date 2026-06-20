package domain

import "time"

type HealthSample struct {
	ID                   int64          `json:"id"`
	SupplierID           int64          `json:"supplier_id"`
	Source               string         `json:"source"`
	Model                string         `json:"model"`
	FirstTokenLatencyMS  int64          `json:"first_token_latency_ms"`
	TotalLatencyMS       int64          `json:"total_latency_ms"`
	StatusCode           int            `json:"status_code"`
	ErrorClass           string         `json:"error_class,omitempty"`
	ObservedConcurrency  int            `json:"observed_concurrency"`
	AvailableConcurrency *int           `json:"available_concurrency,omitempty"`
	ConcurrencyLimit     *int           `json:"concurrency_limit,omitempty"`
	RawPayload           map[string]any `json:"raw_payload,omitempty"`
	CapturedAt           time.Time      `json:"captured_at"`
	CreatedAt            time.Time      `json:"created_at"`
}

type HealthEventType string

const (
	HealthEventTypeSlowFirstToken  HealthEventType = "slow_first_token"
	HealthEventTypeSlowTotal       HealthEventType = "slow_total"
	HealthEventTypeRequestError    HealthEventType = "request_error"
	HealthEventTypeConcurrencyFull HealthEventType = "concurrency_full"
)

type HealthEventStatus string

const (
	HealthEventStatusOpen         HealthEventStatus = "open"
	HealthEventStatusAcknowledged HealthEventStatus = "acknowledged"
	HealthEventStatusIgnored      HealthEventStatus = "ignored"
)

func (s HealthEventStatus) Valid() bool {
	switch s {
	case HealthEventStatusOpen, HealthEventStatusAcknowledged, HealthEventStatusIgnored:
		return true
	default:
		return false
	}
}

type HealthEvent struct {
	ID             int64             `json:"id"`
	SupplierID     int64             `json:"supplier_id"`
	SampleID       int64             `json:"sample_id"`
	Type           HealthEventType   `json:"type"`
	Model          string            `json:"model"`
	ObservedValue  int64             `json:"observed_value"`
	ThresholdValue int64             `json:"threshold_value"`
	StatusCode     int               `json:"status_code"`
	ErrorClass     string            `json:"error_class,omitempty"`
	Status         HealthEventStatus `json:"status"`
	CreatedAt      time.Time         `json:"created_at"`
	AcknowledgedAt *time.Time        `json:"acknowledged_at,omitempty"`
}
