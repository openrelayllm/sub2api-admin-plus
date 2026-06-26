package suppliergroups

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/stretchr/testify/require"
)

func newSupplierGroupSQLMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	})
	return db, mock
}

func TestSQLRepositoryCreateChangeEvents(t *testing.T) {
	db, mock := newSupplierGroupSQLMock(t)
	repo := NewSQLRepository(db)
	createdAt := time.Date(2026, 6, 26, 10, 0, 0, 0, time.UTC)
	oldRate := 0.13
	changePercent := 100.0

	mock.ExpectQuery(`INSERT INTO admin_plus_supplier_group_change_events`).
		WithArgs(
			int64(7),
			int64(11),
			"gpt-low",
			"gpt-low",
			"openai",
			"increase",
			oldRate,
			0.26,
			changePercent,
			true,
			createdAt,
		).
		WillReturnRows(newSupplierGroupChangeEventRows().AddRow(
			int64(21),
			int64(7),
			int64(11),
			"gpt-low",
			"gpt-low",
			"openai",
			"increase",
			oldRate,
			0.26,
			changePercent,
			true,
			createdAt,
		))

	items, err := repo.CreateChangeEvents(context.Background(), []*adminplusdomain.SupplierGroupChangeEvent{
		{
			SupplierID:                 7,
			SupplierGroupID:            11,
			ExternalGroupID:            "gpt-low",
			GroupName:                  "gpt-low",
			ProviderFamily:             "openai",
			Direction:                  adminplusdomain.SupplierGroupChangeDirectionIncrease,
			OldEffectiveRateMultiplier: &oldRate,
			NewEffectiveRateMultiplier: 0.26,
			ChangePercent:              &changePercent,
			LowRate:                    true,
			CreatedAt:                  createdAt,
		},
	})

	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(21), items[0].ID)
	require.Equal(t, adminplusdomain.SupplierGroupChangeDirectionIncrease, items[0].Direction)
}

func TestSQLRepositoryListChangeEvents(t *testing.T) {
	db, mock := newSupplierGroupSQLMock(t)
	repo := NewSQLRepository(db)
	createdAt := time.Date(2026, 6, 26, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`FROM admin_plus_supplier_group_change_events\s+WHERE supplier_id = \$1 AND direction = \$2 AND low_rate = \$3\s+ORDER BY created_at DESC, id DESC\s+LIMIT \$4`).
		WithArgs(int64(7), "new", true, 20).
		WillReturnRows(newSupplierGroupChangeEventRows().AddRow(
			int64(21),
			int64(7),
			int64(11),
			"gpt-low",
			"gpt-low",
			"openai",
			"new",
			nil,
			0.05,
			nil,
			true,
			createdAt,
		))

	lowRate := true
	items, err := repo.ListChangeEvents(context.Background(), EventFilter{
		SupplierID: 7,
		Direction:  adminplusdomain.SupplierGroupChangeDirectionNew,
		LowRate:    &lowRate,
		Limit:      20,
	})

	require.NoError(t, err)
	require.Len(t, items, 1)
	require.True(t, items[0].LowRate)
	require.Nil(t, items[0].OldEffectiveRateMultiplier)
}

func newSupplierGroupChangeEventRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id",
		"supplier_id",
		"supplier_group_id",
		"external_group_id",
		"group_name",
		"provider_family",
		"direction",
		"old_effective_rate_multiplier",
		"new_effective_rate_multiplier",
		"change_percent",
		"low_rate",
		"created_at",
	})
}
