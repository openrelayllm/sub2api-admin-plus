DO $$
BEGIN
    IF to_regclass('public.admin_plus_announcement_events') IS NULL
       AND to_regclass('public.admin_plus_promotion_events') IS NOT NULL THEN
        ALTER TABLE admin_plus_promotion_events RENAME TO admin_plus_announcement_events;
    END IF;
END $$;

ALTER TABLE IF EXISTS admin_plus_announcement_events
    DROP CONSTRAINT IF EXISTS admin_plus_promotion_events_type_check,
    DROP CONSTRAINT IF EXISTS admin_plus_announcement_events_type_check;

ALTER TABLE IF EXISTS admin_plus_announcement_events
    ADD CONSTRAINT admin_plus_announcement_events_type_check
    CHECK (type IN ('recharge_bonus', 'rate_discount', 'package_deal', 'limited_offer', 'maintenance', 'incident', 'notice', 'other'));

ALTER TABLE IF EXISTS admin_plus_announcement_events
    DROP CONSTRAINT IF EXISTS admin_plus_promotion_events_status_check,
    DROP CONSTRAINT IF EXISTS admin_plus_announcement_events_status_check;

ALTER TABLE IF EXISTS admin_plus_announcement_events
    ADD CONSTRAINT admin_plus_announcement_events_status_check
    CHECK (status IN ('open', 'acknowledged', 'ignored'));

ALTER TABLE IF EXISTS admin_plus_announcement_events
    DROP CONSTRAINT IF EXISTS admin_plus_promotion_events_recommendation_check,
    DROP CONSTRAINT IF EXISTS admin_plus_announcement_events_recommendation_check;

ALTER TABLE IF EXISTS admin_plus_announcement_events
    ADD CONSTRAINT admin_plus_announcement_events_recommendation_check
    CHECK (recommendation IN ('recharge_to_unlock', 'switch_candidate', 'monitor_only', 'informational'));

ALTER INDEX IF EXISTS idx_admin_plus_promotion_events_supplier_status
    RENAME TO idx_admin_plus_announcement_events_supplier_status;

ALTER INDEX IF EXISTS idx_admin_plus_promotion_events_recommendation
    RENAME TO idx_admin_plus_announcement_events_recommendation;

CREATE INDEX IF NOT EXISTS idx_admin_plus_announcement_events_supplier_status
    ON admin_plus_announcement_events(supplier_id, status, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_admin_plus_announcement_events_recommendation
    ON admin_plus_announcement_events(recommendation, status, created_at DESC, id DESC);
