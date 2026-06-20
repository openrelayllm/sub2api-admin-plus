package billing

import (
	"context"
	"sync"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
)

type MemoryRepository struct {
	mu     sync.Mutex
	nextID int64
	lines  []*adminplusdomain.SupplierBillLine
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{nextID: 1}
}

func (r *MemoryRepository) CreateBillLine(_ context.Context, line *adminplusdomain.SupplierBillLine) (*adminplusdomain.SupplierBillLine, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cp := cloneMemoryBillLine(line)
	cp.ID = r.nextID
	r.nextID++
	r.lines = append(r.lines, cp)
	return cloneMemoryBillLine(cp), nil
}

func cloneMemoryBillLine(in *adminplusdomain.SupplierBillLine) *adminplusdomain.SupplierBillLine {
	if in == nil {
		return nil
	}
	out := *in
	if in.EndedAt != nil {
		t := *in.EndedAt
		out.EndedAt = &t
	}
	if in.RawPayload != nil {
		out.RawPayload = make(map[string]any, len(in.RawPayload))
		for k, v := range in.RawPayload {
			out.RawPayload[k] = v
		}
	}
	return &out
}
