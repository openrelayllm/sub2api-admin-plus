package scheduler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/lib/pq"
)

func (r *SQLRepository) UpdatePlanConfig(ctx context.Context, planID string, config adminplusdomain.SchedulerPlanConfig) (*adminplusdomain.SchedulerPlan, error) {
	if r == nil || r.db == nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "ADMIN_PLUS_SCHEDULER_DB_NOT_CONFIGURED", "admin plus scheduler database is not configured")
	}
	row := r.db.QueryRowContext(ctx, `
		UPDATE admin_plus_scheduler_plans
		SET status = $2,
			scope = $3,
			frequency_label = $4,
			interval_seconds = $5,
			window_minutes = $6,
			misfire_policy = $7,
			concurrency_policy = $8,
			next_run_at = CASE
				WHEN $2 = 'enabled' AND $5 > 0 THEN NOW() + make_interval(secs => $5::int)
				ELSE NULL
			END,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, task_type, task_types, status, scope, frequency_label, interval_seconds,
			window_minutes, misfire_policy, concurrency_policy, high_cost, description, last_run_at, next_run_at
	`, planID, config.Status, config.Scope, frequencyLabel(time.Duration(config.IntervalSeconds)*time.Second), config.IntervalSeconds, config.WindowMinutes, config.MisfirePolicy, config.ConcurrencyPolicy)
	return scanPlan(row)
}

func (r *SQLRepository) PlanStats(ctx context.Context, plans []adminplusdomain.SchedulerPlan) (map[string]adminplusdomain.SchedulerPlanStats, error) {
	if r == nil || r.db == nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "ADMIN_PLUS_SCHEDULER_DB_NOT_CONFIGURED", "admin plus scheduler database is not configured")
	}
	out := make(map[string]adminplusdomain.SchedulerPlanStats, len(plans))
	for _, plan := range plans {
		taskTypes := plan.TaskTypes
		if len(taskTypes) == 0 {
			taskTypes = []string{plan.TaskType}
		}
		var lastSuccessAt, lastIssueAt sql.NullTime
		var issueCount int
		if err := r.db.QueryRowContext(ctx, `
			WITH scoped AS (
				SELECT status, finished_at, started_at, next_attempt_at
				FROM admin_plus_scheduler_steps
				WHERE task_type = ANY($1)
			),
			success AS (
				SELECT MAX(finished_at) AS last_success_at
				FROM scoped
				WHERE status = 'succeeded'
			)
			SELECT success.last_success_at,
				COUNT(*) FILTER (
					WHERE scoped.status IN ('retryable_failed', 'manual_required', 'dead')
					  AND (success.last_success_at IS NULL OR COALESCE(scoped.finished_at, scoped.started_at, scoped.next_attempt_at) > success.last_success_at)
				)::int AS issue_count,
				MAX(COALESCE(scoped.finished_at, scoped.started_at, scoped.next_attempt_at)) FILTER (
					WHERE scoped.status IN ('retryable_failed', 'manual_required', 'dead')
					  AND (success.last_success_at IS NULL OR COALESCE(scoped.finished_at, scoped.started_at, scoped.next_attempt_at) > success.last_success_at)
				) AS last_issue_at
			FROM success
			LEFT JOIN scoped ON TRUE
			GROUP BY success.last_success_at
		`, pq.Array(taskTypes)).Scan(&lastSuccessAt, &issueCount, &lastIssueAt); err != nil {
			return nil, err
		}
		stat := adminplusdomain.SchedulerPlanStats{
			PlanID:        plan.ID,
			LastSuccessAt: timePtr(lastSuccessAt),
			IssueCount:    issueCount,
			LastIssueAt:   timePtr(lastIssueAt),
		}
		if issueCount > 0 {
			var status, reason string
			err := r.db.QueryRowContext(ctx, `
				SELECT status, reason
				FROM admin_plus_scheduler_steps
				WHERE task_type = ANY($1)
				  AND status IN ('retryable_failed', 'manual_required', 'dead')
				  AND ($2::timestamptz IS NULL OR COALESCE(finished_at, started_at, next_attempt_at) > $2)
				ORDER BY COALESCE(finished_at, started_at, next_attempt_at) DESC, id DESC
				LIMIT 1
			`, pq.Array(taskTypes), nullableTime(stat.LastSuccessAt)).Scan(&status, &reason)
			if err != nil && err != sql.ErrNoRows {
				return nil, err
			}
			if err == nil {
				stat.LastIssue = firstNonEmpty(reasonCodeFromText(reason), reason, status)
			}
		}
		out[plan.ID] = stat
	}
	return out, nil
}
