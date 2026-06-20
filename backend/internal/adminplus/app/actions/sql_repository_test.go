package actions

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func newActionSQLMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	})
	return db, mock
}

func TestSQLRepositoryCreateActionRecommendation(t *testing.T) {
	db, mock := newActionSQLMock(t)
	repo := NewSQLRepository(db)
	createdAt := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	targetSupplierID := int64(9)

	mock.ExpectQuery(`INSERT INTO admin_plus_action_recommendations`).
		WithArgs(
			int64(7),
			targetSupplierID,
			"switch_supplier",
			"critical",
			"open",
			"switch_from_depleted_supplier",
			"Switch supplier",
			"depleted",
			"restore traffic",
			true,
			sqlmock.AnyArg(),
			createdAt,
		).
		WillReturnRows(newActionRows().AddRow(
			int64(21),
			int64(7),
			targetSupplierID,
			"switch_supplier",
			"critical",
			"open",
			"switch_from_depleted_supplier",
			"Switch supplier",
			"depleted",
			"restore traffic",
			true,
			pq.StringArray{"balance_depleted", "candidate_available"},
			createdAt,
		))

	got, err := repo.CreateRecommendation(context.Background(), &adminplusdomain.ActionRecommendation{
		SupplierID:       7,
		TargetSupplierID: &targetSupplierID,
		Type:             adminplusdomain.ActionTypeSwitchSupplier,
		Severity:         adminplusdomain.ActionSeverityCritical,
		Status:           adminplusdomain.ActionStatusOpen,
		ReasonCode:       "switch_from_depleted_supplier",
		Title:            "Switch supplier",
		Description:      "depleted",
		ExpectedImpact:   "restore traffic",
		RequiresApproval: true,
		Signals:          []string{"balance_depleted", "candidate_available"},
		CreatedAt:        createdAt,
	})

	require.NoError(t, err)
	require.Equal(t, int64(21), got.ID)
	require.NotNil(t, got.TargetSupplierID)
	require.Equal(t, adminplusdomain.ActionSeverityCritical, got.Severity)
	require.Equal(t, []string{"balance_depleted", "candidate_available"}, got.Signals)
}

func TestSQLRepositoryListActionRecommendationsFiltersWithParameterizedQuery(t *testing.T) {
	db, mock := newActionSQLMock(t)
	repo := NewSQLRepository(db)
	createdAt := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`FROM admin_plus_action_recommendations\s+WHERE 1=1 AND supplier_id = \$1 AND status = \$2 AND severity = \$3 AND type = \$4\s+ORDER BY created_at DESC, id DESC\s+LIMIT \$5`).
		WithArgs(int64(7), "open", "warning", "degrade_supplier", 50).
		WillReturnRows(newActionRows().AddRow(
			int64(21),
			int64(7),
			nil,
			"degrade_supplier",
			"warning",
			"open",
			"supplier_performance_degraded",
			"Degrade supplier",
			"slow",
			"reduce slow responses",
			true,
			pq.StringArray{"slow_first_token"},
			createdAt,
		))

	items, err := repo.ListRecommendations(context.Background(), ActionFilter{
		SupplierID: 7,
		Status:     adminplusdomain.ActionStatusOpen,
		Severity:   adminplusdomain.ActionSeverityWarning,
		Type:       adminplusdomain.ActionTypeDegradeSupplier,
		Limit:      50,
	})

	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, adminplusdomain.ActionTypeDegradeSupplier, items[0].Type)
	require.Equal(t, []string{"slow_first_token"}, items[0].Signals)
}

func TestSQLRepositoryUpdateActionRecommendationStatusNotFound(t *testing.T) {
	db, mock := newActionSQLMock(t)
	repo := NewSQLRepository(db)

	mock.ExpectQuery(`UPDATE admin_plus_action_recommendations`).
		WithArgs(int64(404), "acknowledged").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.UpdateRecommendationStatus(context.Background(), 404, adminplusdomain.ActionStatusAcknowledged)

	require.Error(t, err)
	require.Contains(t, err.Error(), "ACTION_RECOMMENDATION_NOT_FOUND")
}

func newActionRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id",
		"supplier_id",
		"target_supplier_id",
		"type",
		"severity",
		"status",
		"reason_code",
		"title",
		"description",
		"expected_impact",
		"requires_approval",
		"signals",
		"created_at",
	})
}
