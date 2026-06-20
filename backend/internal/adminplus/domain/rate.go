package domain

import "time"

type RateSnapshot struct {
	ID            int64          `json:"id"`
	SupplierID    int64          `json:"supplier_id"`
	Source        string         `json:"source"`
	Model         string         `json:"model"`
	BillingMode   string         `json:"billing_mode"`
	PriceItem     string         `json:"price_item"`
	Unit          string         `json:"unit"`
	Currency      string         `json:"currency"`
	PriceMicros   int64          `json:"price_micros"`
	RawPayload    map[string]any `json:"raw_payload,omitempty"`
	CapturedAt    time.Time      `json:"captured_at"`
	CreatedAt     time.Time      `json:"created_at"`
}

type RateChangeDirection string

const (
	RateChangeDirectionNew      RateChangeDirection = "new"
	RateChangeDirectionIncrease RateChangeDirection = "increase"
	RateChangeDirectionDecrease RateChangeDirection = "decrease"
)

type RateChangeStatus string

const (
	RateChangeStatusOpen         RateChangeStatus = "open"
	RateChangeStatusAcknowledged RateChangeStatus = "acknowledged"
	RateChangeStatusIgnored      RateChangeStatus = "ignored"
)

func (s RateChangeStatus) Valid() bool {
	switch s {
	case RateChangeStatusOpen, RateChangeStatusAcknowledged, RateChangeStatusIgnored:
		return true
	default:
		return false
	}
}

type RateChangeEvent struct {
	ID                int64               `json:"id"`
	SupplierID        int64               `json:"supplier_id"`
	SnapshotID        int64               `json:"snapshot_id"`
	Model             string              `json:"model"`
	BillingMode       string              `json:"billing_mode"`
	PriceItem         string              `json:"price_item"`
	Unit              string              `json:"unit"`
	Currency          string              `json:"currency"`
	OldPriceMicros    *int64              `json:"old_price_micros,omitempty"`
	NewPriceMicros    int64               `json:"new_price_micros"`
	Direction         RateChangeDirection `json:"direction"`
	ChangePercent     *float64            `json:"change_percent,omitempty"`
	ThresholdPercent  float64             `json:"threshold_percent"`
	ThresholdExceeded bool                `json:"threshold_exceeded"`
	Status            RateChangeStatus    `json:"status"`
	CreatedAt         time.Time           `json:"created_at"`
	AcknowledgedAt     *time.Time          `json:"acknowledged_at,omitempty"`
}
