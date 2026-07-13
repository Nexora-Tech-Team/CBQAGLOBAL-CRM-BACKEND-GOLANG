-- Timesheet entries for the Project Management module's "IT Audit Timesheet" page.
-- Namespaced with pm_ like the rest of the module. Linked to the CRM-project task
-- board (pm_project_tasks / projects / users) since timesheet work is logged
-- against real CRM Advisory/Audit Program projects, not the standalone pm_tasks board.

CREATE TABLE IF NOT EXISTS pm_timesheets (
    id BIGSERIAL PRIMARY KEY,
    crm_project_id BIGINT REFERENCES projects(id) ON DELETE CASCADE,
    task_id UUID REFERENCES pm_project_tasks(id) ON DELETE SET NULL,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    work_date DATE NOT NULL,
    hours NUMERIC(5, 2) NOT NULL CHECK (hours > 0),
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    approved_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMP,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_pm_timesheets_crm_project_id ON pm_timesheets(crm_project_id);
CREATE INDEX IF NOT EXISTS idx_pm_timesheets_user_id ON pm_timesheets(user_id);
CREATE INDEX IF NOT EXISTS idx_pm_timesheets_work_date ON pm_timesheets(work_date);
CREATE INDEX IF NOT EXISTS idx_pm_timesheets_status ON pm_timesheets(status);
CREATE INDEX IF NOT EXISTS idx_pm_timesheets_deleted ON pm_timesheets(deleted);
