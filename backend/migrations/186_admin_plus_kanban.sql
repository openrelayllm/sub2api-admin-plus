CREATE TABLE IF NOT EXISTS admin_plus_market_price_snapshots (
    id BIGSERIAL PRIMARY KEY,
    source_type TEXT NOT NULL DEFAULT 'manual',
    source_name TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    site_id BIGINT NULL REFERENCES admin_plus_site_catalog_sites(id) ON DELETE SET NULL,
    supplier_id BIGINT NULL REFERENCES admin_plus_suppliers(id) ON DELETE SET NULL,
    model TEXT NOT NULL,
    billing_mode TEXT NOT NULL DEFAULT 'tokens',
    price_item TEXT NOT NULL DEFAULT 'blended',
    unit TEXT NOT NULL DEFAULT '1m_tokens',
    currency TEXT NOT NULL DEFAULT 'USD',
    price_micros BIGINT NOT NULL DEFAULT 0,
    package_label TEXT NOT NULL DEFAULT '',
    package_price_cents BIGINT NULL,
    package_quota TEXT NOT NULL DEFAULT '',
    rate_multiplier DOUBLE PRECISION NULL,
    min_recharge_cents BIGINT NULL,
    bonus_percent DOUBLE PRECISION NULL,
    confidence DOUBLE PRECISION NOT NULL DEFAULT 1,
    observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_market_price_source_type_check CHECK (source_type IN ('manual', 'site_catalog', 'site_discovery', 'provider_page', 'api')),
    CONSTRAINT admin_plus_market_price_confidence_check CHECK (confidence >= 0 AND confidence <= 1)
);

CREATE INDEX IF NOT EXISTS idx_admin_plus_market_price_model_observed
    ON admin_plus_market_price_snapshots(model, observed_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_admin_plus_market_price_source_observed
    ON admin_plus_market_price_snapshots(source_type, source_name, observed_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_admin_plus_market_price_site_observed
    ON admin_plus_market_price_snapshots(site_id, observed_at DESC, id DESC)
    WHERE site_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS admin_plus_cache_efficiency_snapshots (
    id BIGSERIAL PRIMARY KEY,
    supply_type TEXT NOT NULL DEFAULT 'supplier',
    supplier_id BIGINT NULL REFERENCES admin_plus_suppliers(id) ON DELETE SET NULL,
    local_sub2api_account_id BIGINT NULL,
    model TEXT NOT NULL,
    routing_strategy TEXT NOT NULL DEFAULT 'unknown',
    sticky_scope TEXT NOT NULL DEFAULT 'none',
    sample_requests INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens BIGINT NOT NULL DEFAULT 0,
    cache_write_tokens BIGINT NOT NULL DEFAULT 0,
    input_tokens BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    cache_hit_ratio DOUBLE PRECISION NOT NULL DEFAULT 0,
    duplicate_input_tokens BIGINT NOT NULL DEFAULT 0,
    estimated_waste_cents BIGINT NOT NULL DEFAULT 0,
    avg_ttft_ms BIGINT NULL,
    avg_total_latency_ms BIGINT NULL,
    status TEXT NOT NULL DEFAULT 'unknown',
    notes TEXT NOT NULL DEFAULT '',
    observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_cache_efficiency_supply_type_check CHECK (supply_type IN ('supplier', 'own_pool', 'competitor', 'custom')),
    CONSTRAINT admin_plus_cache_efficiency_strategy_check CHECK (routing_strategy IN ('unknown', 'fixed_account', 'round_robin', 'weighted_round_robin', 'sticky', 'least_loaded', 'custom')),
    CONSTRAINT admin_plus_cache_efficiency_scope_check CHECK (sticky_scope IN ('none', 'user', 'api_key', 'project', 'session', 'organization', 'custom')),
    CONSTRAINT admin_plus_cache_efficiency_status_check CHECK (status IN ('unknown', 'healthy', 'watching', 'risky', 'bad')),
    CONSTRAINT admin_plus_cache_efficiency_ratio_check CHECK (cache_hit_ratio >= 0 AND cache_hit_ratio <= 1)
);

CREATE INDEX IF NOT EXISTS idx_admin_plus_cache_efficiency_model_observed
    ON admin_plus_cache_efficiency_snapshots(model, observed_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_admin_plus_cache_efficiency_supplier_observed
    ON admin_plus_cache_efficiency_snapshots(supplier_id, observed_at DESC, id DESC)
    WHERE supplier_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_admin_plus_cache_efficiency_account_observed
    ON admin_plus_cache_efficiency_snapshots(local_sub2api_account_id, observed_at DESC, id DESC)
    WHERE local_sub2api_account_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS admin_plus_supply_quality_snapshots (
    id BIGSERIAL PRIMARY KEY,
    supply_type TEXT NOT NULL DEFAULT 'supplier',
    supplier_id BIGINT NULL REFERENCES admin_plus_suppliers(id) ON DELETE SET NULL,
    local_sub2api_account_id BIGINT NULL,
    model TEXT NOT NULL DEFAULT '',
    availability_ratio DOUBLE PRECISION NOT NULL DEFAULT 0,
    error_ratio DOUBLE PRECISION NOT NULL DEFAULT 0,
    avg_ttft_ms BIGINT NULL,
    avg_total_latency_ms BIGINT NULL,
    cache_hit_ratio DOUBLE PRECISION NOT NULL DEFAULT 0,
    purity_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    usage_trust_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    balance_risk_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    concurrency_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    quality_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    decision TEXT NOT NULL DEFAULT 'watching',
    notes TEXT NOT NULL DEFAULT '',
    observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_supply_quality_supply_type_check CHECK (supply_type IN ('supplier', 'own_pool', 'competitor', 'custom')),
    CONSTRAINT admin_plus_supply_quality_decision_check CHECK (decision IN ('production', 'watching', 'low_priority', 'paused', 'blocked')),
    CONSTRAINT admin_plus_supply_quality_ratio_check CHECK (
        availability_ratio >= 0 AND availability_ratio <= 1
        AND error_ratio >= 0 AND error_ratio <= 1
        AND cache_hit_ratio >= 0 AND cache_hit_ratio <= 1
        AND purity_score >= 0 AND purity_score <= 100
        AND usage_trust_score >= 0 AND usage_trust_score <= 100
        AND balance_risk_score >= 0 AND balance_risk_score <= 100
        AND concurrency_score >= 0 AND concurrency_score <= 100
        AND quality_score >= 0 AND quality_score <= 100
    )
);

CREATE INDEX IF NOT EXISTS idx_admin_plus_supply_quality_model_observed
    ON admin_plus_supply_quality_snapshots(model, observed_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_admin_plus_supply_quality_supplier_observed
    ON admin_plus_supply_quality_snapshots(supplier_id, observed_at DESC, id DESC)
    WHERE supplier_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_admin_plus_supply_quality_decision_observed
    ON admin_plus_supply_quality_snapshots(decision, observed_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS admin_plus_acceptance_reports (
    id BIGSERIAL PRIMARY KEY,
    supply_type TEXT NOT NULL DEFAULT 'supplier',
    supplier_id BIGINT NULL REFERENCES admin_plus_suppliers(id) ON DELETE SET NULL,
    local_sub2api_account_id BIGINT NULL,
    model TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'watching',
    connectivity_status TEXT NOT NULL DEFAULT 'unknown',
    model_list_status TEXT NOT NULL DEFAULT 'unknown',
    purity_status TEXT NOT NULL DEFAULT 'unknown',
    trial_call_status TEXT NOT NULL DEFAULT 'unknown',
    usage_metering_status TEXT NOT NULL DEFAULT 'unknown',
    cache_audit_status TEXT NOT NULL DEFAULT 'unknown',
    balance_status TEXT NOT NULL DEFAULT 'unknown',
    concurrency_status TEXT NOT NULL DEFAULT 'unknown',
    failure_reason TEXT NOT NULL DEFAULT '',
    recommendation TEXT NOT NULL DEFAULT '',
    report_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_acceptance_supply_type_check CHECK (supply_type IN ('supplier', 'own_pool', 'competitor', 'custom')),
    CONSTRAINT admin_plus_acceptance_status_check CHECK (status IN ('production', 'watching', 'low_priority', 'paused', 'blocked')),
    CONSTRAINT admin_plus_acceptance_step_status_check CHECK (
        connectivity_status IN ('unknown', 'pass', 'warn', 'fail')
        AND model_list_status IN ('unknown', 'pass', 'warn', 'fail')
        AND purity_status IN ('unknown', 'pass', 'warn', 'fail')
        AND trial_call_status IN ('unknown', 'pass', 'warn', 'fail')
        AND usage_metering_status IN ('unknown', 'pass', 'warn', 'fail')
        AND cache_audit_status IN ('unknown', 'pass', 'warn', 'fail')
        AND balance_status IN ('unknown', 'pass', 'warn', 'fail')
        AND concurrency_status IN ('unknown', 'pass', 'warn', 'fail')
    )
);

CREATE INDEX IF NOT EXISTS idx_admin_plus_acceptance_supplier_observed
    ON admin_plus_acceptance_reports(supplier_id, observed_at DESC, id DESC)
    WHERE supplier_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_admin_plus_acceptance_account_observed
    ON admin_plus_acceptance_reports(local_sub2api_account_id, observed_at DESC, id DESC)
    WHERE local_sub2api_account_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_admin_plus_acceptance_model_observed
    ON admin_plus_acceptance_reports(model, observed_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_admin_plus_acceptance_status_observed
    ON admin_plus_acceptance_reports(status, observed_at DESC, id DESC);
