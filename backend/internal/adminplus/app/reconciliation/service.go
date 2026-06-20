package reconciliation

import (
	"context"
	"net/http"
	"strings"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type RunInput struct {
	SupplierBills     []*adminplusdomain.SupplierBillLine
	LocalUsages       []*adminplusdomain.LocalUsageLine
	TimeTolerance     time.Duration
	CostMismatchCents int64
}

type RunResult struct {
	Lines   []*adminplusdomain.ReconciliationLine `json:"lines"`
	Summary adminplusdomain.ReconciliationSummary `json:"summary"`
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Run(_ context.Context, in RunInput) (*RunResult, error) {
	if len(in.SupplierBills) == 0 && len(in.LocalUsages) == 0 {
		return nil, badRequest("RECONCILIATION_INPUT_REQUIRED", "supplier bills or local usages are required")
	}
	if in.TimeTolerance <= 0 {
		in.TimeTolerance = time.Minute
	}
	if in.CostMismatchCents < 0 {
		return nil, badRequest("RECONCILIATION_COST_MISMATCH_INVALID", "cost mismatch threshold must be non-negative")
	}

	localIndexes := buildLocalIndexes(in.LocalUsages)
	usedLocal := make(map[int64]struct{})
	lines := make([]*adminplusdomain.ReconciliationLine, 0, len(in.SupplierBills)+len(in.LocalUsages))
	for _, bill := range in.SupplierBills {
		if bill == nil {
			continue
		}
		local := findBestLocalUsage(bill, localIndexes, usedLocal, in.TimeTolerance)
		if local == nil {
			lines = append(lines, supplierOnlyLine(bill))
			continue
		}
		usedLocal[local.ID] = struct{}{}
		lines = append(lines, matchedLine(bill, local, in.CostMismatchCents))
	}
	for _, usage := range in.LocalUsages {
		if usage == nil {
			continue
		}
		if _, ok := usedLocal[usage.ID]; ok {
			continue
		}
		lines = append(lines, localOnlyLine(usage))
	}
	return &RunResult{
		Lines:   lines,
		Summary: summarize(lines, len(in.SupplierBills), len(in.LocalUsages)),
	}, nil
}

type localIndexes struct {
	byRequestID map[string][]*adminplusdomain.LocalUsageLine
	byModel     map[string][]*adminplusdomain.LocalUsageLine
}

func buildLocalIndexes(usages []*adminplusdomain.LocalUsageLine) localIndexes {
	indexes := localIndexes{
		byRequestID: make(map[string][]*adminplusdomain.LocalUsageLine),
		byModel:     make(map[string][]*adminplusdomain.LocalUsageLine),
	}
	for _, usage := range usages {
		if usage == nil {
			continue
		}
		if key := normalizeKey(usage.ExternalRequestID); key != "" {
			indexes.byRequestID[key] = append(indexes.byRequestID[key], usage)
		}
		if model := normalizeKey(usage.Model); model != "" {
			indexes.byModel[model] = append(indexes.byModel[model], usage)
		}
	}
	return indexes
}

func findBestLocalUsage(bill *adminplusdomain.SupplierBillLine, indexes localIndexes, used map[int64]struct{}, tolerance time.Duration) *adminplusdomain.LocalUsageLine {
	if key := normalizeKey(bill.ExternalRequestID); key != "" {
		if usage := firstUnused(indexes.byRequestID[key], used); usage != nil {
			return usage
		}
	}
	candidates := indexes.byModel[normalizeKey(bill.Model)]
	var best *adminplusdomain.LocalUsageLine
	var bestDelta time.Duration
	for _, usage := range candidates {
		if usage == nil {
			continue
		}
		if _, ok := used[usage.ID]; ok {
			continue
		}
		delta := usage.StartedAt.Sub(bill.StartedAt)
		if delta < 0 {
			delta = -delta
		}
		if delta > tolerance {
			continue
		}
		if best == nil || delta < bestDelta {
			best = usage
			bestDelta = delta
		}
	}
	return best
}

func firstUnused(items []*adminplusdomain.LocalUsageLine, used map[int64]struct{}) *adminplusdomain.LocalUsageLine {
	for _, item := range items {
		if item == nil {
			continue
		}
		if _, ok := used[item.ID]; ok {
			continue
		}
		return item
	}
	return nil
}

func matchedLine(bill *adminplusdomain.SupplierBillLine, usage *adminplusdomain.LocalUsageLine, mismatchCents int64) *adminplusdomain.ReconciliationLine {
	status := adminplusdomain.ReconciliationStatusMatched
	notes := ""
	currency := normalizeCurrency(bill.Currency)
	if normalizeCurrency(usage.Currency) != currency {
		status = adminplusdomain.ReconciliationStatusCurrencyMismatch
		notes = "currency mismatch"
	} else if usage.RevenueCents < bill.CostCents {
		status = adminplusdomain.ReconciliationStatusCostMismatch
		notes = "revenue is below supplier cost"
	} else if mismatchCents > 0 && absInt64(usage.RevenueCents-bill.CostCents) <= mismatchCents {
		status = adminplusdomain.ReconciliationStatusCostMismatch
		notes = "revenue is too close to supplier cost"
	}
	profit := usage.RevenueCents - bill.CostCents
	return &adminplusdomain.ReconciliationLine{
		Status:            status,
		SupplierBillID:    bill.ID,
		LocalUsageID:      usage.ID,
		ExternalRequestID: firstNonEmpty(bill.ExternalRequestID, usage.ExternalRequestID),
		Model:             firstNonEmpty(bill.Model, usage.Model),
		Currency:          currency,
		CostCents:         bill.CostCents,
		RevenueCents:      usage.RevenueCents,
		ProfitCents:       profit,
		ProfitMargin:      profitMargin(profit, usage.RevenueCents),
		Notes:             notes,
	}
}

func supplierOnlyLine(bill *adminplusdomain.SupplierBillLine) *adminplusdomain.ReconciliationLine {
	return &adminplusdomain.ReconciliationLine{
		Status:            adminplusdomain.ReconciliationStatusSupplierOnly,
		SupplierBillID:    bill.ID,
		ExternalRequestID: bill.ExternalRequestID,
		Model:             bill.Model,
		Currency:          normalizeCurrency(bill.Currency),
		CostCents:         bill.CostCents,
		RevenueCents:      0,
		ProfitCents:       -bill.CostCents,
		ProfitMargin:      nil,
		Notes:             "supplier bill has no matching local usage",
	}
}

func localOnlyLine(usage *adminplusdomain.LocalUsageLine) *adminplusdomain.ReconciliationLine {
	return &adminplusdomain.ReconciliationLine{
		Status:            adminplusdomain.ReconciliationStatusLocalOnly,
		LocalUsageID:      usage.ID,
		ExternalRequestID: usage.ExternalRequestID,
		Model:             usage.Model,
		Currency:          normalizeCurrency(usage.Currency),
		CostCents:         0,
		RevenueCents:      usage.RevenueCents,
		ProfitCents:       usage.RevenueCents,
		ProfitMargin:      profitMargin(usage.RevenueCents, usage.RevenueCents),
		Notes:             "local usage has no matching supplier bill",
	}
}

func summarize(lines []*adminplusdomain.ReconciliationLine, supplierCount int, localCount int) adminplusdomain.ReconciliationSummary {
	var summary adminplusdomain.ReconciliationSummary
	summary.TotalSupplierLines = int64(supplierCount)
	summary.TotalLocalLines = int64(localCount)
	for _, line := range lines {
		if line == nil {
			continue
		}
		switch line.Status {
		case adminplusdomain.ReconciliationStatusMatched:
			summary.MatchedLines++
		case adminplusdomain.ReconciliationStatusSupplierOnly:
			summary.SupplierOnlyLines++
		case adminplusdomain.ReconciliationStatusLocalOnly:
			summary.LocalOnlyLines++
		}
		summary.CostCents += line.CostCents
		summary.RevenueCents += line.RevenueCents
		summary.ProfitCents += line.ProfitCents
	}
	summary.ProfitMargin = profitMargin(summary.ProfitCents, summary.RevenueCents)
	return summary
}

func normalizeKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeCurrency(value string) string {
	v := strings.ToUpper(strings.TrimSpace(value))
	if len(v) != 3 {
		return "USD"
	}
	return v
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func profitMargin(profitCents int64, revenueCents int64) *float64 {
	if revenueCents == 0 {
		return nil
	}
	value := float64(profitCents) / float64(revenueCents)
	return &value
}

func absInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func badRequest(reason string, message string) error {
	return infraerrors.New(http.StatusBadRequest, reason, message)
}
