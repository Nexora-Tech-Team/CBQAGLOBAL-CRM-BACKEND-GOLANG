-- Manual Work Log (Requirement C): same pm_task_time_logs table backs both
-- real-time clock-in/out sessions and manually-entered ones (case: user
-- forgot to clock in/out). source distinguishes the two; created_by/
-- updated_by record who actually performed the write (may differ from
-- user_id — an admin/PM logging time on behalf of a teammate).
ALTER TABLE pm_task_time_logs
    ADD COLUMN IF NOT EXISTS source VARCHAR(20) NOT NULL DEFAULT 'realtime',
    ADD COLUMN IF NOT EXISTS created_by BIGINT,
    ADD COLUMN IF NOT EXISTS updated_by BIGINT;

CREATE INDEX IF NOT EXISTS idx_pm_task_time_logs_source ON pm_task_time_logs(source);

-- Backfill: every row that already exists predates this column and was
-- always a real-time clock-in/out session.
UPDATE pm_task_time_logs SET source = 'realtime' WHERE source IS NULL;
