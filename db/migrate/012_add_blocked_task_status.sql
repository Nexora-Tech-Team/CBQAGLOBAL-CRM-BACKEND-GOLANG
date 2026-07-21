-- Add a selectable Kanban status for blocked tasks so automatic Project Stage
-- can surface the existing Blocked priority rule from real task data.
UPDATE pm_task_statuses SET sort_order = 4 WHERE key_name = 'done';

INSERT INTO pm_task_statuses (title, key_name, color, sort_order, hide_from_kanban)
VALUES ('Blocked', 'blocked', '#D63939', 3, FALSE)
ON CONFLICT (key_name) DO UPDATE SET
    title = EXCLUDED.title,
    color = EXCLUDED.color,
    sort_order = EXCLUDED.sort_order,
    hide_from_kanban = EXCLUDED.hide_from_kanban,
    deleted = FALSE,
    updated_at = NOW();
