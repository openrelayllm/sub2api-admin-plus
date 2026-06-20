package sub2api

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func newSub2APISQLMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	})
	return db, mock
}

func TestSQLRepositoryListLocalUsageLinesReadsUsageLogs(t *testing.T) {
	db, mock := newSub2APISQLMock(t)
	repo := NewSQLRepository(ReadDB{DB: db})
	from := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)

	mock.ExpectQuery(`FROM usage_logs ul\s+LEFT JOIN accounts a ON a\.id = ul\.account_id\s+WHERE ul\.created_at >= \$1 AND ul\.created_at < \$2 AND ul\.account_id = \$3.*LIMIT \$4`).
		WithArgs(from, to, int64(7), 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"account_id",
			"name",
			"platform",
			"request_id",
			"model",
			"input_tokens",
			"output_tokens",
			"revenue_cents",
			"created_at",
		}).AddRow(
			int64(99),
			int64(7),
			"OpenAI Production",
			"openai",
			"req-1",
			"gpt-4o-mini",
			int64(1000),
			int64(500),
			int64(123),
			from.Add(time.Hour),
		))

	items, err := repo.ListLocalUsageLines(context.Background(), UsageFilter{
		AccountID: 7,
		From:      from,
		To:        to,
		Limit:     10,
	})

	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(99), items[0].ID)
	require.Equal(t, "OpenAI Production", items[0].AccountName)
	require.Equal(t, int64(123), items[0].RevenueCents)
	require.Equal(t, "USD", items[0].Currency)
}

func TestSQLRepositoryListLocalUsageSummariesGroupsByAccountAndModel(t *testing.T) {
	db, mock := newSub2APISQLMock(t)
	repo := NewSQLRepository(ReadDB{DB: db})
	from := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)

	mock.ExpectQuery(`GROUP BY ul\.account_id, a\.name, a\.platform, COALESCE`).
		WithArgs(from, to, "gpt-4o-mini", 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"account_id",
			"name",
			"platform",
			"model",
			"request_count",
			"input_tokens",
			"output_tokens",
			"revenue_cents",
			"original_cost_cents",
			"avg_first_token_ms",
			"avg_total_latency_ms",
			"window_start",
			"window_end",
			"last_request_created_at",
		}).AddRow(
			int64(7),
			"OpenAI Production",
			"openai",
			"gpt-4o-mini",
			int64(3),
			int64(3000),
			int64(1500),
			int64(456),
			int64(300),
			int64(800),
			int64(2400),
			from,
			to.Add(-time.Hour),
			to.Add(-time.Hour),
		))

	items, err := repo.ListLocalUsageSummaries(context.Background(), UsageFilter{
		Model: "gpt-4o-mini",
		From:  from,
		To:    to,
		Limit: 20,
	})

	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(3), items[0].RequestCount)
	require.Equal(t, int64(456), items[0].RevenueCents)
	require.Equal(t, int64(800), items[0].AvgFirstTokenMs)
}
