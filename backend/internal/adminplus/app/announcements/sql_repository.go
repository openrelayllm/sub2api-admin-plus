package announcements

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type SQLRepository struct {
	db *sql.DB
}

func NewSQLRepository(db *sql.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) CreateEvent(ctx context.Context, event *adminplusdomain.AnnouncementEvent) (*adminplusdomain.AnnouncementEvent, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	rawPayload, err := marshalRawPayload(event.RawPayload)
	if err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO admin_plus_announcement_events (
			supplier_id, source, type, title, description, currency,
			min_recharge_cents, bonus_percent, discount_percent, runtime_status,
			balance_cents, switch_eligible, recommendation, status,
			starts_at, ends_at, captured_at, raw_payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id, supplier_id, source, type, title, description, currency,
			min_recharge_cents, bonus_percent, discount_percent, runtime_status,
			balance_cents, switch_eligible, recommendation, status,
			starts_at, ends_at, captured_at, created_at, acknowledged_at, raw_payload
	`,
		event.SupplierID,
		event.Source,
		string(event.Type),
		event.Title,
		event.Description,
		event.Currency,
		event.MinRechargeCents,
		nullableFloat64(event.BonusPercent),
		nullableFloat64(event.DiscountPercent),
		string(event.RuntimeStatus),
		event.BalanceCents,
		event.SwitchEligible,
		string(event.Recommendation),
		string(event.Status),
		nullableTime(event.StartsAt),
		nullableTime(event.EndsAt),
		event.CapturedAt,
		rawPayload,
	)
	return scanAnnouncementEvent(row)
}

func (r *SQLRepository) ListEvents(ctx context.Context, filter EventFilter) ([]*adminplusdomain.AnnouncementEvent, error) {
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
	if filter.Recommendation != "" {
		where = append(where, "recommendation = "+addArg(string(filter.Recommendation)))
	}
	limitRef := addArg(filter.Limit)
	query := `
		SELECT id, supplier_id, source, type, title, description, currency,
			min_recharge_cents, bonus_percent, discount_percent, runtime_status,
			balance_cents, switch_eligible, recommendation, status,
			starts_at, ends_at, captured_at, created_at, acknowledged_at, raw_payload
		FROM admin_plus_announcement_events
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY created_at DESC, id DESC
		LIMIT ` + limitRef

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*adminplusdomain.AnnouncementEvent, 0)
	for rows.Next() {
		item, err := scanAnnouncementEvent(rows)
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

func (r *SQLRepository) UpdateEventStatus(ctx context.Context, id int64, status adminplusdomain.AnnouncementStatus) (*adminplusdomain.AnnouncementEvent, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	row := r.db.QueryRowContext(ctx, `
		UPDATE admin_plus_announcement_events
		SET status = $2,
			acknowledged_at = CASE WHEN $2 = 'acknowledged' THEN NOW() ELSE NULL END
		WHERE id = $1
		RETURNING id, supplier_id, source, type, title, description, currency,
			min_recharge_cents, bonus_percent, discount_percent, runtime_status,
			balance_cents, switch_eligible, recommendation, status,
			starts_at, ends_at, captured_at, created_at, acknowledged_at, raw_payload
	`, id, string(status))
	event, err := scanAnnouncementEvent(row)
	if err == sql.ErrNoRows {
		return nil, infraerrors.New(http.StatusNotFound, "ANNOUNCEMENT_EVENT_NOT_FOUND", "announcement event not found")
	}
	return event, err
}

type announcementScanner interface {
	Scan(dest ...any) error
}

func scanAnnouncementEvent(scanner announcementScanner) (*adminplusdomain.AnnouncementEvent, error) {
	var event adminplusdomain.AnnouncementEvent
	var eventType, runtimeStatus, recommendation, status string
	var bonusPercent, discountPercent sql.NullFloat64
	var startsAt, endsAt, acknowledgedAt sql.NullTime
	var rawPayload []byte
	err := scanner.Scan(
		&event.ID,
		&event.SupplierID,
		&event.Source,
		&eventType,
		&event.Title,
		&event.Description,
		&event.Currency,
		&event.MinRechargeCents,
		&bonusPercent,
		&discountPercent,
		&runtimeStatus,
		&event.BalanceCents,
		&event.SwitchEligible,
		&recommendation,
		&status,
		&startsAt,
		&endsAt,
		&event.CapturedAt,
		&event.CreatedAt,
		&acknowledgedAt,
		&rawPayload,
	)
	if err != nil {
		return nil, err
	}
	event.Type = adminplusdomain.AnnouncementType(eventType)
	event.RuntimeStatus = adminplusdomain.SupplierRuntimeStatus(runtimeStatus)
	event.Recommendation = adminplusdomain.AnnouncementRecommendation(recommendation)
	event.Status = adminplusdomain.AnnouncementStatus(status)
	if bonusPercent.Valid {
		v := bonusPercent.Float64
		event.BonusPercent = &v
	}
	if discountPercent.Valid {
		v := discountPercent.Float64
		event.DiscountPercent = &v
	}
	if startsAt.Valid {
		t := startsAt.Time
		event.StartsAt = &t
	}
	if endsAt.Valid {
		t := endsAt.Time
		event.EndsAt = &t
	}
	if acknowledgedAt.Valid {
		t := acknowledgedAt.Time
		event.AcknowledgedAt = &t
	}
	if len(rawPayload) > 0 {
		var payload map[string]any
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return nil, err
		}
		event.RawPayload = payload
	}
	return &event, nil
}

func marshalRawPayload(payload map[string]any) ([]byte, error) {
	if len(payload) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(payload)
}

func nullableFloat64(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func dbNotConfigured() error {
	return infraerrors.New(http.StatusInternalServerError, "ADMIN_PLUS_DB_NOT_CONFIGURED", "admin plus database is not configured")
}
