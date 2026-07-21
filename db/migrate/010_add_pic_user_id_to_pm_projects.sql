-- Overview tab on /pm/projects/:id lets a user assign a Project Management
-- PIC (distinct from the CRM project owner in projects.assigned_to). No
-- column existed to store it.
ALTER TABLE pm_projects ADD COLUMN IF NOT EXISTS pic_user_id BIGINT;
