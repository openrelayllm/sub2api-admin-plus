package domain

import "time"

type SupplierGroupChangeDirection string

const (
	SupplierGroupChangeDirectionNew      SupplierGroupChangeDirection = "new"
	SupplierGroupChangeDirectionIncrease SupplierGroupChangeDirection = "increase"
	SupplierGroupChangeDirectionDecrease SupplierGroupChangeDirection = "decrease"
)

func (d SupplierGroupChangeDirection) Valid() bool {
	switch d {
	case SupplierGroupChangeDirectionNew, SupplierGroupChangeDirectionIncrease, SupplierGroupChangeDirectionDecrease:
		return true
	default:
		return false
	}
}

type SupplierGroupChangeEvent struct {
	ID                         int64                        `json:"id"`
	SupplierID                 int64                        `json:"supplier_id"`
	SupplierGroupID            int64                        `json:"supplier_group_id"`
	ExternalGroupID            string                       `json:"external_group_id"`
	GroupName                  string                       `json:"group_name"`
	ProviderFamily             string                       `json:"provider_family"`
	Direction                  SupplierGroupChangeDirection `json:"direction"`
	OldEffectiveRateMultiplier *float64                     `json:"old_effective_rate_multiplier,omitempty"`
	NewEffectiveRateMultiplier float64                      `json:"new_effective_rate_multiplier"`
	ChangePercent              *float64                     `json:"change_percent,omitempty"`
	LowRate                    bool                         `json:"low_rate"`
	CreatedAt                  time.Time                    `json:"created_at"`
}
