ALTER TABLE admin_plus_scheduler_steps
    ADD COLUMN IF NOT EXISTS attempts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_attempts INTEGER NOT NULL DEFAULT 3,
    ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS locked_by TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_admin_plus_scheduler_steps_claim
    ON admin_plus_scheduler_steps(status, next_attempt_at, id)
    WHERE status IN ('queued', 'retryable_failed', 'running');

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
