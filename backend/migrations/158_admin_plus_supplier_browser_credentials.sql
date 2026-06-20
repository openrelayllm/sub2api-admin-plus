ALTER TABLE admin_plus_suppliers
    ADD COLUMN IF NOT EXISTS browser_login_username_ciphertext TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS browser_login_password_ciphertext TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS browser_login_token_ciphertext TEXT NOT NULL DEFAULT '';
