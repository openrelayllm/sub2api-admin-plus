ALTER TABLE admin_plus_suppliers
    ADD COLUMN IF NOT EXISTS recharge_multiplier DOUBLE PRECISION NOT NULL DEFAULT 1;

UPDATE admin_plus_suppliers
SET recharge_multiplier = 1
WHERE recharge_multiplier IS NULL OR recharge_multiplier <= 0;

DO $$
BEGIN
    ALTER TABLE admin_plus_suppliers
        ADD CONSTRAINT admin_plus_suppliers_recharge_multiplier_check CHECK (recharge_multiplier > 0);
EXCEPTION WHEN duplicate_object THEN
    NULL;
END $$;

ALTER TABLE admin_plus_supplier_funding_transactions
    ADD COLUMN IF NOT EXISTS recharge_multiplier DOUBLE PRECISION NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS actual_payment_cents BIGINT NOT NULL DEFAULT 0;

UPDATE admin_plus_supplier_funding_transactions
SET recharge_multiplier = 1
WHERE recharge_multiplier IS NULL OR recharge_multiplier <= 0;

UPDATE admin_plus_supplier_funding_transactions
SET actual_payment_cents = CASE
    WHEN actual_payment_cents > 0 THEN actual_payment_cents
    WHEN cash_amount_cents > 0 THEN cash_amount_cents
    ELSE amount_cents
END;

DO $$
BEGIN
    ALTER TABLE admin_plus_supplier_funding_transactions
        ADD CONSTRAINT admin_plus_supplier_funding_payment_check CHECK (
            recharge_multiplier > 0
            AND actual_payment_cents >= 0
        );
EXCEPTION WHEN duplicate_object THEN
    NULL;
END $$;

ALTER TABLE admin_plus_supplier_cost_ledger_entries
    ADD COLUMN IF NOT EXISTS actual_payment_cents BIGINT NOT NULL DEFAULT 0;

UPDATE admin_plus_supplier_cost_ledger_entries
SET actual_payment_cents = CASE
    WHEN actual_payment_cents <> 0 THEN actual_payment_cents
    WHEN cash_amount_cents <> 0 THEN cash_amount_cents
    ELSE amount_cents
END;

ALTER TABLE admin_plus_supplier_cost_snapshots
    ADD COLUMN IF NOT EXISTS recharge_actual_payment_cents BIGINT NOT NULL DEFAULT 0;

UPDATE admin_plus_supplier_cost_snapshots
SET recharge_actual_payment_cents = CASE
    WHEN recharge_actual_payment_cents > 0 THEN recharge_actual_payment_cents
    WHEN completed_funding_cash_cents > 0 THEN completed_funding_cash_cents + entitlement_amount_cents
    ELSE completed_funding_amount_cents + entitlement_amount_cents
END;
