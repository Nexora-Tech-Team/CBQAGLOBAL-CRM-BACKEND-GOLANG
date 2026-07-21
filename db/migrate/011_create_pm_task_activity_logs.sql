-- Audit trail for pm_project_tasks (the CRM-linked PM baru workspace task
-- board). Task Detail drawer's "Activity" section was a static placeholder;
-- this backs it with real, append-only history.
--
-- No FK on task_id: task deletion is already soft-delete (pm_project_tasks
-- .deleted = TRUE), so rows always remain resolvable — but keeping this
-- table FK-free means log rows are never at risk from any future hard-delete
-- path either, per "don't lose logs when a task is deleted".
CREATE TABLE IF NOT EXISTS pm_task_activity_logs (
    id BIGSERIAL PRIMARY KEY,
    task_id UUID NOT NULL,
    project_id BIGINT,
    actor_user_id BIGINT,
    actor_name TEXT,
    action VARCHAR(50) NOT NULL,
    field_name VARCHAR(50),
    old_value TEXT,
    new_value TEXT,
    description TEXT NOT NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pm_task_activity_logs_task_id ON pm_task_activity_logs(task_id);
CREATE INDEX IF NOT EXISTS idx_pm_task_activity_logs_project_id ON pm_task_activity_logs(project_id);
CREATE INDEX IF NOT EXISTS idx_pm_task_activity_logs_created_at ON pm_task_activity_logs(created_at);
