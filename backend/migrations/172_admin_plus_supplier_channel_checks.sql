CREATE TABLE IF NOT EXISTS admin_plus_supplier_channel_check_snapshots (
    id BIGSERIAL PRIMARY KEY,
    supplier_id BIGINT NOT NULL REFERENCES admin_plus_suppliers(id) ON DELETE CASCADE,
    supplier_group_id BIGINT NOT NULL REFERENCES admin_plus_supplier_groups(id) ON DELETE CASCADE,
    supplier_key_id BIGINT NOT NULL DEFAULT 0,
    supplier_account_id BIGINT NOT NULL DEFAULT 0,
    local_sub2api_account_id BIGINT NOT NULL DEFAULT 0,
    external_group_id TEXT NOT NULL DEFAULT '',
    group_name TEXT NOT NULL DEFAULT '',
    provider_family TEXT NOT NULL DEFAULT '',
    channel_monitor_id BIGINT NOT NULL DEFAULT 0,
    channel_name TEXT NOT NULL DEFAULT '',
    channel_provider TEXT NOT NULL DEFAULT '',
    primary_model TEXT NOT NULL DEFAULT '',
    remote_status TEXT NOT NULL DEFAULT 'unknown',
    probe_model TEXT NOT NULL DEFAULT '',
    probe_status TEXT NOT NULL DEFAULT 'untested',
    recommended BOOLEAN NOT NULL DEFAULT FALSE,
    effective_rate_multiplier DOUBLE PRECISION NOT NULL DEFAULT 0,
    first_token_ms BIGINT NOT NULL DEFAULT 0,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    status_code INTEGER NOT NULL DEFAULT 0,
    error_class TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    local_account_schedulable BOOLEAN NOT NULL DEFAULT FALSE,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_supplier_channel_check_latency_check CHECK (first_token_ms >= 0 AND duration_ms >= 0),
    CONSTRAINT admin_plus_supplier_channel_check_rate_check CHECK (effective_rate_multiplier >= 0)
);

CREATE INDEX IF NOT EXISTS idx_admin_plus_supplier_channel_checks_supplier_group
    ON admin_plus_supplier_channel_check_snapshots(supplier_id, supplier_group_id, captured_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_admin_plus_supplier_channel_checks_supplier_recommended
    ON admin_plus_supplier_channel_check_snapshots(supplier_id, recommended, effective_rate_multiplier, captured_at DESC, id DESC);

ALTER TABLE supplier_provision_jobs
    DROP CONSTRAINT IF EXISTS supplier_provision_jobs_type_check;

ALTER TABLE supplier_provision_jobs
    ADD CONSTRAINT supplier_provision_jobs_type_check CHECK (
        job_type IN ('sync_groups', 'provision_group_key', 'provision_all_group_keys', 'repair_binding', 'sync_supplier_costs', 'check_supplier_channels')
    );

ALTER TABLE supplier_provision_steps
    DROP CONSTRAINT IF EXISTS supplier_provision_steps_type_check;

ALTER TABLE supplier_provision_steps
    ADD CONSTRAINT supplier_provision_steps_type_check CHECK (
        step_type IN (
            'ensure_supplier_session',
            'sync_supplier_group',
            'ensure_third_party_key',
            'ensure_sub2api_group',
            'ensure_sub2api_account',
            'upsert_admin_plus_binding',
            'enqueue_initial_collection',
            'provision_all_group_keys',
            'repair_binding',
            'sync_supplier_costs',
            'check_supplier_channels'
        )
    );
