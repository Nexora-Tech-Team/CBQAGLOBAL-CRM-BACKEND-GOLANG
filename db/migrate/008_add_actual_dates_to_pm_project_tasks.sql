-- Same gap as progress_pct (migration 007): the "New Task" form has always
-- collected Actual Start / Actual Finish, but pm_project_tasks had nowhere
-- to store them, so they were silently dropped on create and always showed
-- blank in the Tasks list and task detail popup.
ALTER TABLE pm_project_tasks ADD COLUMN IF NOT EXISTS actual_start_date TIMESTAMP;
ALTER TABLE pm_project_tasks ADD COLUMN IF NOT EXISTS actual_finish_date TIMESTAMP;
