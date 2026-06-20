ALTER TABLE admin_plus_extension_tasks
    ADD COLUMN IF NOT EXISTS schedule_key TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS admin_plus_extension_tasks_schedule_key_unique
    ON admin_plus_extension_tasks(schedule_key)
    WHERE schedule_key <> '';
