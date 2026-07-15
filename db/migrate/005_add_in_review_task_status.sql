-- The PM project-detail Kanban board (frontend KANBAN_COLS) already offers an
-- "In Review" column, but pm_task_statuses only seeded to_do/in_progress/done.
-- Add the missing row so tasks can actually move into that column.
UPDATE pm_task_statuses SET sort_order = 3 WHERE key_name = 'done';

INSERT INTO pm_task_statuses (title, key_name, color, sort_order, hide_from_kanban)
VALUES ('In Review', 'in_review', '#6f42c1', 2, FALSE)
ON CONFLICT (key_name) DO NOTHING;
