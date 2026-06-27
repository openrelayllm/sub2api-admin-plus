CREATE TABLE IF NOT EXISTS admin_plus_purity_public_reports (
    id BIGSERIAL PRIMARY KEY,
    request_hash TEXT NOT NULL,
    provider TEXT NOT NULL,
    api_base_host TEXT NOT NULL,
    score INTEGER NOT NULL DEFAULT 0,
    verdict TEXT NOT NULL,
    checks_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    metrics_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    public_summary_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_plus_purity_public_reports_created
    ON admin_plus_purity_public_reports(created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_admin_plus_purity_public_reports_provider_host
    ON admin_plus_purity_public_reports(provider, api_base_host, created_at DESC);
