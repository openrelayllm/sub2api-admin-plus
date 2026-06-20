package rates

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type SQLRepository struct {
	db *sql.DB
}

func NewSQLRepository(db *sql.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) CreateSnapshot(ctx context.Context, snapshot *adminplusdomain.RateSnapshot) (*adminplusdomain.RateSnapshot, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	rawPayload, err := marshalRawPayload(snapshot.RawPayload)
	if err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO admin_plus_rate_snapshots (
			supplier_id, source, model, billing_mode, price_item,
			unit, currency, price_micros, raw_payload, captured_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, supplier_id, source, model, billing_mode, price_item,
			unit, currency, price_micros, raw_payload, captured_at, created_at
	`,
		snapshot.SupplierID,
		snapshot.Source,
		snapshot.Model,
		snapshot.BillingMode,
		snapshot.PriceItem,
		snapshot.Unit,
		snapshot.Currency,
		snapshot.PriceMicros,
		rawPayload,
		snapshot.CapturedAt,
	)
	return scanRateSnapshot(row)
}

func (r *SQLRepository) FindLatestComparableSnapshot(ctx context.Context, snapshot *adminplusdomain.RateSnapshot) (*adminplusdomain.RateSnapshot, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT id, supplier_id, source, model, billing_mode, price_item,
			unit, currency, price_micros, raw_payload, captured_at, created_at
		FROM admin_plus_rate_snapshots
		WHERE supplier_id = $1
			AND model = $2
			AND billing_mode = $3
			AND price_item = $4
			AND unit = $5
			AND currency = $6
			AND captured_at <= $7
		ORDER BY captured_at DESC, id DESC
		LIMIT 1
	`,
		snapshot.SupplierID,
		snapshot.Model,
		snapshot.BillingMode,
		snapshot.PriceItem,
		snapshot.Unit,
		snapshot.Currency,
		snapshot.CapturedAt,
	)
	previous, err := scanRateSnapshot(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return previous, err
}

func (r *SQLRepository) CreateChangeEvent(ctx context.Context, event *adminplusdomain.RateChangeEvent) (*adminplusdomain.RateChangeEvent, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO admin_plus_rate_change_events (
			supplier_id, snapshot_id, model, billing_mode, price_item,
			unit, currency, old_price_micros, new_price_micros, direction,
			change_percent, threshold_percent, threshold_exceeded, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, supplier_id, snapshot_id, model, billing_mode, price_item,
			unit, currency, old_price_micros, new_price_micros, direction,
			change_percent, threshold_percent, threshold_exceeded, status,
			created_at, acknowledged_at
	`,
		event.SupplierID,
		event.SnapshotID,
		event.Model,
		event.BillingMode,
		event.PriceItem,
		event.Unit,
		event.Currency,
		nullableInt64(event.OldPriceMicros),
		event.NewPriceMicros,
		string(event.Direction),
		nullableFloat64(event.ChangePercent),
		event.ThresholdPercent,
		event.ThresholdExceeded,
		string(event.Status),
	)
	return scanRateChangeEvent(row)
}

func (r *SQLRepository) ListSnapshots(ctx context.Context, filter SnapshotFilter) ([]*adminplusdomain.RateSnapshot, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	where := []string{"1=1"}
	args := make([]any, 0, 3)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.SupplierID > 0 {
		where = append(where, "supplier_id = "+addArg(filter.SupplierID))
	}
	if filter.Model != "" {
		where = append(where, "model = "+addArg(filter.Model))
	}
	limitRef := addArg(filter.Limit)

	query := `
		SELECT id, supplier_id, source, model, billing_mode, price_item,
			unit, currency, price_micros, raw_payload, captured_at, created_at
		FROM admin_plus_rate_snapshots
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY captured_at DESC, id DESC
		LIMIT ` + limitRef

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	items := make([]*adminplusdomain.RateSnapshot, 0)
	for rows.Next() {
		item, err := scanRateSnapshot(rows)
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

func (r *SQLRepository) ListChangeEvents(ctx context.Context, filter EventFilter) ([]*adminplusdomain.RateChangeEvent, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	where := []string{"1=1"}
	args := make([]any, 0, 3)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.SupplierID > 0 {
		where = append(where, "supplier_id = "+addArg(filter.SupplierID))
	}
	if filter.Status != "" {
		where = append(where, "status = "+addArg(string(filter.Status)))
	}
	limitRef := addArg(filter.Limit)

	query := `
		SELECT id, supplier_id, snapshot_id, model, billing_mode, price_item,
			unit, currency, old_price_micros, new_price_micros, direction,
			change_percent, threshold_percent, threshold_exceeded, status,
			created_at, acknowledged_at
		FROM admin_plus_rate_change_events
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY created_at DESC, id DESC
		LIMIT ` + limitRef

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	items := make([]*adminplusdomain.RateChangeEvent, 0)
	for rows.Next() {
		item, err := scanRateChangeEvent(rows)
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

func (r *SQLRepository) UpdateChangeEventStatus(ctx context.Context, id int64, status adminplusdomain.RateChangeStatus) (*adminplusdomain.RateChangeEvent, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	row := r.db.QueryRowContext(ctx, `
		UPDATE admin_plus_rate_change_events
		SET status = $2,
			acknowledged_at = CASE WHEN $2 = 'acknowledged' THEN NOW() ELSE NULL END
		WHERE id = $1
		RETURNING id, supplier_id, snapshot_id, model, billing_mode, price_item,
			unit, currency, old_price_micros, new_price_micros, direction,
			change_percent, threshold_percent, threshold_exceeded, status,
			created_at, acknowledged_at
	`, id, string(status))
	event, err := scanRateChangeEvent(row)
	if err == sql.ErrNoRows {
		return nil, infraerrors.New(http.StatusNotFound, "RATE_EVENT_NOT_FOUND", "rate event not found")
	}
	return event, err
}

type rateScanner interface {
	Scan(dest ...any) error
}

func scanRateSnapshot(scanner rateScanner) (*adminplusdomain.RateSnapshot, error) {
	var snapshot adminplusdomain.RateSnapshot
	var rawPayload []byte
	err := scanner.Scan(
		&snapshot.ID,
		&snapshot.SupplierID,
		&snapshot.Source,
		&snapshot.Model,
		&snapshot.BillingMode,
		&snapshot.PriceItem,
		&snapshot.Unit,
		&snapshot.Currency,
		&snapshot.PriceMicros,
		&rawPayload,
		&snapshot.CapturedAt,
		&snapshot.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(rawPayload) > 0 {
		var payload map[string]any
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return nil, err
		}
		snapshot.RawPayload = payload
	}
	return &snapshot, nil
}

func scanRateChangeEvent(scanner rateScanner) (*adminplusdomain.RateChangeEvent, error) {
	var event adminplusdomain.RateChangeEvent
	var direction string
	var status string
	var oldPrice sql.NullInt64
	var changePercent sql.NullFloat64
	var acknowledgedAt sql.NullTime
	err := scanner.Scan(
		&event.ID,
		&event.SupplierID,
		&event.SnapshotID,
		&event.Model,
		&event.BillingMode,
		&event.PriceItem,
		&event.Unit,
		&event.Currency,
		&oldPrice,
		&event.NewPriceMicros,
		&direction,
		&changePercent,
		&event.ThresholdPercent,
		&event.ThresholdExceeded,
		&status,
		&event.CreatedAt,
		&acknowledgedAt,
	)
	if err != nil {
		return nil, err
	}
	event.Direction = adminplusdomain.RateChangeDirection(direction)
	event.Status = adminplusdomain.RateChangeStatus(status)
	if oldPrice.Valid {
		v := oldPrice.Int64
		event.OldPriceMicros = &v
	}
	if changePercent.Valid {
		v := changePercent.Float64
		event.ChangePercent = &v
	}
	if acknowledgedAt.Valid {
		t := acknowledgedAt.Time
		event.AcknowledgedAt = &t
	}
	return &event, nil
}

func marshalRawPayload(payload map[string]any) ([]byte, error) {
	if len(payload) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(payload)
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

func dbNotConfigured() error {
	return infraerrors.New(http.StatusInternalServerError, "ADMIN_PLUS_DB_NOT_CONFIGURED", "admin plus database is not configured")
}
