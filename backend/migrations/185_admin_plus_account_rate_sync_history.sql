CREATE TABLE IF NOT EXISTS admin_plus_account_rate_sync_history (
    id BIGSERIAL PRIMARY KEY,
    local_sub2api_account_id BIGINT NOT NULL DEFAULT 0,
    local_account_name TEXT NOT NULL DEFAULT '',
    local_account_platform TEXT NOT NULL DEFAULT '',
    key_fingerprint TEXT NOT NULL DEFAULT '',
    key_last4 TEXT NOT NULL DEFAULT '',
    supplier_id BIGINT NOT NULL DEFAULT 0,
    supplier_name TEXT NOT NULL DEFAULT '',
    supplier_type TEXT NOT NULL DEFAULT '',
    supplier_group_id BIGINT NOT NULL DEFAULT 0,
    supplier_group_name TEXT NOT NULL DEFAULT '',
    supplier_key_id BIGINT NOT NULL DEFAULT 0,
    match_source TEXT NOT NULL DEFAULT '',
    effective_rate_multiplier DOUBLE PRECISION NOT NULL DEFAULT 0,
    target_account_name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'not_found',
    error_code TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    renamed BOOLEAN NOT NULL DEFAULT FALSE,
    old_account_name TEXT NOT NULL DEFAULT '',
    new_account_name TEXT NOT NULL DEFAULT '',
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_account_rate_sync_status_check CHECK (
        status IN ('matched', 'renamed', 'not_found', 'ambiguous', 'failed')
    ),
    CONSTRAINT admin_plus_account_rate_sync_rate_check CHECK (effective_rate_multiplier >= 0)
);

CREATE INDEX IF NOT EXISTS idx_admin_plus_account_rate_sync_history_account
    ON admin_plus_account_rate_sync_history(local_sub2api_account_id, synced_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_admin_plus_account_rate_sync_history_synced
    ON admin_plus_account_rate_sync_history(synced_at DESC, id DESC);
