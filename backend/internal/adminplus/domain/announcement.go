package domain

import "time"

type AnnouncementType string

const (
	AnnouncementTypeRechargeBonus AnnouncementType = "recharge_bonus"
	AnnouncementTypeRateDiscount  AnnouncementType = "rate_discount"
	AnnouncementTypePackageDeal   AnnouncementType = "package_deal"
	AnnouncementTypeLimitedOffer  AnnouncementType = "limited_offer"
	AnnouncementTypeMaintenance   AnnouncementType = "maintenance"
	AnnouncementTypeIncident      AnnouncementType = "incident"
	AnnouncementTypeNotice        AnnouncementType = "notice"
	AnnouncementTypeOther         AnnouncementType = "other"
)

func (t AnnouncementType) Valid() bool {
	switch t {
	case AnnouncementTypeRechargeBonus,
		AnnouncementTypeRateDiscount,
		AnnouncementTypePackageDeal,
		AnnouncementTypeLimitedOffer,
		AnnouncementTypeMaintenance,
		AnnouncementTypeIncident,
		AnnouncementTypeNotice,
		AnnouncementTypeOther:
		return true
	default:
		return false
	}
}

func (t AnnouncementType) IsCostAnnouncement() bool {
	switch t {
	case AnnouncementTypeRechargeBonus, AnnouncementTypeRateDiscount, AnnouncementTypePackageDeal, AnnouncementTypeLimitedOffer:
		return true
	default:
		return false
	}
}

type AnnouncementRecommendation string

const (
	AnnouncementRecommendationRechargeToUnlock AnnouncementRecommendation = "recharge_to_unlock"
	AnnouncementRecommendationSwitchCandidate  AnnouncementRecommendation = "switch_candidate"
	AnnouncementRecommendationMonitorOnly      AnnouncementRecommendation = "monitor_only"
	AnnouncementRecommendationInformational    AnnouncementRecommendation = "informational"
)

type AnnouncementStatus string

const (
	AnnouncementStatusOpen         AnnouncementStatus = "open"
	AnnouncementStatusAcknowledged AnnouncementStatus = "acknowledged"
	AnnouncementStatusIgnored      AnnouncementStatus = "ignored"
)

func (s AnnouncementStatus) Valid() bool {
	switch s {
	case AnnouncementStatusOpen, AnnouncementStatusAcknowledged, AnnouncementStatusIgnored:
		return true
	default:
		return false
	}
}

type AnnouncementEvent struct {
	ID               int64                      `json:"id"`
	SupplierID       int64                      `json:"supplier_id"`
	Source           string                     `json:"source"`
	Type             AnnouncementType           `json:"type"`
	Title            string                     `json:"title"`
	Description      string                     `json:"description,omitempty"`
	Currency         string                     `json:"currency"`
	MinRechargeCents int64                      `json:"min_recharge_cents"`
	BonusPercent     *float64                   `json:"bonus_percent,omitempty"`
	DiscountPercent  *float64                   `json:"discount_percent,omitempty"`
	RuntimeStatus    SupplierRuntimeStatus      `json:"runtime_status"`
	BalanceCents     int64                      `json:"balance_cents"`
	SwitchEligible   bool                       `json:"switch_eligible"`
	Recommendation   AnnouncementRecommendation `json:"recommendation"`
	Status           AnnouncementStatus         `json:"status"`
	StartsAt         *time.Time                 `json:"starts_at,omitempty"`
	EndsAt           *time.Time                 `json:"ends_at,omitempty"`
	CapturedAt       time.Time                  `json:"captured_at"`
	CreatedAt        time.Time                  `json:"created_at"`
	AcknowledgedAt   *time.Time                 `json:"acknowledged_at,omitempty"`
	RawPayload       map[string]any             `json:"raw_payload,omitempty"`
}
