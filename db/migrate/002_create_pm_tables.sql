-- Project Management module tables.
-- Namespaced with pm_ to keep them isolated from other business domains.

CREATE TABLE IF NOT EXISTS pm_task_statuses (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(100) NOT NULL,
    key_name VARCHAR(100) NOT NULL UNIQUE,
    color VARCHAR(7) NOT NULL DEFAULT '#1672B9',
    sort_order INTEGER NOT NULL DEFAULT 0,
    hide_from_kanban BOOLEAN NOT NULL DEFAULT FALSE,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

INSERT INTO pm_task_statuses (title, key_name, color, sort_order, hide_from_kanban)
VALUES
    ('To Do', 'to_do', '#F9A52D', 0, FALSE),
    ('In progress', 'in_progress', '#1672B9', 1, FALSE),
    ('Done', 'done', '#00B393', 2, FALSE)
ON CONFLICT (key_name) DO NOTHING;

CREATE TABLE IF NOT EXISTS pm_clients (
    id BIGSERIAL PRIMARY KEY,
    company_name TEXT NOT NULL,
    type VARCHAR(50),
    address TEXT,
    city VARCHAR(150),
    state VARCHAR(150),
    zip VARCHAR(50),
    country VARCHAR(150),
    created_date TIMESTAMP,
    website TEXT,
    phone VARCHAR(100),
    currency_symbol VARCHAR(20),
    labels TEXT,
    owner_id UUID REFERENCES musers(id) ON DELETE SET NULL,
    created_by UUID REFERENCES musers(id) ON DELETE SET NULL,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_pm_clients_deleted ON pm_clients(deleted);

CREATE TABLE IF NOT EXISTS pm_project_statuses (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(100) NOT NULL,
    key_name VARCHAR(100),
    icon VARCHAR(100),
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pm_projects (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    project_type VARCHAR(80),
    start_date DATE,
    deadline DATE,
    client_id BIGINT REFERENCES pm_clients(id) ON DELETE SET NULL,
    created_date TIMESTAMP,
    created_by UUID REFERENCES musers(id) ON DELETE SET NULL,
    status VARCHAR(50),
    status_id BIGINT REFERENCES pm_project_statuses(id) ON DELETE SET NULL,
    labels TEXT,
    price NUMERIC(14, 2),
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_pm_projects_deleted ON pm_projects(deleted);
CREATE INDEX IF NOT EXISTS idx_pm_projects_client_id ON pm_projects(client_id);
CREATE INDEX IF NOT EXISTS idx_pm_projects_status_id ON pm_projects(status_id);

CREATE TABLE IF NOT EXISTS pm_project_members (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID REFERENCES musers(id) ON DELETE CASCADE,
    project_id BIGINT NOT NULL REFERENCES pm_projects(id) ON DELETE CASCADE,
    is_leader BOOLEAN NOT NULL DEFAULT FALSE,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_pm_project_members_project_id ON pm_project_members(project_id);
CREATE INDEX IF NOT EXISTS idx_pm_project_members_user_id ON pm_project_members(user_id);

CREATE TABLE IF NOT EXISTS pm_tasks (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    project_id BIGINT REFERENCES pm_projects(id) ON DELETE SET NULL,
    assigned_to UUID REFERENCES musers(id) ON DELETE SET NULL,
    deadline TIMESTAMP,
    labels TEXT,
    points SMALLINT NOT NULL DEFAULT 1,
    status VARCHAR(30) NOT NULL DEFAULT 'to_do',
    status_id BIGINT REFERENCES pm_task_statuses(id),
    priority_id BIGINT NOT NULL DEFAULT 0,
    start_date TIMESTAMP,
    collaborators TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    parent_task_id BIGINT REFERENCES pm_tasks(id) ON DELETE SET NULL,
    ticket_id BIGINT NOT NULL DEFAULT 0,
    status_changed_at TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    context VARCHAR(30) NOT NULL DEFAULT 'general',
    created_by UUID REFERENCES musers(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_pm_tasks_status_id ON pm_tasks(status_id);
CREATE INDEX IF NOT EXISTS idx_pm_tasks_project_id ON pm_tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_pm_tasks_assigned_to ON pm_tasks(assigned_to);
CREATE INDEX IF NOT EXISTS idx_pm_tasks_deleted ON pm_tasks(deleted);

CREATE TABLE IF NOT EXISTS pm_tickets (
    id BIGSERIAL PRIMARY KEY,
    client_id BIGINT REFERENCES pm_clients(id) ON DELETE SET NULL,
    project_id BIGINT REFERENCES pm_projects(id) ON DELETE SET NULL,
    ticket_type_id BIGINT NOT NULL DEFAULT 0,
    title TEXT NOT NULL,
    created_by UUID REFERENCES musers(id) ON DELETE SET NULL,
    requested_by UUID REFERENCES musers(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    status VARCHAR(30) NOT NULL DEFAULT 'new',
    last_activity_at TIMESTAMP,
    assigned_to UUID REFERENCES musers(id) ON DELETE SET NULL,
    creator_name VARCHAR(100) NOT NULL DEFAULT '',
    creator_email VARCHAR(255) NOT NULL DEFAULT '',
    labels TEXT,
    task_id BIGINT NOT NULL DEFAULT 0,
    closed_at TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    cc_contacts_and_emails TEXT,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_pm_tickets_status ON pm_tickets(status);
CREATE INDEX IF NOT EXISTS idx_pm_tickets_project_id ON pm_tickets(project_id);
CREATE INDEX IF NOT EXISTS idx_pm_tickets_assigned_to ON pm_tickets(assigned_to);
CREATE INDEX IF NOT EXISTS idx_pm_tickets_deleted ON pm_tickets(deleted);

CREATE TABLE IF NOT EXISTS pm_ticket_comments (
    id BIGSERIAL PRIMARY KEY,
    created_by UUID REFERENCES musers(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    description TEXT NOT NULL,
    ticket_id BIGINT NOT NULL REFERENCES pm_tickets(id) ON DELETE CASCADE,
    files TEXT,
    is_note BOOLEAN NOT NULL DEFAULT FALSE,
    deleted BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_pm_ticket_comments_ticket_id ON pm_ticket_comments(ticket_id);

CREATE TABLE IF NOT EXISTS pm_ticket_templates (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    ticket_type_id BIGINT NOT NULL DEFAULT 0,
    private_note TEXT NOT NULL DEFAULT '',
    created_by UUID REFERENCES musers(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted BOOLEAN NOT NULL DEFAULT FALSE
);

INSERT INTO pm_ticket_templates (title, description, ticket_type_id, private_note)
SELECT 'General Support', 'Please describe the issue, expected result, and affected project.', 0, ''
WHERE NOT EXISTS (SELECT 1 FROM pm_ticket_templates WHERE title = 'General Support' AND deleted = FALSE);

CREATE TABLE IF NOT EXISTS pm_activity_logs (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES musers(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL,
    log_type VARCHAR(30) NOT NULL,
    log_type_title TEXT NOT NULL,
    log_type_id BIGINT NOT NULL DEFAULT 0,
    changes TEXT,
    log_for VARCHAR(30) NOT NULL DEFAULT '0',
    log_for_id BIGINT NOT NULL DEFAULT 0,
    deleted BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_pm_activity_logs_type_id ON pm_activity_logs(log_type, log_type_id);
CREATE INDEX IF NOT EXISTS idx_pm_activity_logs_created_at ON pm_activity_logs(created_at DESC);
