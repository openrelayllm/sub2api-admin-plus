package kanban

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
)

type SQLRepository struct {
	db *sql.DB
}

func NewSQLRepository(db *sql.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) CreateMarketPriceSnapshot(ctx context.Context, snapshot *adminplusdomain.MarketPriceSnapshot) (*adminplusdomain.MarketPriceSnapshot, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	rawPayload, err := marshalRawPayload(snapshot.RawPayload)
	if err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO admin_plus_market_price_snapshots (
			source_type, source_name, source_url, site_id, supplier_id,
			model, billing_mode, price_item, unit, currency, price_micros,
			package_label, package_price_cents, package_quota, rate_multiplier,
			min_recharge_cents, bonus_percent, confidence, observed_at, raw_payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING `+marketPriceColumns(),
		snapshot.SourceType,
		snapshot.SourceName,
		snapshot.SourceURL,
		nullablePositiveID(snapshot.SiteID),
		nullablePositiveID(snapshot.SupplierID),
		snapshot.Model,
		snapshot.BillingMode,
		snapshot.PriceItem,
		snapshot.Unit,
		snapshot.Currency,
		snapshot.PriceMicros,
		snapshot.PackageLabel,
		nullableInt64(snapshot.PackagePriceCents),
		snapshot.PackageQuota,
		nullableFloat64(snapshot.RateMultiplier),
		nullableInt64(snapshot.MinRechargeCents),
		nullableFloat64(snapshot.BonusPercent),
		snapshot.Confidence,
		snapshot.ObservedAt,
		rawPayload,
	)
	return scanMarketPriceSnapshot(row)
}

func (r *SQLRepository) ListMarketPriceSnapshots(ctx context.Context, filter MarketPriceFilter) ([]*adminplusdomain.MarketPriceSnapshot, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	where := []string{"1=1"}
	args := make([]any, 0, 5)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.Model != "" {
		where = append(where, "model = "+addArg(filter.Model))
	}
	if filter.SourceType != "" {
		where = append(where, "source_type = "+addArg(filter.SourceType))
	}
	if filter.SiteID > 0 {
		where = append(where, "site_id = "+addArg(filter.SiteID))
	}
	if filter.SupplierID > 0 {
		where = append(where, "supplier_id = "+addArg(filter.SupplierID))
	}
	limitRef := addArg(filter.Limit)
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+marketPriceColumns()+`
		FROM admin_plus_market_price_snapshots
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY observed_at DESC, id DESC
		LIMIT `+limitRef, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]*adminplusdomain.MarketPriceSnapshot, 0)
	for rows.Next() {
		item, err := scanMarketPriceSnapshot(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *SQLRepository) CreateCacheEfficiencySnapshot(ctx context.Context, snapshot *adminplusdomain.CacheEfficiencySnapshot) (*adminplusdomain.CacheEfficiencySnapshot, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	rawPayload, err := marshalRawPayload(snapshot.RawPayload)
	if err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO admin_plus_cache_efficiency_snapshots (
			supply_type, supplier_id, local_sub2api_account_id, model,
			routing_strategy, sticky_scope, sample_requests, cache_read_tokens,
			cache_write_tokens, input_tokens, output_tokens, cache_hit_ratio,
			duplicate_input_tokens, estimated_waste_cents, avg_ttft_ms,
			avg_total_latency_ms, status, notes, observed_at, raw_payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING `+cacheEfficiencyColumns(),
		snapshot.SupplyType,
		nullablePositiveID(snapshot.SupplierID),
		nullablePositiveID(snapshot.LocalSub2APIAccountID),
		snapshot.Model,
		snapshot.RoutingStrategy,
		snapshot.StickyScope,
		snapshot.SampleRequests,
		snapshot.CacheReadTokens,
		snapshot.CacheWriteTokens,
		snapshot.InputTokens,
		snapshot.OutputTokens,
		snapshot.CacheHitRatio,
		snapshot.DuplicateInputTokens,
		snapshot.EstimatedWasteCents,
		nullableInt64(snapshot.AvgTTFTMS),
		nullableInt64(snapshot.AvgTotalLatencyMS),
		snapshot.Status,
		snapshot.Notes,
		snapshot.ObservedAt,
		rawPayload,
	)
	return scanCacheEfficiencySnapshot(row)
}

func (r *SQLRepository) ListCacheEfficiencySnapshots(ctx context.Context, filter CacheEfficiencyFilter) ([]*adminplusdomain.CacheEfficiencySnapshot, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	where := []string{"1=1"}
	args := make([]any, 0, 6)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.Model != "" {
		where = append(where, "model = "+addArg(filter.Model))
	}
	if filter.SupplyType != "" {
		where = append(where, "supply_type = "+addArg(filter.SupplyType))
	}
	if filter.SupplierID > 0 {
		where = append(where, "supplier_id = "+addArg(filter.SupplierID))
	}
	if filter.LocalSub2APIAccountID > 0 {
		where = append(where, "local_sub2api_account_id = "+addArg(filter.LocalSub2APIAccountID))
	}
	if filter.Status != "" {
		where = append(where, "status = "+addArg(filter.Status))
	}
	limitRef := addArg(filter.Limit)
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+cacheEfficiencyColumns()+`
		FROM admin_plus_cache_efficiency_snapshots
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY observed_at DESC, id DESC
		LIMIT `+limitRef, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]*adminplusdomain.CacheEfficiencySnapshot, 0)
	for rows.Next() {
		item, err := scanCacheEfficiencySnapshot(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *SQLRepository) CreateSupplyQualitySnapshot(ctx context.Context, snapshot *adminplusdomain.SupplyQualitySnapshot) (*adminplusdomain.SupplyQualitySnapshot, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	rawPayload, err := marshalRawPayload(snapshot.RawPayload)
	if err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO admin_plus_supply_quality_snapshots (
			supply_type, supplier_id, local_sub2api_account_id, model,
			availability_ratio, error_ratio, avg_ttft_ms, avg_total_latency_ms,
			cache_hit_ratio, purity_score, usage_trust_score, balance_risk_score,
			concurrency_score, quality_score, decision, notes, observed_at, raw_payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING `+supplyQualityColumns(),
		snapshot.SupplyType,
		nullablePositiveID(snapshot.SupplierID),
		nullablePositiveID(snapshot.LocalSub2APIAccountID),
		snapshot.Model,
		snapshot.AvailabilityRatio,
		snapshot.ErrorRatio,
		nullableInt64(snapshot.AvgTTFTMS),
		nullableInt64(snapshot.AvgTotalLatencyMS),
		snapshot.CacheHitRatio,
		snapshot.PurityScore,
		snapshot.UsageTrustScore,
		snapshot.BalanceRiskScore,
		snapshot.ConcurrencyScore,
		snapshot.QualityScore,
		snapshot.Decision,
		snapshot.Notes,
		snapshot.ObservedAt,
		rawPayload,
	)
	return scanSupplyQualitySnapshot(row)
}

func (r *SQLRepository) ListSupplyQualitySnapshots(ctx context.Context, filter SupplyQualityFilter) ([]*adminplusdomain.SupplyQualitySnapshot, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	where := []string{"1=1"}
	args := make([]any, 0, 6)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.Model != "" {
		where = append(where, "model = "+addArg(filter.Model))
	}
	if filter.SupplyType != "" {
		where = append(where, "supply_type = "+addArg(filter.SupplyType))
	}
	if filter.SupplierID > 0 {
		where = append(where, "supplier_id = "+addArg(filter.SupplierID))
	}
	if filter.LocalSub2APIAccountID > 0 {
		where = append(where, "local_sub2api_account_id = "+addArg(filter.LocalSub2APIAccountID))
	}
	if filter.Decision != "" {
		where = append(where, "decision = "+addArg(filter.Decision))
	}
	limitRef := addArg(filter.Limit)
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+supplyQualityColumns()+`
		FROM admin_plus_supply_quality_snapshots
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY observed_at DESC, id DESC
		LIMIT `+limitRef, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]*adminplusdomain.SupplyQualitySnapshot, 0)
	for rows.Next() {
		item, err := scanSupplyQualitySnapshot(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *SQLRepository) CreateAcceptanceReport(ctx context.Context, report *adminplusdomain.AcceptanceReport) (*adminplusdomain.AcceptanceReport, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	reportPayload, err := marshalRawPayload(report.ReportPayload)
	if err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO admin_plus_acceptance_reports (
			supply_type, supplier_id, local_sub2api_account_id, model, status,
			connectivity_status, model_list_status, purity_status, trial_call_status,
			usage_metering_status, cache_audit_status, balance_status, concurrency_status,
			failure_reason, recommendation, report_payload, observed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING `+acceptanceReportColumns(),
		report.SupplyType,
		nullablePositiveID(report.SupplierID),
		nullablePositiveID(report.LocalSub2APIAccountID),
		report.Model,
		report.Status,
		report.ConnectivityStatus,
		report.ModelListStatus,
		report.PurityStatus,
		report.TrialCallStatus,
		report.UsageMeteringStatus,
		report.CacheAuditStatus,
		report.BalanceStatus,
		report.ConcurrencyStatus,
		report.FailureReason,
		report.Recommendation,
		reportPayload,
		report.ObservedAt,
	)
	return scanAcceptanceReport(row)
}

func (r *SQLRepository) ListAcceptanceReports(ctx context.Context, filter AcceptanceReportFilter) ([]*adminplusdomain.AcceptanceReport, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	where := []string{"1=1"}
	args := make([]any, 0, 6)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.Model != "" {
		where = append(where, "model = "+addArg(filter.Model))
	}
	if filter.SupplyType != "" {
		where = append(where, "supply_type = "+addArg(filter.SupplyType))
	}
	if filter.SupplierID > 0 {
		where = append(where, "supplier_id = "+addArg(filter.SupplierID))
	}
	if filter.LocalSub2APIAccountID > 0 {
		where = append(where, "local_sub2api_account_id = "+addArg(filter.LocalSub2APIAccountID))
	}
	if filter.Status != "" {
		where = append(where, "status = "+addArg(filter.Status))
	}
	limitRef := addArg(filter.Limit)
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+acceptanceReportColumns()+`
		FROM admin_plus_acceptance_reports
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY observed_at DESC, id DESC
		LIMIT `+limitRef, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]*adminplusdomain.AcceptanceReport, 0)
	for rows.Next() {
		item, err := scanAcceptanceReport(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *SQLRepository) CreateKanbanEvent(ctx context.Context, event *adminplusdomain.KanbanEvent) (*adminplusdomain.KanbanEvent, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	payload, err := marshalRawPayload(event.Payload)
	if err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO admin_plus_kanban_events (
			event_type, severity, status, model, source_type, source_id,
			related_snapshot_type, related_snapshot_id, title, description,
			recommendation, payload, occurred_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING `+kanbanEventColumns(),
		event.EventType,
		event.Severity,
		event.Status,
		event.Model,
		event.SourceType,
		nullablePositiveID(event.SourceID),
		event.RelatedSnapshotType,
		nullablePositiveID(event.RelatedSnapshotID),
		event.Title,
		event.Description,
		event.Recommendation,
		payload,
		event.OccurredAt,
	)
	return scanKanbanEvent(row)
}

func (r *SQLRepository) ListKanbanEvents(ctx context.Context, filter KanbanEventFilter) ([]*adminplusdomain.KanbanEvent, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	where := []string{"1=1"}
	args := make([]any, 0, 5)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.Model != "" {
		where = append(where, "model = "+addArg(filter.Model))
	}
	if filter.EventType != "" {
		where = append(where, "event_type = "+addArg(filter.EventType))
	}
	if filter.Severity != "" {
		where = append(where, "severity = "+addArg(filter.Severity))
	}
	if filter.Status != "" {
		where = append(where, "status = "+addArg(filter.Status))
	}
	limitRef := addArg(filter.Limit)
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+kanbanEventColumns()+`
		FROM admin_plus_kanban_events
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY occurred_at DESC, id DESC
		LIMIT `+limitRef, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]*adminplusdomain.KanbanEvent, 0)
	for rows.Next() {
		item, err := scanKanbanEvent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *SQLRepository) UpdateKanbanEventStatus(ctx context.Context, id int64, status string) (*adminplusdomain.KanbanEvent, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	row := r.db.QueryRowContext(ctx, `
		UPDATE admin_plus_kanban_events
		SET status = $2
		WHERE id = $1
		RETURNING `+kanbanEventColumns(), id, status)
	return scanKanbanEvent(row)
}

func (r *SQLRepository) ListSupplierRateCosts(ctx context.Context, model string, limit int) ([]*SupplierRateCost, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	where := []string{"price_micros >= 0"}
	args := make([]any, 0, 2)
	if strings.TrimSpace(model) != "" {
		args = append(args, strings.TrimSpace(model))
		where = append(where, fmt.Sprintf("model = $%d", len(args)))
	}
	args = append(args, limit)
	limitRef := fmt.Sprintf("$%d", len(args))
	rows, err := r.db.QueryContext(ctx, `
		SELECT supplier_id, model, currency, price_micros, captured_at
		FROM (
			SELECT DISTINCT ON (supplier_id, model, currency, billing_mode, price_item, unit)
				supplier_id, model, currency, price_micros, captured_at
			FROM admin_plus_rate_snapshots
			WHERE `+strings.Join(where, " AND ")+`
			ORDER BY supplier_id, model, currency, billing_mode, price_item, unit, captured_at DESC, id DESC
		) latest
		ORDER BY captured_at DESC
		LIMIT `+limitRef, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]*SupplierRateCost, 0)
	for rows.Next() {
		var item SupplierRateCost
		if err := rows.Scan(&item.SupplierID, &item.Model, &item.Currency, &item.PriceMicros, &item.CapturedAt); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanMarketPriceSnapshot(scanner scanner) (*adminplusdomain.MarketPriceSnapshot, error) {
	var item adminplusdomain.MarketPriceSnapshot
	var siteID sql.NullInt64
	var supplierID sql.NullInt64
	var packagePrice sql.NullInt64
	var rateMultiplier sql.NullFloat64
	var minRecharge sql.NullInt64
	var bonusPercent sql.NullFloat64
	var rawPayload []byte
	err := scanner.Scan(
		&item.ID,
		&item.SourceType,
		&item.SourceName,
		&item.SourceURL,
		&siteID,
		&supplierID,
		&item.Model,
		&item.BillingMode,
		&item.PriceItem,
		&item.Unit,
		&item.Currency,
		&item.PriceMicros,
		&item.PackageLabel,
		&packagePrice,
		&item.PackageQuota,
		&rateMultiplier,
		&minRecharge,
		&bonusPercent,
		&item.Confidence,
		&item.ObservedAt,
		&rawPayload,
		&item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if siteID.Valid {
		item.SiteID = siteID.Int64
	}
	if supplierID.Valid {
		item.SupplierID = supplierID.Int64
	}
	if packagePrice.Valid {
		v := packagePrice.Int64
		item.PackagePriceCents = &v
	}
	if rateMultiplier.Valid {
		v := rateMultiplier.Float64
		item.RateMultiplier = &v
	}
	if minRecharge.Valid {
		v := minRecharge.Int64
		item.MinRechargeCents = &v
	}
	if bonusPercent.Valid {
		v := bonusPercent.Float64
		item.BonusPercent = &v
	}
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &item.RawPayload); err != nil {
			return nil, err
		}
	}
	return &item, nil
}

func scanCacheEfficiencySnapshot(scanner scanner) (*adminplusdomain.CacheEfficiencySnapshot, error) {
	var item adminplusdomain.CacheEfficiencySnapshot
	var supplierID sql.NullInt64
	var accountID sql.NullInt64
	var avgTTFT sql.NullInt64
	var avgTotal sql.NullInt64
	var rawPayload []byte
	err := scanner.Scan(
		&item.ID,
		&item.SupplyType,
		&supplierID,
		&accountID,
		&item.Model,
		&item.RoutingStrategy,
		&item.StickyScope,
		&item.SampleRequests,
		&item.CacheReadTokens,
		&item.CacheWriteTokens,
		&item.InputTokens,
		&item.OutputTokens,
		&item.CacheHitRatio,
		&item.DuplicateInputTokens,
		&item.EstimatedWasteCents,
		&avgTTFT,
		&avgTotal,
		&item.Status,
		&item.Notes,
		&item.ObservedAt,
		&rawPayload,
		&item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if supplierID.Valid {
		item.SupplierID = supplierID.Int64
	}
	if accountID.Valid {
		item.LocalSub2APIAccountID = accountID.Int64
	}
	if avgTTFT.Valid {
		v := avgTTFT.Int64
		item.AvgTTFTMS = &v
	}
	if avgTotal.Valid {
		v := avgTotal.Int64
		item.AvgTotalLatencyMS = &v
	}
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &item.RawPayload); err != nil {
			return nil, err
		}
	}
	return &item, nil
}

func scanSupplyQualitySnapshot(scanner scanner) (*adminplusdomain.SupplyQualitySnapshot, error) {
	var item adminplusdomain.SupplyQualitySnapshot
	var supplierID sql.NullInt64
	var accountID sql.NullInt64
	var avgTTFT sql.NullInt64
	var avgTotal sql.NullInt64
	var rawPayload []byte
	err := scanner.Scan(
		&item.ID,
		&item.SupplyType,
		&supplierID,
		&accountID,
		&item.Model,
		&item.AvailabilityRatio,
		&item.ErrorRatio,
		&avgTTFT,
		&avgTotal,
		&item.CacheHitRatio,
		&item.PurityScore,
		&item.UsageTrustScore,
		&item.BalanceRiskScore,
		&item.ConcurrencyScore,
		&item.QualityScore,
		&item.Decision,
		&item.Notes,
		&item.ObservedAt,
		&rawPayload,
		&item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if supplierID.Valid {
		item.SupplierID = supplierID.Int64
	}
	if accountID.Valid {
		item.LocalSub2APIAccountID = accountID.Int64
	}
	if avgTTFT.Valid {
		v := avgTTFT.Int64
		item.AvgTTFTMS = &v
	}
	if avgTotal.Valid {
		v := avgTotal.Int64
		item.AvgTotalLatencyMS = &v
	}
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &item.RawPayload); err != nil {
			return nil, err
		}
	}
	return &item, nil
}

func scanAcceptanceReport(scanner scanner) (*adminplusdomain.AcceptanceReport, error) {
	var item adminplusdomain.AcceptanceReport
	var supplierID sql.NullInt64
	var accountID sql.NullInt64
	var reportPayload []byte
	err := scanner.Scan(
		&item.ID,
		&item.SupplyType,
		&supplierID,
		&accountID,
		&item.Model,
		&item.Status,
		&item.ConnectivityStatus,
		&item.ModelListStatus,
		&item.PurityStatus,
		&item.TrialCallStatus,
		&item.UsageMeteringStatus,
		&item.CacheAuditStatus,
		&item.BalanceStatus,
		&item.ConcurrencyStatus,
		&item.FailureReason,
		&item.Recommendation,
		&reportPayload,
		&item.ObservedAt,
		&item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if supplierID.Valid {
		item.SupplierID = supplierID.Int64
	}
	if accountID.Valid {
		item.LocalSub2APIAccountID = accountID.Int64
	}
	if len(reportPayload) > 0 {
		if err := json.Unmarshal(reportPayload, &item.ReportPayload); err != nil {
			return nil, err
		}
	}
	return &item, nil
}

func scanKanbanEvent(scanner scanner) (*adminplusdomain.KanbanEvent, error) {
	var item adminplusdomain.KanbanEvent
	var sourceID sql.NullInt64
	var relatedID sql.NullInt64
	var payload []byte
	err := scanner.Scan(
		&item.ID,
		&item.EventType,
		&item.Severity,
		&item.Status,
		&item.Model,
		&item.SourceType,
		&sourceID,
		&item.RelatedSnapshotType,
		&relatedID,
		&item.Title,
		&item.Description,
		&item.Recommendation,
		&payload,
		&item.OccurredAt,
		&item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if sourceID.Valid {
		item.SourceID = sourceID.Int64
	}
	if relatedID.Valid {
		item.RelatedSnapshotID = relatedID.Int64
	}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &item.Payload); err != nil {
			return nil, err
		}
	}
	return &item, nil
}

func marketPriceColumns() string {
	return `id, source_type, source_name, source_url, site_id, supplier_id,
		model, billing_mode, price_item, unit, currency, price_micros,
		package_label, package_price_cents, package_quota, rate_multiplier,
		min_recharge_cents, bonus_percent, confidence, observed_at, raw_payload, created_at`
}

func cacheEfficiencyColumns() string {
	return `id, supply_type, supplier_id, local_sub2api_account_id, model,
		routing_strategy, sticky_scope, sample_requests, cache_read_tokens,
		cache_write_tokens, input_tokens, output_tokens, cache_hit_ratio,
		duplicate_input_tokens, estimated_waste_cents, avg_ttft_ms,
		avg_total_latency_ms, status, notes, observed_at, raw_payload, created_at`
}

func supplyQualityColumns() string {
	return `id, supply_type, supplier_id, local_sub2api_account_id, model,
		availability_ratio, error_ratio, avg_ttft_ms, avg_total_latency_ms,
		cache_hit_ratio, purity_score, usage_trust_score, balance_risk_score,
		concurrency_score, quality_score, decision, notes, observed_at, raw_payload, created_at`
}

func acceptanceReportColumns() string {
	return `id, supply_type, supplier_id, local_sub2api_account_id, model, status,
		connectivity_status, model_list_status, purity_status, trial_call_status,
		usage_metering_status, cache_audit_status, balance_status, concurrency_status,
		failure_reason, recommendation, report_payload, observed_at, created_at`
}

func kanbanEventColumns() string {
	return `id, event_type, severity, status, model, source_type, source_id,
		related_snapshot_type, related_snapshot_id, title, description,
		recommendation, payload, occurred_at, created_at`
}

func marshalRawPayload(payload map[string]any) ([]byte, error) {
	if len(payload) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(payload)
}

func nullablePositiveID(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableFloat64(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}
