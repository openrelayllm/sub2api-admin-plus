package promotions

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/stretchr/testify/require"
)

func newPromotionSQLMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	})
	return db, mock
}

func TestSQLRepositoryCreatePromotionEvent(t *testing.T) {
	db, mock := newPromotionSQLMock(t)
	repo := NewSQLRepository(db)
	capturedAt := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	createdAt := capturedAt.Add(time.Second)
	bonus := 20.0

	mock.ExpectQuery(`INSERT INTO admin_plus_promotion_events`).
		WithArgs(
			int64(7),
			"chrome",
			"recharge_bonus",
			"Recharge bonus",
			"20 percent bonus",
			"USD",
			int64(10000),
			bonus,
			nil,
			"monitor_only",
			int64(0),
			false,
			"recharge_to_unlock",
			"open",
			nil,
			nil,
			capturedAt,
			sqlmock.AnyArg(),
		).
		WillReturnRows(newPromotionRows().AddRow(
			int64(21),
			int64(7),
			"chrome",
			"recharge_bonus",
			"Recharge bonus",
			"20 percent bonus",
			"USD",
			int64(10000),
			bonus,
			nil,
			"monitor_only",
			int64(0),
			false,
			"recharge_to_unlock",
			"open",
			nil,
			nil,
			capturedAt,
			createdAt,
			nil,
			[]byte(`{"page":"promo"}`),
		))

	got, err := repo.CreateEvent(context.Background(), &adminplusdomain.PromotionEvent{
		SupplierID:       7,
		Source:           "chrome",
		Type:             adminplusdomain.PromotionTypeRechargeBonus,
		Title:            "Recharge bonus",
		Description:      "20 percent bonus",
		Currency:         "USD",
		MinRechargeCents: 10000,
		BonusPercent:     &bonus,
		RuntimeStatus:    adminplusdomain.SupplierRuntimeStatusMonitorOnly,
		BalanceCents:     0,
		SwitchEligible:   false,
		Recommendation:   adminplusdomain.PromotionRecommendationRechargeToUnlock,
		Status:           adminplusdomain.PromotionStatusOpen,
		CapturedAt:       capturedAt,
		RawPayload:       map[string]any{"page": "promo"},
	})

	require.NoError(t, err)
	require.Equal(t, int64(21), got.ID)
	require.NotNil(t, got.BonusPercent)
	require.Equal(t, "promo", got.RawPayload["page"])
}

func TestSQLRepositoryListPromotionEventsFiltersWithParameterizedQuery(t *testing.T) {
	db, mock := newPromotionSQLMock(t)
	repo := NewSQLRepository(db)
	capturedAt := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`FROM admin_plus_promotion_events\s+WHERE 1=1 AND supplier_id = \$1 AND status = \$2 AND recommendation = \$3\s+ORDER BY created_at DESC, id DESC\s+LIMIT \$4`).
		WithArgs(int64(7), "open", "recharge_to_unlock", 50).
		WillReturnRows(newPromotionRows().AddRow(
			int64(21),
			int64(7),
			"chrome",
			"recharge_bonus",
			"Recharge bonus",
			"",
			"USD",
			int64(10000),
			nil,
			nil,
			"monitor_only",
			int64(0),
			false,
			"recharge_to_unlock",
			"open",
			nil,
			nil,
			capturedAt,
			capturedAt,
			nil,
			[]byte(`{}`),
		))

	items, err := repo.ListEvents(context.Background(), EventFilter{
		SupplierID:     7,
		Status:         adminplusdomain.PromotionStatusOpen,
		Recommendation: adminplusdomain.PromotionRecommendationRechargeToUnlock,
		Limit:          50,
	})

	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, adminplusdomain.PromotionRecommendationRechargeToUnlock, items[0].Recommendation)
}

func TestSQLRepositoryUpdatePromotionEventStatusNotFound(t *testing.T) {
	db, mock := newPromotionSQLMock(t)
	repo := NewSQLRepository(db)

	mock.ExpectQuery(`UPDATE admin_plus_promotion_events`).
		WithArgs(int64(404), "acknowledged").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.UpdateEventStatus(context.Background(), 404, adminplusdomain.PromotionStatusAcknowledged)

	require.Error(t, err)
	require.Contains(t, err.Error(), "PROMOTION_EVENT_NOT_FOUND")
}

func newPromotionRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id",
		"supplier_id",
		"source",
		"type",
		"title",
		"description",
		"currency",
		"min_recharge_cents",
		"bonus_percent",
		"discount_percent",
		"runtime_status",
		"balance_cents",
		"switch_eligible",
		"recommendation",
		"status",
		"starts_at",
		"ends_at",
		"captured_at",
		"created_at",
		"acknowledged_at",
		"raw_payload",
	})
}
