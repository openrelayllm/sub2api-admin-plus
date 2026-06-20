CREATE TABLE IF NOT EXISTS admin_plus_rate_snapshots (
    id BIGSERIAL PRIMARY KEY,
    supplier_id BIGINT NOT NULL REFERENCES admin_plus_suppliers(id) ON DELETE CASCADE,
    source TEXT NOT NULL DEFAULT 'manual',
    model TEXT NOT NULL,
    billing_mode TEXT NOT NULL,
    price_item TEXT NOT NULL,
    unit TEXT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'USD',
    price_micros BIGINT NOT NULL,
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_rate_snapshots_price_non_negative CHECK (price_micros >= 0)
);

CREATE INDEX IF NOT EXISTS idx_admin_plus_rate_snapshots_supplier_captured
    ON admin_plus_rate_snapshots(supplier_id, captured_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_admin_plus_rate_snapshots_lookup
    ON admin_plus_rate_snapshots(supplier_id, model, billing_mode, price_item, unit, currency, captured_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS admin_plus_rate_change_events (
    id BIGSERIAL PRIMARY KEY,
    supplier_id BIGINT NOT NULL REFERENCES admin_plus_suppliers(id) ON DELETE CASCADE,
    snapshot_id BIGINT NOT NULL REFERENCES admin_plus_rate_snapshots(id) ON DELETE CASCADE,
    model TEXT NOT NULL,
    billing_mode TEXT NOT NULL,
    price_item TEXT NOT NULL,
    unit TEXT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'USD',
    old_price_micros BIGINT NULL,
    new_price_micros BIGINT NOT NULL,
    direction TEXT NOT NULL,
    change_percent DOUBLE PRECISION NULL,
    threshold_percent DOUBLE PRECISION NOT NULL DEFAULT 1,
    threshold_exceeded BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL DEFAULT 'open',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at TIMESTAMPTZ NULL,
    CONSTRAINT admin_plus_rate_change_events_direction_check CHECK (direction IN ('new', 'increase', 'decrease')),
    CONSTRAINT admin_plus_rate_change_events_status_check CHECK (status IN ('open', 'acknowledged', 'ignored'))
);

CREATE INDEX IF NOT EXISTS idx_admin_plus_rate_change_events_supplier_status
    ON admin_plus_rate_change_events(supplier_id, status, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_admin_plus_rate_change_events_snapshot
    ON admin_plus_rate_change_events(snapshot_id);
