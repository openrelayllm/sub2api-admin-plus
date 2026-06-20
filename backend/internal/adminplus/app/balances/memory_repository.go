package balances

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type MemoryRepository struct {
	mu             sync.Mutex
	nextSnapshotID int64
	nextEventID    int64
	snapshots      []*adminplusdomain.BalanceSnapshot
	events         []*adminplusdomain.BalanceEvent
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		nextSnapshotID: 1,
		nextEventID:    1,
	}
}

func (r *MemoryRepository) CreateSnapshot(_ context.Context, snapshot *adminplusdomain.BalanceSnapshot) (*adminplusdomain.BalanceSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cp := cloneMemoryBalanceSnapshot(snapshot)
	cp.ID = r.nextSnapshotID
	r.nextSnapshotID++
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = cp.CapturedAt
	}
	r.snapshots = append(r.snapshots, cp)
	return cloneMemoryBalanceSnapshot(cp), nil
}

func (r *MemoryRepository) FindLatestSnapshot(_ context.Context, supplierID int64, currency string, capturedAt time.Time) (*adminplusdomain.BalanceSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var latest *adminplusdomain.BalanceSnapshot
	for _, item := range r.snapshots {
		if item.SupplierID != supplierID || item.Currency != currency {
			continue
		}
		if item.CapturedAt.After(capturedAt) {
			continue
		}
		if latest == nil || item.CapturedAt.After(latest.CapturedAt) || (item.CapturedAt.Equal(latest.CapturedAt) && item.ID > latest.ID) {
			latest = item
		}
	}
	return cloneMemoryBalanceSnapshot(latest), nil
}

func (r *MemoryRepository) CreateEvent(_ context.Context, event *adminplusdomain.BalanceEvent) (*adminplusdomain.BalanceEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cp := cloneMemoryBalanceEvent(event)
	cp.ID = r.nextEventID
	r.nextEventID++
	r.events = append(r.events, cp)
	return cloneMemoryBalanceEvent(cp), nil
}

func (r *MemoryRepository) ListSnapshots(_ context.Context, filter SnapshotFilter) ([]*adminplusdomain.BalanceSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := make([]*adminplusdomain.BalanceSnapshot, 0, len(r.snapshots))
	for _, item := range r.snapshots {
		if filter.SupplierID > 0 && item.SupplierID != filter.SupplierID {
			continue
		}
		items = append(items, cloneMemoryBalanceSnapshot(item))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CapturedAt.Equal(items[j].CapturedAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].CapturedAt.After(items[j].CapturedAt)
	})
	if filter.Limit > 0 && len(items) > filter.Limit {
		items = items[:filter.Limit]
	}
	return items, nil
}

func (r *MemoryRepository) ListEvents(_ context.Context, filter EventFilter) ([]*adminplusdomain.BalanceEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := make([]*adminplusdomain.BalanceEvent, 0, len(r.events))
	for _, item := range r.events {
		if filter.SupplierID > 0 && item.SupplierID != filter.SupplierID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		items = append(items, cloneMemoryBalanceEvent(item))
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

func (r *MemoryRepository) UpdateEventStatus(_ context.Context, id int64, status adminplusdomain.BalanceEventStatus) (*adminplusdomain.BalanceEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, event := range r.events {
		if event.ID == id {
			event.Status = status
			return cloneMemoryBalanceEvent(event), nil
		}
	}
	return nil, infraerrors.New(http.StatusNotFound, "BALANCE_EVENT_NOT_FOUND", "balance event not found")
}

func cloneMemoryBalanceSnapshot(in *adminplusdomain.BalanceSnapshot) *adminplusdomain.BalanceSnapshot {
	if in == nil {
		return nil
	}
	out := *in
	if in.RawPayload != nil {
		out.RawPayload = make(map[string]any, len(in.RawPayload))
		for k, v := range in.RawPayload {
			out.RawPayload[k] = v
		}
	}
	return &out
}

func cloneMemoryBalanceEvent(in *adminplusdomain.BalanceEvent) *adminplusdomain.BalanceEvent {
	if in == nil {
		return nil
	}
	out := *in
	if in.OldBalanceCents != nil {
		v := *in.OldBalanceCents
		out.OldBalanceCents = &v
	}
	if in.AcknowledgedAt != nil {
		t := *in.AcknowledgedAt
		out.AcknowledgedAt = &t
	}
	return &out
}
