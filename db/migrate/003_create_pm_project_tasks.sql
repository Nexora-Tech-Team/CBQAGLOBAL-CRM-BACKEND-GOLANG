-- Task board for real CRM projects (Advisory / Audit Program services).
-- Namespaced with pm_ like the rest of the Project Management module.
-- Uses an application-generated UUID primary key (see domain/pm/service),
-- kept separate from the legacy pm_tasks table (BIGSERIAL PK) to avoid
-- colliding with the existing self-contained pm_* dataset.

CREATE TABLE IF NOT EXISTS pm_project_tasks (
    id UUID PRIMARY KEY,
    crm_project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    parent_task_id UUID REFERENCES pm_project_tasks(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    description TEXT,
    assigned_to BIGINT REFERENCES users(id) ON DELETE SET NULL,
    start_date TIMESTAMP,
    deadline TIMESTAMP,
    labels TEXT,
    points SMALLINT NOT NULL DEFAULT 1,
    status VARCHAR(30) NOT NULL DEFAULT 'to_do',
    status_id BIGINT REFERENCES pm_task_statuses(id),
    priority_id BIGINT NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    status_changed_at TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_pm_project_tasks_crm_project_id ON pm_project_tasks(crm_project_id);
CREATE INDEX IF NOT EXISTS idx_pm_project_tasks_assigned_to ON pm_project_tasks(assigned_to);
CREATE INDEX IF NOT EXISTS idx_pm_project_tasks_deleted ON pm_project_tasks(deleted);
