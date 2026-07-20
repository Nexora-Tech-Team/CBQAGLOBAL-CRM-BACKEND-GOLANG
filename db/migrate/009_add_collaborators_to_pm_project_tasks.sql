-- pm_tasks (legacy Library module) already has a `collaborators` column,
-- but pm_project_tasks (the CRM-linked PM baru workspace) never got one —
-- so tasks on /pm/projects/:id can't record collaborators like the legacy
-- Kanban tasks can. Match the legacy shape (plain TEXT, comma-separated).
ALTER TABLE pm_project_tasks ADD COLUMN IF NOT EXISTS collaborators TEXT;
