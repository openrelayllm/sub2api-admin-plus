package promotions

import (
	"context"
	"net/http"
	"sort"
	"sync"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type MemoryRepository struct {
	mu          sync.Mutex
	nextEventID int64
	events      []*adminplusdomain.PromotionEvent
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{nextEventID: 1}
}

func (r *MemoryRepository) CreateEvent(_ context.Context, event *adminplusdomain.PromotionEvent) (*adminplusdomain.PromotionEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cp := cloneMemoryPromotionEvent(event)
	cp.ID = r.nextEventID
	r.nextEventID++
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = cp.CapturedAt
	}
	r.events = append(r.events, cp)
	return cloneMemoryPromotionEvent(cp), nil
}

func (r *MemoryRepository) ListEvents(_ context.Context, filter EventFilter) ([]*adminplusdomain.PromotionEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := make([]*adminplusdomain.PromotionEvent, 0, len(r.events))
	for _, item := range r.events {
		if filter.SupplierID > 0 && item.SupplierID != filter.SupplierID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if filter.Recommendation != "" && item.Recommendation != filter.Recommendation {
			continue
		}
		items = append(items, cloneMemoryPromotionEvent(item))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if filter.Limit > 0 && len(items) > filter.Limit {
		items = items[:filter.Limit]
	}
	return items, nil
}

func (r *MemoryRepository) UpdateEventStatus(_ context.Context, id int64, status adminplusdomain.PromotionStatus) (*adminplusdomain.PromotionEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, event := range r.events {
		if event.ID == id {
			event.Status = status
			return cloneMemoryPromotionEvent(event), nil
		}
	}
	return nil, infraerrors.New(http.StatusNotFound, "PROMOTION_EVENT_NOT_FOUND", "promotion event not found")
}

func cloneMemoryPromotionEvent(in *adminplusdomain.PromotionEvent) *adminplusdomain.PromotionEvent {
	if in == nil {
		return nil
	}
	out := *in
	if in.BonusPercent != nil {
		v := *in.BonusPercent
		out.BonusPercent = &v
	}
	if in.DiscountPercent != nil {
		v := *in.DiscountPercent
		out.DiscountPercent = &v
	}
	if in.StartsAt != nil {
		t := *in.StartsAt
		out.StartsAt = &t
	}
	if in.EndsAt != nil {
		t := *in.EndsAt
		out.EndsAt = &t
	}
	if in.AcknowledgedAt != nil {
		t := *in.AcknowledgedAt
		out.AcknowledgedAt = &t
	}
	if in.RawPayload != nil {
		out.RawPayload = make(map[string]any, len(in.RawPayload))
		for k, v := range in.RawPayload {
			out.RawPayload[k] = v
		}
	}
	return &out
}
