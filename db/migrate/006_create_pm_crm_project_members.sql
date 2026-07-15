-- Dedicated Team CRUD for CRM-project-linked PM tasks (Project Detail > Team tab).
-- Previously the Team tab only derived members from who already had a task
-- assigned in pm_project_tasks — there was no way to add/remove a member
-- directly. This table lets a project explicitly manage its member list,
-- scoped by crm_project_id so each project only sees/edits its own team.
CREATE TABLE IF NOT EXISTS pm_crm_project_members (
    id UUID PRIMARY KEY,
    crm_project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(100),
    is_leader BOOLEAN NOT NULL DEFAULT FALSE,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_pm_crm_project_members_project_user
    ON pm_crm_project_members(crm_project_id, user_id) WHERE deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_pm_crm_project_members_project_id ON pm_crm_project_members(crm_project_id);
