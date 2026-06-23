CREATE TABLE IF NOT EXISTS admin_plus_scheduler_runs (
    id TEXT PRIMARY KEY,
    legacy_run_id TEXT NOT NULL DEFAULT '',
    trigger_type TEXT NOT NULL,
    task_type TEXT NOT NULL,
    status TEXT NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    supplier_count INTEGER NOT NULL DEFAULT 0,
    total_steps INTEGER NOT NULL DEFAULT 0,
    succeeded_steps INTEGER NOT NULL DEFAULT 0,
    failed_steps INTEGER NOT NULL DEFAULT 0,
    skipped_steps INTEGER NOT NULL DEFAULT 0,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    error_code TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    request_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    result_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_scheduler_runs_status_check CHECK (
        status IN ('queued', 'running', 'succeeded', 'partial_succeeded', 'retryable_failed', 'manual_required', 'dead', 'cancelled', 'skipped')
    )
);

CREATE INDEX IF NOT EXISTS idx_admin_plus_scheduler_runs_requested
    ON admin_plus_scheduler_runs(requested_at DESC);

CREATE INDEX IF NOT EXISTS idx_admin_plus_scheduler_runs_status
    ON admin_plus_scheduler_runs(status, requested_at DESC);

CREATE TABLE IF NOT EXISTS admin_plus_scheduler_plans (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    task_type TEXT NOT NULL,
    task_types TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    status TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT '',
    frequency_label TEXT NOT NULL DEFAULT '',
    interval_seconds BIGINT NOT NULL DEFAULT 0,
    window_minutes INTEGER NOT NULL DEFAULT 5,
    misfire_policy TEXT NOT NULL DEFAULT 'fire_once',
    concurrency_policy TEXT NOT NULL DEFAULT 'forbid',
    high_cost BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT NOT NULL DEFAULT '',
    last_run_at TIMESTAMPTZ NULL,
    next_run_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_scheduler_plans_status_check CHECK (
        status IN ('enabled', 'paused', 'disabled')
    )
);

CREATE INDEX IF NOT EXISTS idx_admin_plus_scheduler_plans_due
    ON admin_plus_scheduler_plans(status, next_run_at)
    WHERE status = 'enabled';

CREATE TABLE IF NOT EXISTS admin_plus_scheduler_steps (
    id BIGSERIAL PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES admin_plus_scheduler_runs(id) ON DELETE CASCADE,
    supplier_id BIGINT NOT NULL DEFAULT 0,
    supplier_name TEXT NOT NULL DEFAULT '',
    task_type TEXT NOT NULL,
    action TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    schedule_key TEXT NOT NULL DEFAULT '',
    extension_task_id BIGINT NOT NULL DEFAULT 0,
    result_count INTEGER NOT NULL DEFAULT 0,
    reason TEXT NOT NULL DEFAULT '',
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_by TEXT NOT NULL DEFAULT '',
    locked_until TIMESTAMPTZ NULL,
    request_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    result_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_scheduler_steps_status_check CHECK (
        status IN ('queued', 'running', 'succeeded', 'skipped', 'retryable_failed', 'manual_required', 'dead', 'cancelled')
    )
);

CREATE INDEX IF NOT EXISTS idx_admin_plus_scheduler_steps_run
    ON admin_plus_scheduler_steps(run_id, id);

CREATE INDEX IF NOT EXISTS idx_admin_plus_scheduler_steps_claim
    ON admin_plus_scheduler_steps(status, next_attempt_at, id)
    WHERE status IN ('queued', 'retryable_failed', 'running');

CREATE INDEX IF NOT EXISTS idx_admin_plus_scheduler_steps_supplier
    ON admin_plus_scheduler_steps(supplier_id, created_at DESC);

CREATE TABLE IF NOT EXISTS admin_plus_scheduler_attempts (
    id BIGSERIAL PRIMARY KEY,
    step_id BIGINT NOT NULL REFERENCES admin_plus_scheduler_steps(id) ON DELETE CASCADE,
    run_id TEXT NOT NULL REFERENCES admin_plus_scheduler_runs(id) ON DELETE CASCADE,
    supplier_id BIGINT NOT NULL DEFAULT 0,
    task_type TEXT NOT NULL,
    status TEXT NOT NULL,
    worker_id TEXT NOT NULL DEFAULT '',
    attempt_no INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    duration_ms BIGINT NOT NULL DEFAULT 0,
    error_code TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    request_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    response_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_plus_scheduler_attempts_step
    ON admin_plus_scheduler_attempts(step_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_admin_plus_scheduler_attempts_run
    ON admin_plus_scheduler_attempts(run_id, created_at DESC);

CREATE TABLE IF NOT EXISTS admin_plus_scheduler_actions (
    id TEXT PRIMARY KEY,
    supplier_id BIGINT NOT NULL DEFAULT 0,
    supplier_name TEXT NOT NULL DEFAULT '',
    severity TEXT NOT NULL,
    status TEXT NOT NULL,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    recommended_operation TEXT NOT NULL DEFAULT '',
    evidence_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    related_run_id TEXT NOT NULL DEFAULT '',
    related_step_id BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ NULL,
    CONSTRAINT admin_plus_scheduler_actions_status_check CHECK (
        status IN ('open', 'investigating', 'ready_to_execute', 'executing', 'verifying', 'resolved', 'ignored')
    ),
    CONSTRAINT admin_plus_scheduler_actions_severity_check CHECK (
        severity IN ('info', 'warning', 'critical')
    )
);

CREATE INDEX IF NOT EXISTS idx_admin_plus_scheduler_actions_status
    ON admin_plus_scheduler_actions(status, created_at DESC);

CREATE TABLE IF NOT EXISTS admin_plus_scheduler_settings (
    key TEXT PRIMARY KEY,
    value JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
