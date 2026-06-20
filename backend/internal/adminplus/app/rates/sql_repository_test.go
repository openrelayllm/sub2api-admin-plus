package rates

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/stretchr/testify/require"
)

func newRateSQLMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	})
	return db, mock
}

func TestSQLRepositoryCreateSnapshot(t *testing.T) {
	db, mock := newRateSQLMock(t)
	repo := NewSQLRepository(db)
	capturedAt := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	createdAt := capturedAt.Add(time.Second)

	mock.ExpectQuery(`INSERT INTO admin_plus_rate_snapshots`).
		WithArgs(
			int64(7),
			"manual",
			"gpt-4o-mini",
			"token",
			"input",
			"1m_tokens",
			"USD",
			int64(150000),
			sqlmock.AnyArg(),
			capturedAt,
		).
		WillReturnRows(newRateSnapshotRows().AddRow(
			int64(11),
			int64(7),
			"manual",
			"gpt-4o-mini",
			"token",
			"input",
			"1m_tokens",
			"USD",
			int64(150000),
			[]byte(`{"source":"manual"}`),
			capturedAt,
			createdAt,
		))

	got, err := repo.CreateSnapshot(context.Background(), &adminplusdomain.RateSnapshot{
		SupplierID:  7,
		Source:      "manual",
		Model:       "gpt-4o-mini",
		BillingMode: "token",
		PriceItem:   "input",
		Unit:        "1m_tokens",
		Currency:    "USD",
		PriceMicros: 150000,
		RawPayload:  map[string]any{"source": "manual"},
		CapturedAt:  capturedAt,
	})

	require.NoError(t, err)
	require.Equal(t, int64(11), got.ID)
	require.Equal(t, "manual", got.RawPayload["source"])
}

func TestSQLRepositoryFindLatestComparableSnapshotNoRows(t *testing.T) {
	db, mock := newRateSQLMock(t)
	repo := NewSQLRepository(db)
	capturedAt := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`FROM admin_plus_rate_snapshots\s+WHERE supplier_id = \$1`).
		WithArgs(
			int64(7),
			"gpt-4o-mini",
			"token",
			"input",
			"1m_tokens",
			"USD",
			capturedAt,
		).
		WillReturnError(sql.ErrNoRows)

	got, err := repo.FindLatestComparableSnapshot(context.Background(), &adminplusdomain.RateSnapshot{
		SupplierID:  7,
		Model:       "gpt-4o-mini",
		BillingMode: "token",
		PriceItem:   "input",
		Unit:        "1m_tokens",
		Currency:    "USD",
		CapturedAt:  capturedAt,
	})

	require.NoError(t, err)
	require.Nil(t, got)
}

func TestSQLRepositoryCreateChangeEvent(t *testing.T) {
	db, mock := newRateSQLMock(t)
	repo := NewSQLRepository(db)
	createdAt := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	oldPrice := int64(100000)
	changePercent := 25.0

	mock.ExpectQuery(`INSERT INTO admin_plus_rate_change_events`).
		WithArgs(
			int64(7),
			int64(11),
			"gpt-4o-mini",
			"token",
			"input",
			"1m_tokens",
			"USD",
			oldPrice,
			int64(125000),
			"increase",
			changePercent,
			10.0,
			true,
			"open",
		).
		WillReturnRows(newRateChangeEventRows().AddRow(
			int64(21),
			int64(7),
			int64(11),
			"gpt-4o-mini",
			"token",
			"input",
			"1m_tokens",
			"USD",
			oldPrice,
			int64(125000),
			"increase",
			changePercent,
			10.0,
			true,
			"open",
			createdAt,
			nil,
		))

	got, err := repo.CreateChangeEvent(context.Background(), &adminplusdomain.RateChangeEvent{
		SupplierID:        7,
		SnapshotID:        11,
		Model:             "gpt-4o-mini",
		BillingMode:       "token",
		PriceItem:         "input",
		Unit:              "1m_tokens",
		Currency:          "USD",
		OldPriceMicros:    &oldPrice,
		NewPriceMicros:    125000,
		Direction:         adminplusdomain.RateChangeDirectionIncrease,
		ChangePercent:     &changePercent,
		ThresholdPercent:  10,
		ThresholdExceeded: true,
		Status:            adminplusdomain.RateChangeStatusOpen,
	})

	require.NoError(t, err)
	require.Equal(t, int64(21), got.ID)
	require.NotNil(t, got.ChangePercent)
	require.Equal(t, adminplusdomain.RateChangeDirectionIncrease, got.Direction)
}

func TestSQLRepositoryListSnapshotsFiltersWithParameterizedQuery(t *testing.T) {
	db, mock := newRateSQLMock(t)
	repo := NewSQLRepository(db)
	capturedAt := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`FROM admin_plus_rate_snapshots\s+WHERE 1=1 AND supplier_id = \$1 AND model = \$2\s+ORDER BY captured_at DESC, id DESC\s+LIMIT \$3`).
		WithArgs(int64(7), "gpt-4o-mini", 50).
		WillReturnRows(newRateSnapshotRows().AddRow(
			int64(11),
			int64(7),
			"manual",
			"gpt-4o-mini",
			"token",
			"input",
			"1m_tokens",
			"USD",
			int64(150000),
			[]byte(`{}`),
			capturedAt,
			capturedAt,
		))

	items, err := repo.ListSnapshots(context.Background(), SnapshotFilter{
		SupplierID: 7,
		Model:      "gpt-4o-mini",
		Limit:      50,
	})

	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(150000), items[0].PriceMicros)
}

func TestSQLRepositoryUpdateChangeEventStatusNotFound(t *testing.T) {
	db, mock := newRateSQLMock(t)
	repo := NewSQLRepository(db)

	mock.ExpectQuery(`UPDATE admin_plus_rate_change_events`).
		WithArgs(int64(404), "acknowledged").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.UpdateChangeEventStatus(context.Background(), 404, adminplusdomain.RateChangeStatusAcknowledged)

	require.Error(t, err)
	require.Contains(t, err.Error(), "RATE_EVENT_NOT_FOUND")
}

func newRateSnapshotRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id",
		"supplier_id",
		"source",
		"model",
		"billing_mode",
		"price_item",
		"unit",
		"currency",
		"price_micros",
		"raw_payload",
		"captured_at",
		"created_at",
	})
}

func newRateChangeEventRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id",
		"supplier_id",
		"snapshot_id",
		"model",
		"billing_mode",
		"price_item",
		"unit",
		"currency",
		"old_price_micros",
		"new_price_micros",
		"direction",
		"change_percent",
		"threshold_percent",
		"threshold_exceeded",
		"status",
		"created_at",
		"acknowledged_at",
	})
}
