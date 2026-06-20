package promotions

import (
	"context"
	"net/http"
	"strings"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type RecordPromotionInput struct {
	SupplierID       int64
	Source           string
	Type             adminplusdomain.PromotionType
	Title            string
	Description      string
	Currency         string
	MinRechargeCents int64
	BonusPercent     *float64
	DiscountPercent  *float64
	RuntimeStatus    adminplusdomain.SupplierRuntimeStatus
	BalanceCents     int64
	StartsAt         *time.Time
	EndsAt           *time.Time
	CapturedAt       *time.Time
	RawPayload       map[string]any
}

type EventFilter struct {
	SupplierID     int64
	Status         adminplusdomain.PromotionStatus
	Recommendation adminplusdomain.PromotionRecommendation
	Limit          int
}

type Repository interface {
	CreateEvent(ctx context.Context, event *adminplusdomain.PromotionEvent) (*adminplusdomain.PromotionEvent, error)
	ListEvents(ctx context.Context, filter EventFilter) ([]*adminplusdomain.PromotionEvent, error)
	UpdateEventStatus(ctx context.Context, id int64, status adminplusdomain.PromotionStatus) (*adminplusdomain.PromotionEvent, error)
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

func (s *Service) RecordPromotion(ctx context.Context, in RecordPromotionInput) (*adminplusdomain.PromotionEvent, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("promotion service is not configured")
	}
	event, err := s.buildEvent(in)
	if err != nil {
		return nil, err
	}
	return s.repo.CreateEvent(ctx, event)
}

func (s *Service) ListEvents(ctx context.Context, filter EventFilter) ([]*adminplusdomain.PromotionEvent, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("promotion service is not configured")
	}
	if filter.SupplierID < 0 {
		return nil, badRequest("PROMOTION_SUPPLIER_ID_INVALID", "invalid supplier id")
	}
	if filter.Status != "" && !filter.Status.Valid() {
		return nil, badRequest("PROMOTION_STATUS_INVALID", "invalid promotion status")
	}
	filter.Limit = normalizeLimit(filter.Limit)
	return s.repo.ListEvents(ctx, filter)
}

func (s *Service) AcknowledgeEvent(ctx context.Context, id int64) (*adminplusdomain.PromotionEvent, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("promotion service is not configured")
	}
	if id <= 0 {
		return nil, badRequest("PROMOTION_EVENT_ID_INVALID", "invalid promotion event id")
	}
	return s.repo.UpdateEventStatus(ctx, id, adminplusdomain.PromotionStatusAcknowledged)
}

func (s *Service) buildEvent(in RecordPromotionInput) (*adminplusdomain.PromotionEvent, error) {
	if in.SupplierID <= 0 {
		return nil, badRequest("PROMOTION_SUPPLIER_ID_INVALID", "invalid supplier id")
	}
	if !in.Type.Valid() {
		return nil, badRequest("PROMOTION_TYPE_INVALID", "invalid promotion type")
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, badRequest("PROMOTION_TITLE_REQUIRED", "promotion title is required")
	}
	if len(title) > 160 {
		return nil, badRequest("PROMOTION_TITLE_TOO_LONG", "promotion title must be 160 characters or less")
	}
	if in.MinRechargeCents < 0 {
		return nil, badRequest("PROMOTION_MIN_RECHARGE_INVALID", "minimum recharge must be non-negative")
	}
	if in.BalanceCents < 0 {
		return nil, badRequest("PROMOTION_BALANCE_INVALID", "balance must be non-negative")
	}
	if err := validatePercent("PROMOTION_BONUS_PERCENT_INVALID", in.BonusPercent); err != nil {
		return nil, err
	}
	if err := validatePercent("PROMOTION_DISCOUNT_PERCENT_INVALID", in.DiscountPercent); err != nil {
		return nil, err
	}
	if in.StartsAt != nil && in.EndsAt != nil && in.EndsAt.Before(*in.StartsAt) {
		return nil, badRequest("PROMOTION_TIME_RANGE_INVALID", "promotion end time must be after start time")
	}
	runtimeStatus := in.RuntimeStatus
	if runtimeStatus == "" {
		runtimeStatus = adminplusdomain.SupplierRuntimeStatusMonitorOnly
	}
	if !runtimeStatus.Valid() {
		return nil, badRequest("PROMOTION_RUNTIME_STATUS_INVALID", "invalid supplier runtime status")
	}
	capturedAt := s.now().UTC()
	if in.CapturedAt != nil {
		capturedAt = in.CapturedAt.UTC()
	}
	switchEligible := adminplusdomain.CanUseSupplierForSwitching(runtimeStatus, in.BalanceCents)
	return &adminplusdomain.PromotionEvent{
		SupplierID:       in.SupplierID,
		Source:           normalizeSource(in.Source),
		Type:             in.Type,
		Title:            title,
		Description:      trimLimit(in.Description, 2000),
		Currency:         normalizeCurrency(in.Currency),
		MinRechargeCents: in.MinRechargeCents,
		BonusPercent:     cloneFloat64(in.BonusPercent),
		DiscountPercent:  cloneFloat64(in.DiscountPercent),
		RuntimeStatus:    runtimeStatus,
		BalanceCents:     in.BalanceCents,
		SwitchEligible:   switchEligible,
		Recommendation:   resolveRecommendation(runtimeStatus, in.BalanceCents, switchEligible),
		Status:           adminplusdomain.PromotionStatusOpen,
		StartsAt:         cloneTime(in.StartsAt),
		EndsAt:           cloneTime(in.EndsAt),
		CapturedAt:       capturedAt,
		RawPayload:       in.RawPayload,
	}, nil
}

func resolveRecommendation(status adminplusdomain.SupplierRuntimeStatus, balanceCents int64, switchEligible bool) adminplusdomain.PromotionRecommendation {
	if status == adminplusdomain.SupplierRuntimeStatusDisabled {
		return adminplusdomain.PromotionRecommendationInformational
	}
	if switchEligible {
		return adminplusdomain.PromotionRecommendationSwitchCandidate
	}
	if balanceCents <= 0 {
		return adminplusdomain.PromotionRecommendationRechargeToUnlock
	}
	return adminplusdomain.PromotionRecommendationMonitorOnly
}

func validatePercent(reason string, value *float64) error {
	if value == nil {
		return nil
	}
	if *value < 0 || *value > 100 {
		return badRequest(reason, "promotion percent must be between 0 and 100")
	}
	return nil
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

func cloneFloat64(in *float64) *float64 {
	if in == nil {
		return nil
	}
	v := *in
	return &v
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
