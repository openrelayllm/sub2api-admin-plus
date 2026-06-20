package domain

import "time"

type PromotionType string

const (
	PromotionTypeRechargeBonus PromotionType = "recharge_bonus"
	PromotionTypeRateDiscount  PromotionType = "rate_discount"
	PromotionTypePackageDeal   PromotionType = "package_deal"
	PromotionTypeLimitedOffer  PromotionType = "limited_offer"
	PromotionTypeOther         PromotionType = "other"
)

func (t PromotionType) Valid() bool {
	switch t {
	case PromotionTypeRechargeBonus, PromotionTypeRateDiscount, PromotionTypePackageDeal, PromotionTypeLimitedOffer, PromotionTypeOther:
		return true
	default:
		return false
	}
}

type PromotionRecommendation string

const (
	PromotionRecommendationRechargeToUnlock PromotionRecommendation = "recharge_to_unlock"
	PromotionRecommendationSwitchCandidate  PromotionRecommendation = "switch_candidate"
	PromotionRecommendationMonitorOnly      PromotionRecommendation = "monitor_only"
	PromotionRecommendationInformational    PromotionRecommendation = "informational"
)

type PromotionStatus string

const (
	PromotionStatusOpen         PromotionStatus = "open"
	PromotionStatusAcknowledged PromotionStatus = "acknowledged"
	PromotionStatusIgnored      PromotionStatus = "ignored"
)

func (s PromotionStatus) Valid() bool {
	switch s {
	case PromotionStatusOpen, PromotionStatusAcknowledged, PromotionStatusIgnored:
		return true
	default:
		return false
	}
}

type PromotionEvent struct {
	ID               int64                   `json:"id"`
	SupplierID       int64                   `json:"supplier_id"`
	Source           string                  `json:"source"`
	Type             PromotionType           `json:"type"`
	Title            string                  `json:"title"`
	Description      string                  `json:"description,omitempty"`
	Currency         string                  `json:"currency"`
	MinRechargeCents int64                   `json:"min_recharge_cents"`
	BonusPercent     *float64                `json:"bonus_percent,omitempty"`
	DiscountPercent  *float64                `json:"discount_percent,omitempty"`
	RuntimeStatus    SupplierRuntimeStatus   `json:"runtime_status"`
	BalanceCents     int64                   `json:"balance_cents"`
	SwitchEligible   bool                    `json:"switch_eligible"`
	Recommendation   PromotionRecommendation `json:"recommendation"`
	Status           PromotionStatus         `json:"status"`
	StartsAt         *time.Time              `json:"starts_at,omitempty"`
	EndsAt           *time.Time              `json:"ends_at,omitempty"`
	CapturedAt       time.Time               `json:"captured_at"`
	CreatedAt        time.Time               `json:"created_at"`
	AcknowledgedAt   *time.Time              `json:"acknowledged_at,omitempty"`
	RawPayload       map[string]any          `json:"raw_payload,omitempty"`
}
