package extension

import (
	"context"
	"encoding/json"
	"math"
	"time"

	balancesapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/balances"
	billingapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/billing"
	healthapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/health"
	promotionsapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/promotions"
	ratesapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/rates"
	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
)

type ResultProcessor interface {
	ProcessTaskResult(ctx context.Context, task *adminplusdomain.ExtensionTask, result map[string]any) (map[string]any, error)
}

type IngestProcessor struct {
	rates      *ratesapp.Service
	balances   *balancesapp.Service
	promotions *promotionsapp.Service
	health     *healthapp.Service
	billing    *billingapp.Service
}

func NewIngestProcessor(
	rates *ratesapp.Service,
	balances *balancesapp.Service,
	promotions *promotionsapp.Service,
	health *healthapp.Service,
	billing *billingapp.Service,
) *IngestProcessor {
	return &IngestProcessor{
		rates:      rates,
		balances:   balances,
		promotions: promotions,
		health:     health,
		billing:    billing,
	}
}

func (p *IngestProcessor) ProcessTaskResult(ctx context.Context, task *adminplusdomain.ExtensionTask, result map[string]any) (map[string]any, error) {
	if p == nil || task == nil {
		return nil, nil
	}
	switch task.Type {
	case adminplusdomain.ExtensionTaskTypeFetchRates:
		return p.processRates(ctx, task, result)
	case adminplusdomain.ExtensionTaskTypeFetchBalance:
		return p.processBalance(ctx, task, result)
	case adminplusdomain.ExtensionTaskTypeFetchPromotions:
		return p.processPromotions(ctx, task, result)
	case adminplusdomain.ExtensionTaskTypeFetchHealth:
		return p.processHealth(ctx, task, result)
	case adminplusdomain.ExtensionTaskTypeExportBills:
		return p.processBills(ctx, task, result)
	default:
		return nil, nil
	}
}

func (p *IngestProcessor) processRates(ctx context.Context, task *adminplusdomain.ExtensionTask, result map[string]any) (map[string]any, error) {
	entriesRaw, ok := result["entries"].([]any)
	if !ok || len(entriesRaw) == 0 {
		return nil, nil
	}
	entries := make([]ratesapp.RateEntryInput, 0, len(entriesRaw))
	for _, raw := range entriesRaw {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		entries = append(entries, ratesapp.RateEntryInput{
			Model:       stringValue(item, "model"),
			BillingMode: stringValue(item, "billing_mode"),
			PriceItem:   stringValue(item, "price_item"),
			Unit:        stringValue(item, "unit"),
			Currency:    stringValue(item, "currency"),
			PriceMicros: int64Value(item, "price_micros"),
			RawPayload:  mapValue(item, "raw_payload"),
		})
	}
	if len(entries) == 0 {
		return nil, nil
	}
	capturedAt := optionalTimeValue(result, "captured_at")
	ingested, err := p.rates.RecordSnapshot(ctx, ratesapp.RecordSnapshotInput{
		SupplierID:       task.SupplierID,
		Source:           sourceValue(result),
		CapturedAt:       capturedAt,
		ThresholdPercent: float64Value(result, "threshold_percent"),
		Entries:          entries,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"rate_snapshots": len(ingested.Snapshots),
		"rate_events":    len(ingested.Events),
	}, nil
}

func (p *IngestProcessor) processBalance(ctx context.Context, task *adminplusdomain.ExtensionTask, result map[string]any) (map[string]any, error) {
	capturedAt := optionalTimeValue(result, "captured_at")
	event, snapshot, err := p.balances.RecordSnapshot(ctx, balancesapp.RecordSnapshotInput{
		SupplierID:               task.SupplierID,
		Source:                   sourceValue(result),
		RuntimeStatus:            adminplusdomain.SupplierRuntimeStatus(stringValue(result, "runtime_status")),
		BalanceCents:             int64Value(result, "balance_cents"),
		Currency:                 stringValue(result, "currency"),
		LowBalanceThresholdCents: int64Value(result, "low_balance_threshold_cents"),
		RawPayload:               mapValue(result, "raw_payload"),
		CapturedAt:               capturedAt,
	})
	if err != nil {
		return nil, err
	}
	out := map[string]any{"balance_snapshot_id": snapshot.ID}
	if event != nil {
		out["balance_event_id"] = event.ID
		out["balance_event_type"] = string(event.Type)
	}
	return out, nil
}

func (p *IngestProcessor) processPromotions(ctx context.Context, task *adminplusdomain.ExtensionTask, result map[string]any) (map[string]any, error) {
	itemsRaw, ok := result["promotions"].([]any)
	if !ok || len(itemsRaw) == 0 {
		return nil, nil
	}
	count := 0
	for _, raw := range itemsRaw {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		startsAt := optionalTimeValue(item, "starts_at")
		endsAt := optionalTimeValue(item, "ends_at")
		capturedAt := optionalTimeValue(item, "captured_at")
		_, err := p.promotions.RecordPromotion(ctx, promotionsapp.RecordPromotionInput{
			SupplierID:       task.SupplierID,
			Source:           sourceValue(result),
			Type:             adminplusdomain.PromotionType(stringValue(item, "type")),
			Title:            stringValue(item, "title"),
			Description:      stringValue(item, "description"),
			Currency:         stringValue(item, "currency"),
			MinRechargeCents: int64Value(item, "min_recharge_cents"),
			BonusPercent:     optionalFloat64Value(item, "bonus_percent"),
			DiscountPercent:  optionalFloat64Value(item, "discount_percent"),
			RuntimeStatus:    adminplusdomain.SupplierRuntimeStatus(stringValue(item, "runtime_status")),
			BalanceCents:     int64Value(item, "balance_cents"),
			StartsAt:         startsAt,
			EndsAt:           endsAt,
			CapturedAt:       capturedAt,
			RawPayload:       mapValue(item, "raw_payload"),
		})
		if err != nil {
			return nil, err
		}
		count++
	}
	return map[string]any{"promotion_events": count}, nil
}

func (p *IngestProcessor) processHealth(ctx context.Context, task *adminplusdomain.ExtensionTask, result map[string]any) (map[string]any, error) {
	capturedAt := optionalTimeValue(result, "captured_at")
	ingested, err := p.health.RecordSample(ctx, healthapp.RecordSampleInput{
		SupplierID:                   task.SupplierID,
		Source:                       sourceValue(result),
		Model:                        stringValue(result, "model"),
		FirstTokenLatencyMS:          int64Value(result, "first_token_latency_ms"),
		TotalLatencyMS:               int64Value(result, "total_latency_ms"),
		StatusCode:                   intValue(result, "status_code"),
		ErrorClass:                   stringValue(result, "error_class"),
		ObservedConcurrency:          intValue(result, "observed_concurrency"),
		AvailableConcurrency:         optionalIntValue(result, "available_concurrency"),
		ConcurrencyLimit:             optionalIntValue(result, "concurrency_limit"),
		FirstTokenThresholdMS:        int64Value(result, "first_token_threshold_ms"),
		TotalLatencyThresholdMS:      int64Value(result, "total_latency_threshold_ms"),
		ConcurrencySaturationPercent: float64Value(result, "concurrency_saturation_percent"),
		RawPayload:                   mapValue(result, "raw_payload"),
		CapturedAt:                   capturedAt,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"health_sample_id": ingested.Sample.ID,
		"health_events":    len(ingested.Events),
	}, nil
}

func (p *IngestProcessor) processBills(ctx context.Context, task *adminplusdomain.ExtensionTask, result map[string]any) (map[string]any, error) {
	linesRaw, ok := result["lines"].([]any)
	if !ok || len(linesRaw) == 0 {
		return nil, nil
	}
	lines := make([]billingapp.ImportBillLineInput, 0, len(linesRaw))
	for _, raw := range linesRaw {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		startedAt := timeValue(item, "started_at")
		if startedAt.IsZero() {
			continue
		}
		lines = append(lines, billingapp.ImportBillLineInput{
			SupplierID:        task.SupplierID,
			Source:            sourceValue(result),
			ExternalBillID:    stringValue(item, "external_bill_id"),
			ExternalRequestID: stringValue(item, "external_request_id"),
			Model:             stringValue(item, "model"),
			Currency:          stringValue(item, "currency"),
			CostCents:         int64Value(item, "cost_cents"),
			InputTokens:       int64Value(item, "input_tokens"),
			OutputTokens:      int64Value(item, "output_tokens"),
			StartedAt:         startedAt,
			EndedAt:           optionalTimeValue(item, "ended_at"),
			RawPayload:        mapValue(item, "raw_payload"),
		})
	}
	if len(lines) == 0 {
		return nil, nil
	}
	created, err := p.billing.ImportBillLines(ctx, lines)
	if err != nil {
		return nil, err
	}
	return map[string]any{"bill_lines": len(created)}, nil
}

func sourceValue(values map[string]any) string {
	value := stringValue(values, "source")
	if value == "" {
		return "chrome"
	}
	return value
}

func stringValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	if value, ok := values[key].(string); ok {
		return value
	}
	return ""
}

func intValue(values map[string]any, key string) int {
	return int(int64Value(values, key))
}

func int64Value(values map[string]any, key string) int64 {
	if values == nil {
		return 0
	}
	switch value := values[key].(type) {
	case int:
		return int64(value)
	case int64:
		return value
	case float64:
		return int64(math.Round(value))
	case json.Number:
		parsed, err := value.Int64()
		if err == nil {
			return parsed
		}
		parsedFloat, err := value.Float64()
		if err == nil {
			return int64(math.Round(parsedFloat))
		}
		return 0
	default:
		return 0
	}
}

func float64Value(values map[string]any, key string) float64 {
	if values == nil {
		return 0
	}
	switch value := values[key].(type) {
	case float64:
		return value
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case json.Number:
		parsed, err := value.Float64()
		if err == nil {
			return parsed
		}
		return 0
	default:
		return 0
	}
}

func optionalFloat64Value(values map[string]any, key string) *float64 {
	if values == nil {
		return nil
	}
	if _, ok := values[key]; !ok || values[key] == nil {
		return nil
	}
	v := float64Value(values, key)
	return &v
}

func optionalIntValue(values map[string]any, key string) *int {
	if values == nil {
		return nil
	}
	if _, ok := values[key]; !ok || values[key] == nil {
		return nil
	}
	v := intValue(values, key)
	return &v
}

func mapValue(values map[string]any, key string) map[string]any {
	if values == nil {
		return nil
	}
	if value, ok := values[key].(map[string]any); ok {
		return value
	}
	return nil
}

func optionalTimeValue(values map[string]any, key string) *time.Time {
	t := timeValue(values, key)
	if t.IsZero() {
		return nil
	}
	return &t
}

func timeValue(values map[string]any, key string) time.Time {
	raw := stringValue(values, key)
	if raw == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func mergeIngestResult(result map[string]any, ingest map[string]any) map[string]any {
	if len(ingest) == 0 {
		return result
	}
	out := make(map[string]any, len(result)+1)
	for key, value := range result {
		out[key] = value
	}
	out["ingest"] = ingest
	return out
}

func ingestError(err error) error {
	return err
}
