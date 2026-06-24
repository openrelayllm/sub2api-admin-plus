CREATE TABLE IF NOT EXISTS admin_plus_site_discovery_settings (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE,
    registration_email TEXT NOT NULL DEFAULT '',
    registration_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    low_rate_threshold DOUBLE PRECISION NOT NULL DEFAULT 0.8,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_site_discovery_settings_singleton CHECK (id),
    CONSTRAINT admin_plus_site_discovery_low_rate_threshold_check CHECK (low_rate_threshold > 0)
);

INSERT INTO admin_plus_site_discovery_settings (id)
VALUES (TRUE)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS admin_plus_site_discovery_runs (
    id BIGSERIAL PRIMARY KEY,
    source_url TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'running',
    total INTEGER NOT NULL DEFAULT 0,
    supported_total INTEGER NOT NULL DEFAULT 0,
    imported_total INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_site_discovery_runs_status_check CHECK (status IN ('running', 'succeeded', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_admin_plus_site_discovery_runs_created
    ON admin_plus_site_discovery_runs(created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS admin_plus_site_discoveries (
    id BIGSERIAL PRIMARY KEY,
    run_id BIGINT NOT NULL REFERENCES admin_plus_site_discovery_runs(id) ON DELETE CASCADE,
    source_url TEXT NOT NULL,
    source_site_id TEXT NOT NULL DEFAULT '',
    source_section TEXT NOT NULL DEFAULT '',
    source_category TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    register_url TEXT NOT NULL,
    dashboard_url TEXT NOT NULL DEFAULT '',
    api_base_url TEXT NOT NULL DEFAULT '',
    host TEXT NOT NULL DEFAULT '',
    domain_hint TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    provider_type TEXT NOT NULL DEFAULT '',
    classification_status TEXT NOT NULL DEFAULT 'unknown',
    classification_confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    classification_evidence JSONB NOT NULL DEFAULT '[]'::jsonb,
    monitor_status TEXT NOT NULL DEFAULT '',
    monitor_available BOOLEAN NULL,
    monitor_uptime_percent DOUBLE PRECISION NULL,
    monitor_avg_response_ms INTEGER NULL,
    monitor_latest_response_ms INTEGER NULL,
    import_status TEXT NOT NULL DEFAULT 'new',
    process_status TEXT NOT NULL DEFAULT 'unprocessed',
    catalog_site_id BIGINT NULL,
    supplier_id BIGINT NULL REFERENCES admin_plus_suppliers(id) ON DELETE SET NULL,
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_site_discoveries_provider_type_check CHECK (provider_type IN ('', 'new_api', 'sub2api')),
    CONSTRAINT admin_plus_site_discoveries_classification_check CHECK (classification_status IN ('supported', 'unknown', 'unsupported')),
    CONSTRAINT admin_plus_site_discoveries_import_status_check CHECK (import_status IN ('new', 'imported', 'skipped')),
    CONSTRAINT admin_plus_site_discoveries_process_status_check CHECK (process_status IN ('unprocessed', 'added_to_catalog', 'registered', 'ignored')),
    CONSTRAINT admin_plus_site_discoveries_confidence_check CHECK (classification_confidence >= 0 AND classification_confidence <= 1)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_admin_plus_site_discoveries_source_site
    ON admin_plus_site_discoveries(source_url, source_site_id)
    WHERE source_site_id <> '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_admin_plus_site_discoveries_register_url
    ON admin_plus_site_discoveries(register_url)
    WHERE register_url <> '';

CREATE INDEX IF NOT EXISTS idx_admin_plus_site_discoveries_provider
    ON admin_plus_site_discoveries(provider_type, classification_status, import_status);

CREATE INDEX IF NOT EXISTS idx_admin_plus_site_discoveries_supplier
    ON admin_plus_site_discoveries(supplier_id)
    WHERE supplier_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_admin_plus_site_discoveries_process
    ON admin_plus_site_discoveries(process_status, updated_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS admin_plus_supplier_registration_credentials (
    id BIGSERIAL PRIMARY KEY,
    discovery_id BIGINT NOT NULL REFERENCES admin_plus_site_discoveries(id) ON DELETE CASCADE,
    supplier_id BIGINT NOT NULL REFERENCES admin_plus_suppliers(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    password_ciphertext TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    verification_status TEXT NOT NULL DEFAULT '',
    extension_task_id BIGINT NULL REFERENCES admin_plus_extension_tasks(id) ON DELETE SET NULL,
    error_code TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    last_attempt_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_supplier_registration_credentials_status_check CHECK (
        status IN ('pending', 'queued', 'running', 'waiting_manual_verification', 'succeeded', 'failed')
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_admin_plus_supplier_registration_credentials_discovery
    ON admin_plus_supplier_registration_credentials(discovery_id);

CREATE INDEX IF NOT EXISTS idx_admin_plus_supplier_registration_credentials_task
    ON admin_plus_supplier_registration_credentials(extension_task_id)
    WHERE extension_task_id IS NOT NULL;

ALTER TABLE admin_plus_extension_tasks
    DROP CONSTRAINT IF EXISTS admin_plus_extension_tasks_type_check;

ALTER TABLE admin_plus_extension_tasks
    ADD CONSTRAINT admin_plus_extension_tasks_type_check
    CHECK (type IN (
        'fetch_rates',
        'fetch_groups',
        'fetch_balance',
        'fetch_announcements',
        'fetch_promotions',
        'export_bills',
        'fetch_usage_costs',
        'fetch_health',
        'check_supplier_channels',
        'capture_supplier_session',
        'register_supplier_account'
    ));
