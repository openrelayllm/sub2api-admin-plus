package domain

import "time"

type BalanceSnapshot struct {
	ID             int64                 `json:"id"`
	SupplierID     int64                 `json:"supplier_id"`
	Source         string                `json:"source"`
	RuntimeStatus  SupplierRuntimeStatus `json:"runtime_status"`
	BalanceCents   int64                 `json:"balance_cents"`
	Currency       string                `json:"currency"`
	SwitchEligible bool                  `json:"switch_eligible"`
	RawPayload     map[string]any        `json:"raw_payload,omitempty"`
	CapturedAt     time.Time             `json:"captured_at"`
	CreatedAt      time.Time             `json:"created_at"`
}

type BalanceEventType string

const (
	BalanceEventTypeLowBalance BalanceEventType = "low_balance"
	BalanceEventTypeDepleted   BalanceEventType = "depleted"
	BalanceEventTypeRecovered  BalanceEventType = "recovered"
)

type BalanceEventStatus string

const (
	BalanceEventStatusOpen         BalanceEventStatus = "open"
	BalanceEventStatusAcknowledged BalanceEventStatus = "acknowledged"
	BalanceEventStatusIgnored      BalanceEventStatus = "ignored"
)

func (s BalanceEventStatus) Valid() bool {
	switch s {
	case BalanceEventStatusOpen, BalanceEventStatusAcknowledged, BalanceEventStatusIgnored:
		return true
	default:
		return false
	}
}

type BalanceEvent struct {
	ID                       int64                 `json:"id"`
	SupplierID               int64                 `json:"supplier_id"`
	SnapshotID               int64                 `json:"snapshot_id"`
	Type                     BalanceEventType      `json:"type"`
	RuntimeStatus            SupplierRuntimeStatus `json:"runtime_status"`
	OldBalanceCents          *int64                `json:"old_balance_cents,omitempty"`
	NewBalanceCents          int64                 `json:"new_balance_cents"`
	LowBalanceThresholdCents int64                 `json:"low_balance_threshold_cents"`
	Currency                 string                `json:"currency"`
	SwitchEligible           bool                  `json:"switch_eligible"`
	Status                   BalanceEventStatus    `json:"status"`
	CreatedAt                time.Time             `json:"created_at"`
	AcknowledgedAt           *time.Time            `json:"acknowledged_at,omitempty"`
}
