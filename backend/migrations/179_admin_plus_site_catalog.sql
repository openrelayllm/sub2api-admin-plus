CREATE TABLE IF NOT EXISTS admin_plus_site_catalog_sites (
    id BIGSERIAL PRIMARY KEY,
    slug TEXT NOT NULL,
    canonical_host TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    short_name TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    provider_type TEXT NOT NULL DEFAULT '',
    site_kind TEXT NOT NULL DEFAULT 'api_relay',
    status TEXT NOT NULL DEFAULT 'draft',
    visibility TEXT NOT NULL DEFAULT 'public',
    quality_status TEXT NOT NULL DEFAULT 'needs_review',
    recommendation_level TEXT NOT NULL DEFAULT 'none',
    recommendation_reason TEXT NOT NULL DEFAULT '',
    risk_level TEXT NOT NULL DEFAULT 'unknown',
    logo_url TEXT NOT NULL DEFAULT '',
    screenshot_url TEXT NOT NULL DEFAULT '',
    primary_language TEXT NOT NULL DEFAULT '',
    country_or_region TEXT NOT NULL DEFAULT '',
    supplier_id BIGINT NULL REFERENCES admin_plus_suppliers(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    published_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_site_catalog_sites_provider_type_check CHECK (provider_type IN ('', 'new_api', 'sub2api')),
    CONSTRAINT admin_plus_site_catalog_sites_kind_check CHECK (site_kind IN ('api_relay', 'official', 'tool', 'client', 'benchmark', 'other')),
    CONSTRAINT admin_plus_site_catalog_sites_status_check CHECK (status IN ('draft', 'reviewing', 'published', 'archived')),
    CONSTRAINT admin_plus_site_catalog_sites_visibility_check CHECK (visibility IN ('public', 'private')),
    CONSTRAINT admin_plus_site_catalog_sites_quality_check CHECK (quality_status IN ('complete', 'needs_review', 'link_broken', 'duplicate')),
    CONSTRAINT admin_plus_site_catalog_sites_recommendation_check CHECK (recommendation_level IN ('none', 'normal', 'featured', 'avoid')),
    CONSTRAINT admin_plus_site_catalog_sites_risk_check CHECK (risk_level IN ('unknown', 'low', 'medium', 'high'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_admin_plus_site_catalog_sites_slug
    ON admin_plus_site_catalog_sites(slug);

CREATE UNIQUE INDEX IF NOT EXISTS idx_admin_plus_site_catalog_sites_canonical_host
    ON admin_plus_site_catalog_sites(canonical_host)
    WHERE canonical_host <> '';

CREATE INDEX IF NOT EXISTS idx_admin_plus_site_catalog_sites_status
    ON admin_plus_site_catalog_sites(status, visibility, quality_status, updated_at DESC);

CREATE TABLE IF NOT EXISTS admin_plus_site_catalog_links (
    id BIGSERIAL PRIMARY KEY,
    site_id BIGINT NOT NULL REFERENCES admin_plus_site_catalog_sites(id) ON DELETE CASCADE,
    link_type TEXT NOT NULL,
    url TEXT NOT NULL,
    label TEXT NOT NULL DEFAULT '',
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL DEFAULT 'unknown',
    last_checked_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_plus_site_catalog_links_type_check CHECK (link_type IN ('homepage', 'register', 'dashboard', 'api_base', 'recharge', 'docs', 'contact')),
    CONSTRAINT admin_plus_site_catalog_links_status_check CHECK (status IN ('unknown', 'ok', 'broken'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_admin_plus_site_catalog_links_unique
    ON admin_plus_site_catalog_links(site_id, link_type, url);

CREATE TABLE IF NOT EXISTS admin_plus_site_catalog_categories (
    id BIGSERIAL PRIMARY KEY,
    parent_id BIGINT NULL REFERENCES admin_plus_site_catalog_categories(id) ON DELETE SET NULL,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    display_order INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_admin_plus_site_catalog_categories_slug
    ON admin_plus_site_catalog_categories(slug);

CREATE TABLE IF NOT EXISTS admin_plus_site_catalog_site_categories (
    site_id BIGINT NOT NULL REFERENCES admin_plus_site_catalog_sites(id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES admin_plus_site_catalog_categories(id) ON DELETE CASCADE,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    display_order INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (site_id, category_id)
);

CREATE TABLE IF NOT EXISTS admin_plus_site_catalog_tags (
    id BIGSERIAL PRIMARY KEY,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    tag_type TEXT NOT NULL DEFAULT 'capability',
    color TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_admin_plus_site_catalog_tags_slug
    ON admin_plus_site_catalog_tags(slug);

CREATE TABLE IF NOT EXISTS admin_plus_site_catalog_site_tags (
    site_id BIGINT NOT NULL REFERENCES admin_plus_site_catalog_sites(id) ON DELETE CASCADE,
    tag_id BIGINT NOT NULL REFERENCES admin_plus_site_catalog_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (site_id, tag_id)
);

CREATE TABLE IF NOT EXISTS admin_plus_site_catalog_sources (
    id BIGSERIAL PRIMARY KEY,
    site_id BIGINT NOT NULL REFERENCES admin_plus_site_catalog_sites(id) ON DELETE CASCADE,
    source_type TEXT NOT NULL,
    source_name TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    source_external_id TEXT NOT NULL DEFAULT '',
    discovery_candidate_id BIGINT NULL,
    observed_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_admin_plus_site_catalog_sources_candidate
    ON admin_plus_site_catalog_sources(site_id, discovery_candidate_id)
    WHERE discovery_candidate_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS admin_plus_site_catalog_metrics (
    id BIGSERIAL PRIMARY KEY,
    site_id BIGINT NOT NULL REFERENCES admin_plus_site_catalog_sites(id) ON DELETE CASCADE,
    metric_type TEXT NOT NULL,
    metric_key TEXT NOT NULL,
    metric_value DOUBLE PRECISION NULL,
    value_text TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE admin_plus_site_discoveries
    ADD COLUMN IF NOT EXISTS source_category TEXT NOT NULL DEFAULT '';

ALTER TABLE admin_plus_site_discoveries
    ADD COLUMN IF NOT EXISTS process_status TEXT NOT NULL DEFAULT 'unprocessed';

ALTER TABLE admin_plus_site_discoveries
    ADD COLUMN IF NOT EXISTS catalog_site_id BIGINT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'admin_plus_site_discoveries_process_status_check'
    ) THEN
        ALTER TABLE admin_plus_site_discoveries
            ADD CONSTRAINT admin_plus_site_discoveries_process_status_check
            CHECK (process_status IN ('unprocessed', 'added_to_catalog', 'registered', 'ignored'));
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'admin_plus_site_discoveries_catalog_site_fk'
    ) THEN
        ALTER TABLE admin_plus_site_discoveries
            ADD CONSTRAINT admin_plus_site_discoveries_catalog_site_fk
            FOREIGN KEY (catalog_site_id) REFERENCES admin_plus_site_catalog_sites(id) ON DELETE SET NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_admin_plus_site_discoveries_process
    ON admin_plus_site_discoveries(process_status, updated_at DESC, id DESC);

INSERT INTO admin_plus_site_catalog_categories (slug, name, description, display_order, enabled)
VALUES
    ('third-party', '第三方中转', '第三方 API 中转和聚合服务', 10, TRUE),
    ('official', '官方平台', '模型官方平台和云厂商平台', 20, TRUE),
    ('opensource', '开源工具', '开源项目、网关和工具', 30, TRUE),
    ('clients', '客户端', 'AI 客户端和桌面工具', 40, TRUE),
    ('benchmarks', '评测平台', '模型评测和排行榜', 50, TRUE),
    ('other', '其他', '暂未分类站点', 999, TRUE)
ON CONFLICT (slug) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    display_order = EXCLUDED.display_order,
    enabled = EXCLUDED.enabled,
    updated_at = NOW();
INSERT INTO admin_plus_site_catalog_tags (slug, name, tag_type, color, enabled)
VALUES
    ('new-api', 'new-api', 'capability', '#0ea5e9', TRUE),
    ('sub2api', 'sub2api', 'capability', '#8b5cf6', TRUE),
    ('cheap', '便宜', 'pricing', '#10b981', TRUE),
    ('stable', '稳定', 'quality', '#059669', TRUE),
    ('manual-verification', '需人工验证', 'risk', '#f59e0b', TRUE),
    ('recharge-supported', '支持充值', 'pricing', '#14b8a6', TRUE)
ON CONFLICT (slug) DO UPDATE
SET name = EXCLUDED.name,
    tag_type = EXCLUDED.tag_type,
    color = EXCLUDED.color,
    enabled = EXCLUDED.enabled,
    updated_at = NOW();
