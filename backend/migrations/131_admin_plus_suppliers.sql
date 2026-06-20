CREATE TABLE IF NOT EXISTS admin_plus_suppliers (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    type TEXT NOT NULL,
    runtime_status TEXT NOT NULL DEFAULT 'monitor_only',
    health_status TEXT NOT NULL DEFAULT 'normal',
    dashboard_url TEXT NOT NULL DEFAULT '',
    api_base_url TEXT NOT NULL DEFAULT '',
    contact TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    admin_api_key_configured BOOLEAN NOT NULL DEFAULT FALSE,
    postgres_configured BOOLEAN NOT NULL DEFAULT FALSE,
    redis_configured BOOLEAN NOT NULL DEFAULT FALSE,
    browser_login_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    masked_admin_api_key TEXT NOT NULL DEFAULT '',
    balance_cents BIGINT NOT NULL DEFAULT 0,
    balance_currency TEXT NOT NULL DEFAULT 'USD',
    balance_updated_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_suppliers_kind_check CHECK (kind IN ('source_account', 'relay', 'browser_only', 'custom')),
    CONSTRAINT admin_plus_suppliers_type_check CHECK (type IN ('openai', 'anthropic', 'gemini', 'sub2api', 'new_api', 'browser_only', 'custom')),
    CONSTRAINT admin_plus_suppliers_runtime_status_check CHECK (runtime_status IN ('monitor_only', 'candidate', 'active', 'disabled')),
    CONSTRAINT admin_plus_suppliers_health_status_check CHECK (health_status IN ('normal', 'unavailable', 'credential_invalid', 'paused')),
    CONSTRAINT admin_plus_suppliers_candidate_balance_check CHECK (
        runtime_status NOT IN ('candidate', 'active') OR balance_cents > 0
    )
);

CREATE INDEX IF NOT EXISTS idx_admin_plus_suppliers_kind ON admin_plus_suppliers(kind);
CREATE INDEX IF NOT EXISTS idx_admin_plus_suppliers_type ON admin_plus_suppliers(type);
CREATE INDEX IF NOT EXISTS idx_admin_plus_suppliers_runtime_status ON admin_plus_suppliers(runtime_status);
CREATE INDEX IF NOT EXISTS idx_admin_plus_suppliers_health_status ON admin_plus_suppliers(health_status);
