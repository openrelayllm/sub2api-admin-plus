package domain

import "time"

type SupplierBillLine struct {
	ID                int64          `json:"id"`
	SupplierID        int64          `json:"supplier_id"`
	Source            string         `json:"source"`
	ExternalBillID    string         `json:"external_bill_id,omitempty"`
	ExternalRequestID string         `json:"external_request_id,omitempty"`
	Model             string         `json:"model"`
	Currency          string         `json:"currency"`
	CostCents         int64          `json:"cost_cents"`
	InputTokens       int64          `json:"input_tokens"`
	OutputTokens      int64          `json:"output_tokens"`
	StartedAt         time.Time      `json:"started_at"`
	EndedAt           *time.Time     `json:"ended_at,omitempty"`
	RawPayload        map[string]any `json:"raw_payload,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
}

type LocalUsageLine struct {
	ID                int64     `json:"id"`
	ExternalRequestID string    `json:"external_request_id,omitempty"`
	Model             string    `json:"model"`
	Currency          string    `json:"currency"`
	RevenueCents      int64     `json:"revenue_cents"`
	InputTokens       int64     `json:"input_tokens"`
	OutputTokens      int64     `json:"output_tokens"`
	StartedAt         time.Time `json:"started_at"`
}

type ReconciliationStatus string

const (
	ReconciliationStatusMatched          ReconciliationStatus = "matched"
	ReconciliationStatusSupplierOnly     ReconciliationStatus = "supplier_only"
	ReconciliationStatusLocalOnly        ReconciliationStatus = "local_only"
	ReconciliationStatusCurrencyMismatch ReconciliationStatus = "currency_mismatch"
	ReconciliationStatusCostMismatch     ReconciliationStatus = "cost_mismatch"
)

type ReconciliationLine struct {
	Status            ReconciliationStatus `json:"status"`
	SupplierBillID    int64                `json:"supplier_bill_id,omitempty"`
	LocalUsageID      int64                `json:"local_usage_id,omitempty"`
	ExternalRequestID string               `json:"external_request_id,omitempty"`
	Model             string               `json:"model"`
	Currency          string               `json:"currency"`
	CostCents         int64                `json:"cost_cents"`
	RevenueCents      int64                `json:"revenue_cents"`
	ProfitCents       int64                `json:"profit_cents"`
	ProfitMargin      *float64             `json:"profit_margin,omitempty"`
	Notes             string               `json:"notes,omitempty"`
}

type ReconciliationSummary struct {
	TotalSupplierLines int64    `json:"total_supplier_lines"`
	TotalLocalLines    int64    `json:"total_local_lines"`
	MatchedLines       int64    `json:"matched_lines"`
	SupplierOnlyLines  int64    `json:"supplier_only_lines"`
	LocalOnlyLines     int64    `json:"local_only_lines"`
	CostCents          int64    `json:"cost_cents"`
	RevenueCents       int64    `json:"revenue_cents"`
	ProfitCents        int64    `json:"profit_cents"`
	ProfitMargin       *float64 `json:"profit_margin,omitempty"`
}
