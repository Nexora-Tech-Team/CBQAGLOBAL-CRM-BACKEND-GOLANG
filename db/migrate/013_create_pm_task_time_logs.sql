-- Clock in/out sessions for pm_project_tasks (Work Timer on the Task Detail
-- drawer). One row per session: started_at set on clock-in, ended_at +
-- duration_seconds set on clock-out. An "active" session is one row per
-- user with ended_at IS NULL — a user may only have one such row at a time
-- across ALL tasks (enforced in the Go service layer via a transaction, not
-- a DB constraint, since a partial unique index on user_id WHERE ended_at
-- IS NULL would also work but the app-level check gives a clean 409 message
-- instead of a raw constraint-violation error).
--
-- No FK on task_id (same reasoning as pm_task_activity_logs): task deletion
-- is soft-delete, so time log history must never be at risk from a future
-- hard-delete path either.
CREATE TABLE IF NOT EXISTS pm_task_time_logs (
    id BIGSERIAL PRIMARY KEY,
    task_id UUID NOT NULL,
    project_id BIGINT,
    user_id BIGINT NOT NULL,
    started_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMP WITHOUT TIME ZONE,
    duration_seconds BIGINT,
    note TEXT,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITHOUT TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_pm_task_time_logs_task_id ON pm_task_time_logs(task_id);
CREATE INDEX IF NOT EXISTS idx_pm_task_time_logs_user_id ON pm_task_time_logs(user_id);

-- Every "is this user already clocked in somewhere" / "close this user's
-- active session on this task" lookup filters on ended_at IS NULL — a
-- partial index on just that open-session subset stays small and fast no
-- matter how large the closed-session history grows.
CREATE INDEX IF NOT EXISTS idx_pm_task_time_logs_active ON pm_task_time_logs(user_id, task_id)
    WHERE ended_at IS NULL AND deleted_at IS NULL;

-- "One active clock-in per user" is checked in the Go service layer for a
-- clean 409 response, but that check-then-insert has a race window under
-- concurrency — this UNIQUE partial index is the actual guarantee: Postgres
-- rejects a second open (ended_at IS NULL) row for the same user_id outright,
-- so two near-simultaneous clock-ins can never both succeed.
CREATE UNIQUE INDEX IF NOT EXISTS uq_pm_task_time_logs_one_active_per_user ON pm_task_time_logs(user_id)
    WHERE ended_at IS NULL AND deleted_at IS NULL;
