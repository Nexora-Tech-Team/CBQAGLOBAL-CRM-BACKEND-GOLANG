-- pm_project_tasks never had a progress column, even though the frontend's
-- "New Task" form has always collected a Progress % field (formData.progressPct)
-- — it was silently dropped on create, so every Kanban card / Gantt bar for a
-- real (non-sample) CRM project task showed a hardcoded 0%.
ALTER TABLE pm_project_tasks ADD COLUMN IF NOT EXISTS progress_pct SMALLINT NOT NULL DEFAULT 0;
