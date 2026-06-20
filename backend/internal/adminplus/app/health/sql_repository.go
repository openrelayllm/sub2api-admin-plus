package health

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

func (r *SQLRepository) CreateSample(ctx context.Context, sample *adminplusdomain.HealthSample) (*adminplusdomain.HealthSample, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	rawPayload, err := marshalRawPayload(sample.RawPayload)
	if err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO admin_plus_health_samples (
			supplier_id, source, model, first_token_latency_ms, total_latency_ms,
			status_code, error_class, observed_concurrency, available_concurrency,
			concurrency_limit, raw_payload, captured_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, supplier_id, source, model, first_token_latency_ms, total_latency_ms,
			status_code, error_class, observed_concurrency, available_concurrency,
			concurrency_limit, raw_payload, captured_at, created_at
	`,
		sample.SupplierID,
		sample.Source,
		sample.Model,
		sample.FirstTokenLatencyMS,
		sample.TotalLatencyMS,
		sample.StatusCode,
		sample.ErrorClass,
		sample.ObservedConcurrency,
		nullableInt(sample.AvailableConcurrency),
		nullableInt(sample.ConcurrencyLimit),
		rawPayload,
		sample.CapturedAt,
	)
	return scanHealthSample(row)
}

func (r *SQLRepository) CreateEvent(ctx context.Context, event *adminplusdomain.HealthEvent) (*adminplusdomain.HealthEvent, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO admin_plus_health_events (
			supplier_id, sample_id, type, model, observed_value, threshold_value,
			status_code, error_class, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, supplier_id, sample_id, type, model, observed_value, threshold_value,
			status_code, error_class, status, created_at, acknowledged_at
	`,
		event.SupplierID,
		event.SampleID,
		string(event.Type),
		event.Model,
		event.ObservedValue,
		event.ThresholdValue,
		event.StatusCode,
		event.ErrorClass,
		string(event.Status),
	)
	return scanHealthEvent(row)
}

func (r *SQLRepository) ListSamples(ctx context.Context, filter SampleFilter) ([]*adminplusdomain.HealthSample, error) {
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
		SELECT id, supplier_id, source, model, first_token_latency_ms, total_latency_ms,
			status_code, error_class, observed_concurrency, available_concurrency,
			concurrency_limit, raw_payload, captured_at, created_at
		FROM admin_plus_health_samples
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY captured_at DESC, id DESC
		LIMIT ` + limitRef

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*adminplusdomain.HealthSample, 0)
	for rows.Next() {
		item, err := scanHealthSample(rows)
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

func (r *SQLRepository) ListEvents(ctx context.Context, filter EventFilter) ([]*adminplusdomain.HealthEvent, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	where := []string{"1=1"}
	args := make([]any, 0, 4)
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
	if filter.Type != "" {
		where = append(where, "type = "+addArg(string(filter.Type)))
	}
	limitRef := addArg(filter.Limit)
	query := `
		SELECT id, supplier_id, sample_id, type, model, observed_value, threshold_value,
			status_code, error_class, status, created_at, acknowledged_at
		FROM admin_plus_health_events
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY created_at DESC, id DESC
		LIMIT ` + limitRef

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*adminplusdomain.HealthEvent, 0)
	for rows.Next() {
		item, err := scanHealthEvent(rows)
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

func (r *SQLRepository) UpdateEventStatus(ctx context.Context, id int64, status adminplusdomain.HealthEventStatus) (*adminplusdomain.HealthEvent, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	row := r.db.QueryRowContext(ctx, `
		UPDATE admin_plus_health_events
		SET status = $2,
			acknowledged_at = CASE WHEN $2 = 'acknowledged' THEN NOW() ELSE NULL END
		WHERE id = $1
		RETURNING id, supplier_id, sample_id, type, model, observed_value, threshold_value,
			status_code, error_class, status, created_at, acknowledged_at
	`, id, string(status))
	event, err := scanHealthEvent(row)
	if err == sql.ErrNoRows {
		return nil, infraerrors.New(http.StatusNotFound, "HEALTH_EVENT_NOT_FOUND", "health event not found")
	}
	return event, err
}

type healthScanner interface {
	Scan(dest ...any) error
}

func scanHealthSample(scanner healthScanner) (*adminplusdomain.HealthSample, error) {
	var sample adminplusdomain.HealthSample
	var availableConcurrency, concurrencyLimit sql.NullInt64
	var rawPayload []byte
	err := scanner.Scan(
		&sample.ID,
		&sample.SupplierID,
		&sample.Source,
		&sample.Model,
		&sample.FirstTokenLatencyMS,
		&sample.TotalLatencyMS,
		&sample.StatusCode,
		&sample.ErrorClass,
		&sample.ObservedConcurrency,
		&availableConcurrency,
		&concurrencyLimit,
		&rawPayload,
		&sample.CapturedAt,
		&sample.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if availableConcurrency.Valid {
		v := int(availableConcurrency.Int64)
		sample.AvailableConcurrency = &v
	}
	if concurrencyLimit.Valid {
		v := int(concurrencyLimit.Int64)
		sample.ConcurrencyLimit = &v
	}
	if len(rawPayload) > 0 {
		var payload map[string]any
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return nil, err
		}
		sample.RawPayload = payload
	}
	return &sample, nil
}

func scanHealthEvent(scanner healthScanner) (*adminplusdomain.HealthEvent, error) {
	var event adminplusdomain.HealthEvent
	var eventType, status string
	var acknowledgedAt sql.NullTime
	err := scanner.Scan(
		&event.ID,
		&event.SupplierID,
		&event.SampleID,
		&eventType,
		&event.Model,
		&event.ObservedValue,
		&event.ThresholdValue,
		&event.StatusCode,
		&event.ErrorClass,
		&status,
		&event.CreatedAt,
		&acknowledgedAt,
	)
	if err != nil {
		return nil, err
	}
	event.Type = adminplusdomain.HealthEventType(eventType)
	event.Status = adminplusdomain.HealthEventStatus(status)
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

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func dbNotConfigured() error {
	return infraerrors.New(http.StatusInternalServerError, "ADMIN_PLUS_DB_NOT_CONFIGURED", "admin plus database is not configured")
}
