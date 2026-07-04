CREATE TABLE IF NOT EXISTS admin_plus_kanban_events (
    id BIGSERIAL PRIMARY KEY,
    event_type TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'info',
    status TEXT NOT NULL DEFAULT 'open',
    model TEXT NOT NULL DEFAULT '',
    source_type TEXT NOT NULL DEFAULT '',
    source_id BIGINT NULL,
    related_snapshot_type TEXT NOT NULL DEFAULT '',
    related_snapshot_id BIGINT NULL,
    title TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    recommendation TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_kanban_events_type_check CHECK (event_type IN ('market_price_drop', 'market_price_rise', 'market_price_anomaly', 'market_model_added', 'market_model_removed', 'market_promotion', 'cache_efficiency_risk', 'supply_quality_risk', 'acceptance_risk', 'unprofitable_model', 'pricing_recommendation')),
    CONSTRAINT admin_plus_kanban_events_severity_check CHECK (severity IN ('info', 'warning', 'critical')),
    CONSTRAINT admin_plus_kanban_events_status_check CHECK (status IN ('open', 'acknowledged', 'ignored')),
    CONSTRAINT admin_plus_kanban_events_snapshot_type_check CHECK (related_snapshot_type IN ('', 'market_price', 'cache_efficiency', 'supply_quality', 'acceptance_report', 'overview'))
);

CREATE INDEX IF NOT EXISTS idx_admin_plus_kanban_events_status_created
    ON admin_plus_kanban_events(status, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_admin_plus_kanban_events_model_created
    ON admin_plus_kanban_events(model, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_admin_plus_kanban_events_type_created
    ON admin_plus_kanban_events(event_type, created_at DESC, id DESC);
