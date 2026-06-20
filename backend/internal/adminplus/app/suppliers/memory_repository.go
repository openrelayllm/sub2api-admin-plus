package suppliers

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type MemoryRepository struct {
	mu        sync.RWMutex
	nextID    int64
	suppliers map[int64]*adminplusdomain.Supplier
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		nextID:    1,
		suppliers: make(map[int64]*adminplusdomain.Supplier),
	}
}

func (r *MemoryRepository) Create(_ context.Context, supplier *adminplusdomain.Supplier) (*adminplusdomain.Supplier, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cp := cloneSupplier(supplier)
	cp.ID = r.nextID
	r.nextID++
	r.suppliers[cp.ID] = cp
	return cloneSupplier(cp), nil
}

func (r *MemoryRepository) Get(_ context.Context, id int64) (*adminplusdomain.Supplier, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	supplier, ok := r.suppliers[id]
	if !ok {
		return nil, infraerrors.New(http.StatusNotFound, "SUPPLIER_NOT_FOUND", "supplier not found")
	}
	return cloneSupplier(supplier), nil
}

func (r *MemoryRepository) List(_ context.Context, filter SupplierFilter) ([]*adminplusdomain.Supplier, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]*adminplusdomain.Supplier, 0, len(r.suppliers))
	for _, supplier := range r.suppliers {
		if !matchesFilter(supplier, filter) {
			continue
		}
		items = append(items, cloneSupplier(supplier))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items, nil
}

func (r *MemoryRepository) UpdateStatus(_ context.Context, id int64, runtimeStatus adminplusdomain.SupplierRuntimeStatus, healthStatus adminplusdomain.SupplierHealthStatus) (*adminplusdomain.Supplier, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	supplier, ok := r.suppliers[id]
	if !ok {
		return nil, infraerrors.New(http.StatusNotFound, "SUPPLIER_NOT_FOUND", "supplier not found")
	}
	supplier.RuntimeStatus = runtimeStatus
	supplier.HealthStatus = healthStatus
	supplier.UpdatedAt = time.Now().UTC()
	return cloneSupplier(supplier), nil
}

func matchesFilter(supplier *adminplusdomain.Supplier, filter SupplierFilter) bool {
	if filter.Kind != "" && supplier.Kind != filter.Kind {
		return false
	}
	if filter.Type != "" && supplier.Type != filter.Type {
		return false
	}
	if filter.RuntimeStatus != "" && supplier.RuntimeStatus != filter.RuntimeStatus {
		return false
	}
	if filter.HealthStatus != "" && supplier.HealthStatus != filter.HealthStatus {
		return false
	}
	if filter.Query != "" {
		haystack := strings.ToLower(supplier.Name + " " + supplier.Contact + " " + supplier.Notes)
		if !strings.Contains(haystack, filter.Query) {
			return false
		}
	}
	return true
}

func cloneSupplier(in *adminplusdomain.Supplier) *adminplusdomain.Supplier {
	if in == nil {
		return nil
	}
	out := *in
	if in.BalanceUpdatedAt != nil {
		t := *in.BalanceUpdatedAt
		out.BalanceUpdatedAt = &t
	}
	return &out
}
