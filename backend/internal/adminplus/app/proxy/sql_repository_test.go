package proxy

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestSQLRepositoryCenterStatusScansMetrics(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows([]string{
		"subscriptions_total",
		"subscriptions_active",
		"nodes_total",
		"healthy_nodes",
		"policies_total",
		"targets_total",
		"slots_total",
		"slots_assigned",
		"assignments_active",
		"recent_errors",
		"node_switches_24h",
		"node_failures_24h",
		"policy_denials_24h",
		"egress_verify_failures_24h",
		"completed_assignments_24h",
		"avg_assignment_seconds_24h",
	}).AddRow(1, 1, 5, 3, 2, 4, 6, 2, 1, 7, 8, 9, 10, 11, 12, 125)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	status, err := NewSQLRepository(db).CenterStatus(context.Background())
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.Equal(t, 1, status.SubscriptionsTotal)
	require.Equal(t, 1, status.SubscriptionsActive)
	require.Equal(t, 5, status.NodesTotal)
	require.Equal(t, 3, status.HealthyNodes)
	require.Equal(t, 2, status.PoliciesTotal)
	require.Equal(t, 4, status.TargetsTotal)
	require.Equal(t, 6, status.SlotsTotal)
	require.Equal(t, 2, status.SlotsAssigned)
	require.Equal(t, 1, status.AssignmentsActive)
	require.Equal(t, 7, status.RecentErrors)
	require.Equal(t, 8, status.NodeSwitches24h)
	require.Equal(t, 9, status.NodeFailures24h)
	require.Equal(t, 10, status.PolicyDenials24h)
	require.Equal(t, 11, status.EgressVerifyFailures24h)
	require.Equal(t, 12, status.CompletedAssignments24h)
	require.Equal(t, 125, status.AvgAssignmentSeconds24h)
}
