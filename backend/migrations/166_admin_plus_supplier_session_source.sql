ALTER TABLE admin_plus_supplier_browser_sessions
    ADD COLUMN IF NOT EXISTS session_source TEXT NOT NULL DEFAULT 'browser_extension';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'admin_plus_supplier_browser_sessions_source_check'
    ) THEN
        ALTER TABLE admin_plus_supplier_browser_sessions
            ADD CONSTRAINT admin_plus_supplier_browser_sessions_source_check
            CHECK (session_source IN ('direct_login', 'browser_extension', 'manual_import'));
    END IF;
END $$;
