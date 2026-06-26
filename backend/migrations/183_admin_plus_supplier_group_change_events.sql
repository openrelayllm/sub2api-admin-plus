CREATE TABLE IF NOT EXISTS admin_plus_supplier_group_change_events (
    id BIGSERIAL PRIMARY KEY,
    supplier_id BIGINT NOT NULL REFERENCES admin_plus_suppliers(id) ON DELETE CASCADE,
    supplier_group_id BIGINT NOT NULL REFERENCES admin_plus_supplier_groups(id) ON DELETE CASCADE,
    external_group_id TEXT NOT NULL,
    group_name TEXT NOT NULL,
    provider_family TEXT NOT NULL DEFAULT 'mixed',
    direction TEXT NOT NULL,
    old_effective_rate_multiplier DOUBLE PRECISION NULL,
    new_effective_rate_multiplier DOUBLE PRECISION NOT NULL,
    change_percent DOUBLE PRECISION NULL,
    low_rate BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_supplier_group_change_events_direction_check CHECK (direction IN ('new', 'increase', 'decrease')),
    CONSTRAINT admin_plus_supplier_group_change_events_rate_check CHECK (
        (old_effective_rate_multiplier IS NULL OR old_effective_rate_multiplier >= 0)
        AND new_effective_rate_multiplier >= 0
    )
);

CREATE INDEX IF NOT EXISTS idx_admin_plus_supplier_group_change_events_supplier_created
    ON admin_plus_supplier_group_change_events(supplier_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_admin_plus_supplier_group_change_events_group_created
    ON admin_plus_supplier_group_change_events(supplier_group_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_admin_plus_supplier_group_change_events_low_rate
    ON admin_plus_supplier_group_change_events(low_rate, created_at DESC, id DESC);
