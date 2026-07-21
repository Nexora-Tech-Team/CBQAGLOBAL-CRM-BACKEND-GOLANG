package repository

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"erp-cbqa-global/domain/pm/model"
)

type RepositoryInterface interface {
	DashboardTotals() (model.Row, error)
	DashboardProjectsByStatus() ([]model.Row, error)
	DashboardTopClients() ([]model.Row, error)
	DashboardRecentProjects() ([]model.Row, error)
	DashboardTaskDistribution(crmProjectID, memberID *int64) (model.Row, error)

	Statuses() ([]model.Row, error)
	TasksByStatus(statusID int64) ([]model.Row, error)

	Projects(search string, clientID *int64, status string) ([]model.Row, error)
	Clients(search string) ([]model.Row, error)
	Members(projectID *int64, search string) ([]model.Row, error)

	Tasks(search string, projectID *int64, assignedTo string) ([]model.Row, error)
	TaskByID(id int64) (model.Row, error)
	ResolveStatusID(statusID *int64, statusKey string) (int64, error)
	StatusKeyByID(statusID int64) (string, error)
	StatusIDByKey(key string) (*int64, error)
	InsertTask(fields map[string]interface{}) (int64, error)
	UpdateTask(id int64, fields map[string]interface{}) error
	DeleteTask(id int64) error

	Tickets(status, search string) ([]model.Row, error)
	TicketByID(id int64) (model.Row, error)
	InsertTicket(fields map[string]interface{}) (int64, error)
	TouchTicketActivity(id int64) error

	TicketComments(ticketID int64) ([]model.Row, error)
	InsertTicketComment(fields map[string]interface{}) (int64, error)
	TicketCommentByID(id int64) (model.Row, error)

	TicketTemplates() ([]model.Row, error)
	ActivityLogs() ([]model.Row, error)
	LogActivity(action, logType, title string, typeID int64, logFor string, logForID int64, changes *string, createdBy *string) error

	// CRM-project-linked task board (real `projects` table, Advisory/Audit Program services).
	CrmProjects(search string) ([]model.Row, error)
	CrmProjectByID(id int64) (model.Row, error)
	UpdateCrmProject(id int64, fields map[string]interface{}) error
	UpsertPmProjectPic(crmProjectID int64, picUserID *int64) error
	ProjectTasksByCrmProject(crmProjectID int64) ([]model.Row, error)
	TeamByCrmProject(crmProjectID int64) ([]model.Row, error)
	ActivityByCrmProject(crmProjectID int64) ([]model.Row, error)
	InsertProjectTask(fields map[string]interface{}) error
	UpdateProjectTask(id string, fields map[string]interface{}) error
	ProjectTaskByID(id string) (model.Row, error)
	MoveProjectTaskByKey(id, statusKey string) error
	DeleteProjectTask(id string) error
	GanttMembers() ([]model.Row, error)

	// Task activity log (pm_task_activity_logs) — audit trail for the Task
	// Detail drawer's "Activity" section.
	InsertTaskActivityLog(fields map[string]interface{}) error
	TaskActivityLogs(taskID string) ([]model.Row, error)
	ResolveUserName(userID int64) (string, error)

	// Team tab CRUD (pm_crm_project_members) — explicit, project-scoped
	// membership, replacing the old "derive from assigned tasks" heuristic.
	CrmProjectMemberByID(id string, crmProjectID int64) (model.Row, error)
	InsertCrmProjectMember(fields map[string]interface{}) error
	UpdateCrmProjectMember(id string, crmProjectID int64, fields map[string]interface{}) error
	DeleteCrmProjectMember(id string, crmProjectID int64) error

	// Timesheets — IT Audit Timesheet page, logged against CRM projects.
	Timesheets(search string, userID, crmProjectID *int64, status string, from, to *string) ([]model.Row, error)
	TimesheetSummary(userID *int64) (model.Row, error)
	TimesheetByID(id int64) (model.Row, error)
	InsertTimesheet(fields map[string]interface{}) (int64, error)
	UpdateTimesheetStatus(id int64, status string, approvedBy *int64) error
	DeleteTimesheet(id int64) error
}

type repository struct {
	DB *gorm.DB
}

func Repository(db *gorm.DB) RepositoryInterface {
	return &repository{DB: db}
}

func like(search string) *string {
	s := strings.TrimSpace(search)
	if s == "" {
		return nil
	}
	pattern := "%" + strings.ToLower(s) + "%"
	return &pattern
}

func blankToNil(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func (r *repository) DashboardTotals() (model.Row, error) {
	var rows []model.Row
	if err := r.DB.Raw(`
		SELECT
		  (SELECT COUNT(*) FROM pm_tasks WHERE deleted = FALSE) AS total_tasks,
		  (SELECT COUNT(*) FROM pm_tasks t JOIN pm_task_statuses s ON s.id = t.status_id WHERE t.deleted = FALSE AND s.key_name = 'to_do') AS todo_tasks,
		  (SELECT COUNT(*) FROM pm_tasks t JOIN pm_task_statuses s ON s.id = t.status_id WHERE t.deleted = FALSE AND s.key_name = 'in_progress') AS in_progress_tasks,
		  (SELECT COUNT(*) FROM pm_tasks t JOIN pm_task_statuses s ON s.id = t.status_id WHERE t.deleted = FALSE AND s.key_name = 'done') AS done_tasks,
		  (SELECT COUNT(*) FROM pm_projects WHERE deleted = FALSE) AS total_projects,
		  (SELECT COUNT(*) FROM pm_clients WHERE deleted = FALSE) AS total_clients,
		  (SELECT COUNT(*) FROM pm_project_members WHERE deleted = FALSE) AS project_members,
		  (SELECT COUNT(*) FROM pm_tickets WHERE deleted = FALSE) AS total_tickets,
		  (SELECT COUNT(*) FROM pm_tickets WHERE deleted = FALSE AND status <> 'closed') AS open_tickets,
		  (SELECT COUNT(*) FROM pm_activity_logs WHERE deleted = FALSE) AS activity_logs
	`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return model.Row{}, nil
	}
	return rows[0], nil
}

func (r *repository) DashboardProjectsByStatus() ([]model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT
		  COALESCE(NULLIF(p.status, ''), 'unknown') AS status_key,
		  COALESCE(ps.title, INITCAP(REPLACE(COALESCE(NULLIF(p.status, ''), 'unknown'), '_', ' '))) AS status_title,
		  COUNT(*) AS count
		FROM pm_projects p
		LEFT JOIN pm_project_statuses ps ON ps.id = p.status_id
		WHERE p.deleted = FALSE
		GROUP BY COALESCE(NULLIF(p.status, ''), 'unknown'), COALESCE(ps.title, INITCAP(REPLACE(COALESCE(NULLIF(p.status, ''), 'unknown'), '_', ' ')))
		ORDER BY count DESC
	`).Scan(&rows).Error
	return rows, err
}

func (r *repository) DashboardTopClients() ([]model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT c.id AS client_id, c.company_name, COUNT(p.id) AS project_count
		FROM pm_clients c
		JOIN pm_projects p ON p.client_id = c.id AND p.deleted = FALSE
		WHERE c.deleted = FALSE
		GROUP BY c.id, c.company_name
		ORDER BY project_count DESC
		LIMIT 8
	`).Scan(&rows).Error
	return rows, err
}

func (r *repository) DashboardRecentProjects() ([]model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT p.id, p.title, p.status, p.deadline, p.created_date,
		       c.company_name AS client_name
		FROM pm_projects p
		LEFT JOIN pm_clients c ON c.id = p.client_id
		WHERE p.deleted = FALSE
		ORDER BY p.created_date DESC NULLS LAST, p.id DESC
		LIMIT 8
	`).Scan(&rows).Error
	return rows, err
}

// DashboardTaskDistribution groups CRM-project-linked tasks (pm_project_tasks)
// by their pm_task_statuses key_name, optionally scoped to a CRM project
// and/or an assignee. Returns a Row keyed by key_name, e.g.
// {"to_do": 4, "in_progress": 2, "in_review": 1, "done": 7}.
func (r *repository) DashboardTaskDistribution(crmProjectID, memberID *int64) (model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT s.key_name, COUNT(t.id) AS count
		FROM pm_task_statuses s
		LEFT JOIN pm_project_tasks t
		  ON t.status_id = s.id AND t.deleted = FALSE
		  AND (?::bigint IS NULL OR t.crm_project_id = ?)
		  AND (?::bigint IS NULL OR t.assigned_to = ?)
		WHERE s.deleted = FALSE
		GROUP BY s.key_name, s.sort_order
		ORDER BY s.sort_order ASC
	`, crmProjectID, crmProjectID, memberID, memberID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := model.Row{}
	for _, row := range rows {
		key, _ := row["key_name"].(string)
		result[key] = row["count"]
	}
	return result, nil
}

func (r *repository) Statuses() ([]model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT id, title, key_name, color, sort_order, hide_from_kanban
		FROM pm_task_statuses
		WHERE deleted = FALSE
		ORDER BY sort_order ASC, id ASC
	`).Scan(&rows).Error
	return rows, err
}

func (r *repository) TasksByStatus(statusID int64) ([]model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT t.*, s.title AS status_title, s.key_name AS status_key, s.color AS status_color
		FROM pm_tasks t
		JOIN pm_task_statuses s ON s.id = t.status_id
		WHERE t.deleted = FALSE AND t.status_id = ?
		ORDER BY t.sort_order ASC, t.id DESC
	`, statusID).Scan(&rows).Error
	return rows, err
}

func (r *repository) Projects(search string, clientID *int64, status string) ([]model.Row, error) {
	s := like(search)
	st := blankToNil(status)
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT
		  p.id, p.title, p.description, p.project_type, p.start_date, p.deadline,
		  p.client_id, p.created_date, p.created_by, p.status, p.status_id, p.labels, p.price,
		  c.company_name AS client_name,
		  ps.title AS status_title, ps.key_name AS status_key,
		  (SELECT COUNT(*) FROM pm_project_members m WHERE m.deleted = FALSE AND m.project_id = p.id) AS member_count,
		  (SELECT COUNT(*) FROM pm_tasks t WHERE t.deleted = FALSE AND t.project_id = p.id) AS task_count
		FROM pm_projects p
		LEFT JOIN pm_clients c ON c.id = p.client_id
		LEFT JOIN pm_project_statuses ps ON ps.id = p.status_id
		WHERE p.deleted = FALSE
		  AND (?::text IS NULL OR LOWER(p.title) LIKE ?
		       OR LOWER(COALESCE(p.description, '')) LIKE ?
		       OR LOWER(COALESCE(c.company_name, '')) LIKE ?)
		  AND (?::bigint IS NULL OR p.client_id = ?)
		  AND (?::text IS NULL OR p.status = ? OR ps.key_name = ?)
		ORDER BY p.created_date DESC NULLS LAST, p.id DESC
		LIMIT 250
	`, s, s, s, s, clientID, clientID, st, st, st).Scan(&rows).Error
	return rows, err
}

func (r *repository) Clients(search string) ([]model.Row, error) {
	s := like(search)
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT
		  c.id, c.company_name, c.type, c.address, c.city, c.state, c.country,
		  c.phone, c.website, c.currency_symbol, c.created_date,
		  (SELECT COUNT(*) FROM pm_projects p WHERE p.deleted = FALSE AND p.client_id = c.id) AS project_count,
		  (SELECT COUNT(*) FROM pm_tickets tk WHERE tk.deleted = FALSE AND tk.client_id = c.id) AS ticket_count
		FROM pm_clients c
		WHERE c.deleted = FALSE
		  AND (?::text IS NULL OR LOWER(c.company_name) LIKE ?
		       OR LOWER(COALESCE(c.city, '')) LIKE ? OR LOWER(COALESCE(c.country, '')) LIKE ?)
		ORDER BY project_count DESC, c.company_name ASC
		LIMIT 500
	`, s, s, s, s).Scan(&rows).Error
	return rows, err
}

func (r *repository) Members(projectID *int64, search string) ([]model.Row, error) {
	s := like(search)
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT
		  m.id, m.user_id, m.project_id, m.is_leader,
		  p.title AS project_title, c.company_name AS client_name,
		  (SELECT COUNT(*) FROM pm_tasks t WHERE t.deleted = FALSE AND t.assigned_to = m.user_id) AS task_count
		FROM pm_project_members m
		LEFT JOIN pm_projects p ON p.id = m.project_id
		LEFT JOIN pm_clients c ON c.id = p.client_id
		WHERE m.deleted = FALSE
		  AND (?::bigint IS NULL OR m.project_id = ?)
		  AND (?::text IS NULL OR LOWER(COALESCE(p.title, '')) LIKE ? OR LOWER(COALESCE(c.company_name, '')) LIKE ?)
		ORDER BY m.is_leader DESC, m.project_id ASC
		LIMIT 500
	`, projectID, projectID, s, s, s).Scan(&rows).Error
	return rows, err
}

func (r *repository) Tasks(search string, projectID *int64, assignedTo string) ([]model.Row, error) {
	s := like(search)
	at := blankToNil(assignedTo)
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT t.*, s.title AS status_title, s.key_name AS status_key, s.color AS status_color
		FROM pm_tasks t
		LEFT JOIN pm_task_statuses s ON s.id = t.status_id
		WHERE t.deleted = FALSE
		  AND (?::text IS NULL OR LOWER(t.title) LIKE ? OR LOWER(COALESCE(t.description, '')) LIKE ?)
		  AND (?::bigint IS NULL OR t.project_id = ?)
		  AND (?::bigint IS NULL OR t.assigned_to = ?::bigint)
		ORDER BY s.sort_order ASC NULLS LAST, t.sort_order ASC, t.id DESC
	`, s, s, s, projectID, projectID, at, at).Scan(&rows).Error
	return rows, err
}

func (r *repository) taskRow(where string, args ...interface{}) (model.Row, error) {
	var rows []model.Row
	sql := fmt.Sprintf(`
		SELECT t.*, s.title AS status_title, s.key_name AS status_key, s.color AS status_color
		FROM pm_tasks t
		LEFT JOIN pm_task_statuses s ON s.id = t.status_id
		WHERE %s
	`, where)
	if err := r.DB.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("record not found")
	}
	return rows[0], nil
}

func (r *repository) TaskByID(id int64) (model.Row, error) {
	return r.taskRow("t.id = ? AND t.deleted = FALSE", id)
}

func (r *repository) ResolveStatusID(statusID *int64, statusKey string) (int64, error) {
	if statusID != nil {
		return *statusID, nil
	}
	key := statusKey
	if strings.TrimSpace(key) == "" {
		key = "to_do"
	}
	id, err := r.StatusIDByKey(key)
	if err != nil {
		return 0, err
	}
	if id == nil {
		return 0, fmt.Errorf("unknown task status: %s", key)
	}
	return *id, nil
}

func (r *repository) StatusIDByKey(key string) (*int64, error) {
	var rows []model.Row
	if err := r.DB.Raw(`SELECT id FROM pm_task_statuses WHERE key_name = ? AND deleted = FALSE LIMIT 1`, key).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	id, ok := toInt64(rows[0]["id"])
	if !ok {
		return nil, nil
	}
	return &id, nil
}

func (r *repository) StatusKeyByID(statusID int64) (string, error) {
	var rows []model.Row
	if err := r.DB.Raw(`SELECT key_name FROM pm_task_statuses WHERE id = ? AND deleted = FALSE`, statusID).Scan(&rows).Error; err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", errors.New("status not found")
	}
	key, _ := rows[0]["key_name"].(string)
	return key, nil
}

func (r *repository) InsertTask(fields map[string]interface{}) (int64, error) {
	columns, placeholders, values := buildInsert(fields)
	var id int64
	sql := fmt.Sprintf(`INSERT INTO pm_tasks (%s) VALUES (%s) RETURNING id`, columns, placeholders)
	if err := r.DB.Raw(sql, values...).Scan(&id).Error; err != nil {
		return 0, err
	}
	return id, nil
}

func (r *repository) UpdateTask(id int64, fields map[string]interface{}) error {
	set, values := buildUpdate(fields)
	values = append(values, id)
	sql := fmt.Sprintf(`UPDATE pm_tasks SET %s, updated_at = NOW() WHERE id = ? AND deleted = FALSE`, set)
	return r.DB.Exec(sql, values...).Error
}

func (r *repository) DeleteTask(id int64) error {
	return r.DB.Exec(`UPDATE pm_tasks SET deleted = TRUE, updated_at = NOW() WHERE id = ?`, id).Error
}

func (r *repository) Tickets(status, search string) ([]model.Row, error) {
	st := blankToNil(status)
	s := like(search)
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT *
		FROM pm_tickets
		WHERE deleted = FALSE
		  AND (?::text IS NULL OR status = ?)
		  AND (?::text IS NULL OR LOWER(title) LIKE ? OR LOWER(COALESCE(creator_name, '')) LIKE ?)
		ORDER BY last_activity_at DESC NULLS LAST, created_at DESC
	`, st, st, s, s, s).Scan(&rows).Error
	return rows, err
}

func (r *repository) TicketByID(id int64) (model.Row, error) {
	var rows []model.Row
	if err := r.DB.Raw(`SELECT * FROM pm_tickets WHERE id = ? AND deleted = FALSE`, id).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("record not found")
	}
	return rows[0], nil
}

func (r *repository) InsertTicket(fields map[string]interface{}) (int64, error) {
	columns, placeholders, values := buildInsert(fields)
	var id int64
	sql := fmt.Sprintf(`INSERT INTO pm_tickets (%s, last_activity_at) VALUES (%s, NOW()) RETURNING id`, columns, placeholders)
	if err := r.DB.Raw(sql, values...).Scan(&id).Error; err != nil {
		return 0, err
	}
	return id, nil
}

func (r *repository) TouchTicketActivity(id int64) error {
	return r.DB.Exec(`UPDATE pm_tickets SET last_activity_at = NOW(), updated_at = NOW() WHERE id = ?`, id).Error
}

func (r *repository) TicketComments(ticketID int64) ([]model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT * FROM pm_ticket_comments
		WHERE ticket_id = ? AND deleted = FALSE
		ORDER BY created_at ASC, id ASC
	`, ticketID).Scan(&rows).Error
	return rows, err
}

func (r *repository) InsertTicketComment(fields map[string]interface{}) (int64, error) {
	columns, placeholders, values := buildInsert(fields)
	var id int64
	sql := fmt.Sprintf(`INSERT INTO pm_ticket_comments (%s) VALUES (%s) RETURNING id`, columns, placeholders)
	if err := r.DB.Raw(sql, values...).Scan(&id).Error; err != nil {
		return 0, err
	}
	return id, nil
}

func (r *repository) TicketCommentByID(id int64) (model.Row, error) {
	var rows []model.Row
	if err := r.DB.Raw(`SELECT * FROM pm_ticket_comments WHERE id = ?`, id).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("record not found")
	}
	return rows[0], nil
}

func (r *repository) TicketTemplates() ([]model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`SELECT * FROM pm_ticket_templates WHERE deleted = FALSE ORDER BY title ASC, id ASC`).Scan(&rows).Error
	return rows, err
}

func (r *repository) ActivityLogs() ([]model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`SELECT * FROM pm_activity_logs WHERE deleted = FALSE ORDER BY created_at DESC, id DESC LIMIT 100`).Scan(&rows).Error
	return rows, err
}

func (r *repository) LogActivity(action, logType, title string, typeID int64, logFor string, logForID int64, changes *string, createdBy *string) error {
	if strings.TrimSpace(title) == "" {
		title = logType
	}
	return r.DB.Exec(`
		INSERT INTO pm_activity_logs (created_by, action, log_type, log_type_title, log_type_id, changes, log_for, log_for_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, createdBy, action, logType, title, typeID, changes, logFor, logForID).Error
}

// CrmProjects powers the Project List page. Owner is resolved from
// projects.cuser_id (the project's creator), not the free-text assigned_to
// column — matching how Assignee is resolved on the Tasks tab. Updated is the
// greatest of the project's own updated_at and its PM tasks' updated_at, so
// the column reflects the most recent activity on the project.
// taskStageJoinSQL aggregates each project's active (non-deleted) PM tasks
// into per-status counts in one grouped pass, joined once per project — this
// keeps the Stage computation below to a single query (no N+1 per-row task
// lookups) for both the list and detail queries that embed it.
const taskStageJoinSQL = `
		LEFT JOIN (
		  SELECT
		    crm_project_id,
		    COUNT(*) AS total_tasks,
		    COUNT(*) FILTER (WHERE status = 'blocked') AS blocked_tasks,
		    COUNT(*) FILTER (WHERE status = 'done') AS done_tasks,
		    COUNT(*) FILTER (WHERE status = 'in_review') AS review_tasks,
		    COUNT(*) FILTER (WHERE status = 'in_progress') AS progress_tasks
		  FROM pm_project_tasks
		  WHERE deleted = FALSE
		  GROUP BY crm_project_id
		) task_stage ON task_stage.crm_project_id = p.id`

// taskStageCaseSQL derives Stage from the joined task_stage counts above.
// Priority (highest first): Blocked > Completed > Review > Fieldwork > Planning.
// Completed only fires once ALL active tasks are done (and at least one exists);
// a project with zero active tasks is always Planning, regardless of any other
// signal. This is the single source of truth for Stage — never set manually.
const taskStageCaseSQL = `
		  CASE
		    WHEN COALESCE(task_stage.total_tasks, 0) = 0 THEN 'Planning'
		    WHEN task_stage.blocked_tasks > 0 THEN 'Blocked'
		    WHEN task_stage.done_tasks = task_stage.total_tasks THEN 'Completed'
		    WHEN task_stage.review_tasks > 0 THEN 'Review'
		    WHEN task_stage.progress_tasks > 0 THEN 'Fieldwork'
		    ELSE 'Planning'
		  END AS stage`

func (r *repository) CrmProjects(search string) ([]model.Row, error) {
	s := like(search)
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT DISTINCT
		  p.id AS crm_project_id,
		  p.id,
		  COALESCE(l.project_name, c.company_name) AS title,
		  p.project_date AS start_date,
		  p.status,
		  p.assigned_to AS owner_name,
		  c.company_name AS client_name,
		  l.company_id,
		  l.assigned_user_id,
		  TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) AS assigned_user_name,
		  NULLIF(TRIM(COALESCE(cu.first_name, '') || ' ' || COALESCE(cu.last_name, '')), '') AS owner,
		  pm.pic_user_id,
		  NULLIF(TRIM(COALESCE(pu.first_name, '') || ' ' || COALESCE(pu.last_name, '')), '') AS pic,
		  `+taskStageCaseSQL+`,
		  GREATEST(
		    p.updated_at,
		    (SELECT MAX(t.updated_at) FROM pm_project_tasks t WHERE t.crm_project_id = p.id AND t.deleted = FALSE)
		  ) AS updated_at
		FROM projects p
		JOIN leads l ON l.id = p.lead_id
		JOIN companies c ON c.id = l.company_id
		JOIN project_services ps ON ps.project_id = p.id
		JOIN services s ON s.id = ps.service_id
		LEFT JOIN users u ON u.id = l.assigned_user_id
		LEFT JOIN users cu ON cu.id = p.cuser_id
		LEFT JOIN pm_projects pm ON pm.crm_project_id = p.id AND pm.deleted = FALSE
		LEFT JOIN users pu ON pu.id = pm.pic_user_id
		`+taskStageJoinSQL+`
		WHERE s.name IN ('Advisory', 'Audit Program')
		  AND (?::text IS NULL
		       OR LOWER(COALESCE(l.project_name, c.company_name)) LIKE ?
		       OR LOWER(c.company_name) LIKE ?)
		ORDER BY p.project_date DESC NULLS LAST
		LIMIT 200
	`, s, s, s).Scan(&rows).Error
	return rows, err
}

func (r *repository) CrmProjectByID(id int64) (model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT DISTINCT
		  p.id AS crm_project_id,
		  p.id,
		  COALESCE(l.project_name, c.company_name) AS title,
		  p.project_date AS start_date,
		  p.valid_until AS end_date,
		  p.project_date AS project_start_date,
		  p.valid_until AS project_end_date,
		  p.status,
		  c.company_name AS client_name,
		  l.company_id,
		  l.project_description AS scope,
		  p.assigned_to AS owner_name,
		  p.assigned_to AS owner,
		  pm.pic_user_id,
		  NULLIF(TRIM(COALESCE(pu.first_name, '') || ' ' || COALESCE(pu.last_name, '')), '') AS pic,
		  `+taskStageCaseSQL+`,
		  p.updated_at
		FROM projects p
		JOIN leads l ON l.id = p.lead_id
		JOIN companies c ON c.id = l.company_id
		LEFT JOIN pm_projects pm ON pm.crm_project_id = p.id AND pm.deleted = FALSE
		LEFT JOIN users pu ON pu.id = pm.pic_user_id
		`+taskStageJoinSQL+`
		WHERE p.id = ?
		LIMIT 1
	`, id).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("project not found: %d", id)
	}
	return rows[0], nil
}

// UpdateCrmProject updates columns on the real CRM `projects` table (owner,
// project dates, etc.) — used by the PM Overview tab's Save action. Fields
// map keys must already be the target `projects` column names.
func (r *repository) UpdateCrmProject(id int64, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	set, values := buildUpdate(fields)
	values = append(values, id)
	sql := fmt.Sprintf(`UPDATE projects SET %s WHERE id = ?`, set)
	return r.DB.Exec(sql, values...).Error
}

// UpsertPmProjectPic sets the PM-side "PIC" (person in charge) for a
// CRM-linked project. pm_projects has no row per CRM project by default
// (it's the legacy Library module's table, keyed by its own `legacy_id`),
// so this creates a minimal shadow row on first use — same pattern as the
// Java backend's ensurePmProject — then just updates pic_user_id on it.
func (r *repository) UpsertPmProjectPic(crmProjectID int64, picUserID *int64) error {
	res := r.DB.Exec(`
		UPDATE pm_projects SET pic_user_id = ?, updated_at = NOW()
		WHERE crm_project_id = ? AND deleted = FALSE
	`, picUserID, crmProjectID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		return nil
	}

	var title string
	if err := r.DB.Raw(`SELECT COALESCE(l.project_name, c.company_name) FROM projects p
		JOIN leads l ON l.id = p.lead_id JOIN companies c ON c.id = l.company_id
		WHERE p.id = ?`, crmProjectID).Scan(&title).Error; err != nil {
		return err
	}
	if title == "" {
		title = fmt.Sprintf("CRM Project #%d", crmProjectID)
	}

	return r.DB.Exec(`
		INSERT INTO pm_projects (legacy_id, title, client_id, created_by, status_id, crm_project_id, pic_user_id)
		VALUES ((SELECT COALESCE(MAX(legacy_id), 0) + 1 FROM pm_projects), ?, 0, 0, 0, ?, ?)
	`, title, crmProjectID, picUserID).Error
}

// ProjectTasksByCrmProject powers both the Tasks tab list "Assignee" column
// and the task detail popup's "Details > Assignee" field. When a task has no
// explicit assignee (t.assigned_to), it falls back to the CRM project's
// owner (projects.cuser_id) — the same source used for the Project List
// "Owner" column — rather than showing a blank/Unassigned name.
// Postgres `uuid` columns (t.id, t.parent_task_id) scan into the generic
// map[string]interface{} Row as raw []byte, which encoding/json silently
// base64-encodes — corrupting the id for every later PATCH/DELETE/parent
// link. Cast every uuid column to ::text so it round-trips as plain text.
func (r *repository) ProjectTasksByCrmProject(crmProjectID int64) ([]model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT t.id::text AS id, t.title, t.description, t.status,
		       t.start_date, t.deadline, t.parent_task_id::text AS parent_task_id, t.crm_project_id,
		       t.assigned_to, t.labels, t.points, t.priority_id, t.sort_order, t.progress_pct,
		       t.actual_start_date, t.actual_finish_date, t.collaborators,
		       t.created_at, t.updated_at,
		       s.title AS status_title, s.key_name AS status_key, s.color AS status_color,
		       COALESCE(
		         NULLIF(TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')), ''),
		         NULLIF(TRIM(COALESCE(cu.first_name, '') || ' ' || COALESCE(cu.last_name, '')), ''),
		         ''
		       ) AS assignee_name
		FROM pm_project_tasks t
		LEFT JOIN pm_task_statuses s ON s.id = t.status_id
		LEFT JOIN users u ON u.id = t.assigned_to
		LEFT JOIN projects p ON p.id = t.crm_project_id
		LEFT JOIN users cu ON cu.id = p.cuser_id
		WHERE t.deleted = FALSE AND t.crm_project_id = ?
		ORDER BY t.parent_task_id ASC NULLS FIRST, t.sort_order ASC, t.id ASC
	`, crmProjectID).Scan(&rows).Error
	return rows, err
}

func (r *repository) InsertProjectTask(fields map[string]interface{}) error {
	columns, placeholders, values := buildInsert(fields)
	sql := fmt.Sprintf(`INSERT INTO pm_project_tasks (%s) VALUES (%s)`, columns, placeholders)
	return r.DB.Exec(sql, values...).Error
}

func (r *repository) UpdateProjectTask(id string, fields map[string]interface{}) error {
	set, values := buildUpdate(fields)
	values = append(values, id)
	sql := fmt.Sprintf(`UPDATE pm_project_tasks SET %s, updated_at = NOW() WHERE id = ? AND deleted = FALSE`, set)
	return r.DB.Exec(sql, values...).Error
}

func (r *repository) ProjectTaskByID(id string) (model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT t.id::text AS id, t.title, t.description, t.status,
		       t.start_date, t.deadline, t.parent_task_id::text AS parent_task_id, t.crm_project_id,
		       t.assigned_to, t.labels, t.points, t.priority_id, t.sort_order, t.progress_pct,
		       t.actual_start_date, t.actual_finish_date, t.collaborators,
		       t.created_at, t.updated_at,
		       s.title AS status_title, s.key_name AS status_key, s.color AS status_color,
		       NULLIF(TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')), '') AS assignee_name
		FROM pm_project_tasks t
		LEFT JOIN pm_task_statuses s ON s.id = t.status_id
		LEFT JOIN users u ON u.id = t.assigned_to
		WHERE t.id = ? AND t.deleted = FALSE
	`, id).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("record not found")
	}
	return rows[0], nil
}

func (r *repository) MoveProjectTaskByKey(id, statusKey string) error {
	return r.DB.Exec(`
		UPDATE pm_project_tasks SET
		  status = ?,
		  status_id = (SELECT id FROM pm_task_statuses WHERE key_name = ? AND deleted = FALSE),
		  status_changed_at = NOW(),
		  updated_at = NOW()
		WHERE id = ? AND deleted = FALSE
	`, statusKey, statusKey, id).Error
}

func (r *repository) DeleteProjectTask(id string) error {
	return r.DB.Exec(`UPDATE pm_project_tasks SET deleted = TRUE, updated_at = NOW() WHERE id = ?`, id).Error
}

// InsertTaskActivityLog appends one audit-trail row. Never deleted alongside
// its task (DeleteProjectTask above is a soft delete anyway, and this table
// has no FK to pm_project_tasks) — logging happens before the delete so the
// "deleted" entry is always the last thing recorded for that task.
func (r *repository) InsertTaskActivityLog(fields map[string]interface{}) error {
	columns, placeholders, values := buildInsert(fields)
	sql := fmt.Sprintf(`INSERT INTO pm_task_activity_logs (%s) VALUES (%s)`, columns, placeholders)
	return r.DB.Exec(sql, values...).Error
}

// TaskActivityLogs returns a task's activity, newest first — matches the
// project-level Activity tab's convention (ActivityByCrmProject also orders
// DESC) so the two feel consistent.
func (r *repository) TaskActivityLogs(taskID string) ([]model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT id, task_id::text AS task_id, project_id, actor_user_id, actor_name,
		       action, field_name, old_value, new_value, description, created_at
		FROM pm_task_activity_logs
		WHERE task_id = ?
		ORDER BY created_at DESC, id DESC
	`, taskID).Scan(&rows).Error
	return rows, err
}

func (r *repository) ResolveUserName(userID int64) (string, error) {
	var name string
	err := r.DB.Raw(`
		SELECT NULLIF(TRIM(COALESCE(first_name, '') || ' ' || COALESCE(last_name, '')), '')
		FROM users WHERE id = ?
	`, userID).Scan(&name).Error
	return name, err
}

// TeamByCrmProject lists this project's explicitly-managed team
// (pm_crm_project_members — added/removed via the Team tab CRUD), each row
// carrying task counts scoped to this project. Every row's crm_project_id
// filter guarantees a project only ever sees/edits its own members.
func (r *repository) TeamByCrmProject(crmProjectID int64) ([]model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT m.id::text AS id, m.crm_project_id, m.user_id, m.role, m.is_leader,
		       TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) AS name,
		       u.email,
		       COUNT(t.id) FILTER (WHERE t.status <> 'done') AS open_tasks,
		       COUNT(t.id) AS total_tasks
		FROM pm_crm_project_members m
		JOIN users u ON u.id = m.user_id
		LEFT JOIN pm_project_tasks t
		  ON t.assigned_to = m.user_id AND t.crm_project_id = m.crm_project_id AND t.deleted = FALSE
		WHERE m.deleted = FALSE AND m.crm_project_id = ?
		GROUP BY m.id, m.crm_project_id, m.user_id, m.role, m.is_leader, name, u.email
		ORDER BY m.is_leader DESC, name ASC
	`, crmProjectID).Scan(&rows).Error
	return rows, err
}

// CrmProjectMemberByID fetches a single team member row scoped to its
// project — used to return the created/updated record after a CRUD write.
func (r *repository) CrmProjectMemberByID(id string, crmProjectID int64) (model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT m.id::text AS id, m.crm_project_id, m.user_id, m.role, m.is_leader, m.created_at, m.updated_at,
		       TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) AS name,
		       u.email
		FROM pm_crm_project_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.id = ? AND m.crm_project_id = ? AND m.deleted = FALSE
	`, id, crmProjectID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("record not found")
	}
	return rows[0], nil
}

func (r *repository) InsertCrmProjectMember(fields map[string]interface{}) error {
	columns, placeholders, values := buildInsert(fields)
	sql := fmt.Sprintf(`INSERT INTO pm_crm_project_members (%s) VALUES (%s)`, columns, placeholders)
	return r.DB.Exec(sql, values...).Error
}

// UpdateCrmProjectMember and DeleteCrmProjectMember both filter on
// crm_project_id (not just id) so a member can only ever be edited/removed
// through its own project.
func (r *repository) UpdateCrmProjectMember(id string, crmProjectID int64, fields map[string]interface{}) error {
	set, values := buildUpdate(fields)
	values = append(values, id, crmProjectID)
	sql := fmt.Sprintf(`UPDATE pm_crm_project_members SET %s, updated_at = NOW() WHERE id = ? AND crm_project_id = ? AND deleted = FALSE`, set)
	return r.DB.Exec(sql, values...).Error
}

func (r *repository) DeleteCrmProjectMember(id string, crmProjectID int64) error {
	return r.DB.Exec(`
		UPDATE pm_crm_project_members SET deleted = TRUE, updated_at = NOW()
		WHERE id = ? AND crm_project_id = ?
	`, id, crmProjectID).Error
}

// ActivityByCrmProject builds a real timeline from task creation and status
// changes — there's no dedicated activity-log table for CRM-linked PM tasks
// yet, so this is derived from pm_project_tasks timestamps directly.
func (r *repository) ActivityByCrmProject(crmProjectID int64) ([]model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT * FROM (
		  SELECT t.id::text AS id, 'created' AS action, t.title, t.created_at AS occurred_at
		  FROM pm_project_tasks t
		  WHERE t.crm_project_id = ? AND t.deleted = FALSE
		  UNION ALL
		  SELECT t.id::text AS id, 'status_changed' AS action, (t.title || ' -> ' || t.status) AS title, t.status_changed_at AS occurred_at
		  FROM pm_project_tasks t
		  WHERE t.crm_project_id = ? AND t.deleted = FALSE AND t.status_changed_at IS NOT NULL
		) events
		ORDER BY occurred_at DESC
		LIMIT 50
	`, crmProjectID, crmProjectID).Scan(&rows).Error
	return rows, err
}

// GanttMembers lists active internal users who belong to the "IT Audit"
// department — this is the PM module's "Member" picker (Gantt assignee +
// Add Team Member), scoped to IT Audit since PM baru is the IT Audit Project
// Management workspace. Previously also required the user to have a lead
// assigned (a leftover from an older, unrelated filter), which wrongly
// excluded real IT Audit staff (auditors) who never get leads assigned —
// dropped that condition so department membership alone determines the list.
func (r *repository) GanttMembers() ([]model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT DISTINCT u.id, TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) AS name,
		       u.employee_id AS code, u.email
		FROM users u
		INNER JOIN departements d ON d.id = u.departement_id
		WHERE u.dtype = 'INTERNAL_USER' AND (u.status IS NULL OR u.status <> '0')
		  AND d.name = 'IT Audit'
		ORDER BY name ASC
		LIMIT 500
	`).Scan(&rows).Error
	return rows, err
}

func (r *repository) Timesheets(search string, userID, crmProjectID *int64, status string, from, to *string) ([]model.Row, error) {
	s := like(search)
	st := blankToNil(status)
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT
		  ts.id, ts.crm_project_id, ts.task_id, ts.user_id, ts.work_date, ts.hours,
		  ts.description, ts.status, ts.approved_by, ts.approved_at, ts.created_at, ts.updated_at,
		  COALESCE(l.project_name, c.company_name) AS project_title,
		  TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) AS member_name,
		  t.title AS task_title
		FROM pm_timesheets ts
		LEFT JOIN projects p ON p.id = ts.crm_project_id
		LEFT JOIN leads l ON l.id = p.lead_id
		LEFT JOIN companies c ON c.id = l.company_id
		LEFT JOIN users u ON u.id = ts.user_id
		LEFT JOIN pm_project_tasks t ON t.id = ts.task_id
		WHERE ts.deleted = FALSE
		  AND (?::bigint IS NULL OR ts.user_id = ?)
		  AND (?::bigint IS NULL OR ts.crm_project_id = ?)
		  AND (?::text IS NULL OR ts.status = ?)
		  AND (?::date IS NULL OR ts.work_date >= ?::date)
		  AND (?::date IS NULL OR ts.work_date <= ?::date)
		  AND (?::text IS NULL
		       OR LOWER(COALESCE(l.project_name, c.company_name, '')) LIKE ?
		       OR LOWER(COALESCE(t.title, '')) LIKE ?
		       OR LOWER(TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, ''))) LIKE ?)
		ORDER BY ts.work_date DESC, ts.id DESC
		LIMIT 500
	`, userID, userID, crmProjectID, crmProjectID, st, st, from, from, to, to, s, s, s, s).Scan(&rows).Error
	return rows, err
}

func (r *repository) TimesheetSummary(userID *int64) (model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT
		  COALESCE(SUM(hours) FILTER (WHERE work_date = CURRENT_DATE), 0) AS hours_today,
		  COALESCE(SUM(hours) FILTER (WHERE work_date >= date_trunc('week', CURRENT_DATE)::date), 0) AS weekly_hours,
		  COUNT(*) FILTER (WHERE status = 'pending') AS pending_count,
		  COUNT(*) FILTER (WHERE status = 'approved') AS approved_count
		FROM pm_timesheets
		WHERE deleted = FALSE
		  AND (?::bigint IS NULL OR user_id = ?)
	`, userID, userID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return model.Row{}, nil
	}
	return rows[0], nil
}

func (r *repository) TimesheetByID(id int64) (model.Row, error) {
	var rows []model.Row
	err := r.DB.Raw(`
		SELECT
		  ts.id, ts.crm_project_id, ts.task_id, ts.user_id, ts.work_date, ts.hours,
		  ts.description, ts.status, ts.approved_by, ts.approved_at, ts.created_at, ts.updated_at,
		  COALESCE(l.project_name, c.company_name) AS project_title,
		  TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) AS member_name,
		  t.title AS task_title
		FROM pm_timesheets ts
		LEFT JOIN projects p ON p.id = ts.crm_project_id
		LEFT JOIN leads l ON l.id = p.lead_id
		LEFT JOIN companies c ON c.id = l.company_id
		LEFT JOIN users u ON u.id = ts.user_id
		LEFT JOIN pm_project_tasks t ON t.id = ts.task_id
		WHERE ts.id = ? AND ts.deleted = FALSE
	`, id).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("record not found")
	}
	return rows[0], nil
}

func (r *repository) InsertTimesheet(fields map[string]interface{}) (int64, error) {
	columns, placeholders, values := buildInsert(fields)
	var id int64
	sql := fmt.Sprintf(`INSERT INTO pm_timesheets (%s) VALUES (%s) RETURNING id`, columns, placeholders)
	if err := r.DB.Raw(sql, values...).Scan(&id).Error; err != nil {
		return 0, err
	}
	return id, nil
}

func (r *repository) UpdateTimesheetStatus(id int64, status string, approvedBy *int64) error {
	return r.DB.Exec(`
		UPDATE pm_timesheets SET
		  status = ?,
		  approved_by = ?,
		  approved_at = CASE WHEN ? = 'pending' THEN NULL ELSE NOW() END,
		  updated_at = NOW()
		WHERE id = ? AND deleted = FALSE
	`, status, approvedBy, status, id).Error
}

func (r *repository) DeleteTimesheet(id int64) error {
	return r.DB.Exec(`UPDATE pm_timesheets SET deleted = TRUE, updated_at = NOW() WHERE id = ?`, id).Error
}

func buildInsert(fields map[string]interface{}) (columns string, placeholders string, values []interface{}) {
	cols := make([]string, 0, len(fields))
	phs := make([]string, 0, len(fields))
	vals := make([]interface{}, 0, len(fields))
	for k, v := range fields {
		cols = append(cols, k)
		phs = append(phs, "?")
		vals = append(vals, v)
	}
	return strings.Join(cols, ", "), strings.Join(phs, ", "), vals
}

func buildUpdate(fields map[string]interface{}) (set string, values []interface{}) {
	parts := make([]string, 0, len(fields))
	vals := make([]interface{}, 0, len(fields))
	for k, v := range fields {
		parts = append(parts, k+" = ?")
		vals = append(vals, v)
	}
	return strings.Join(parts, ", "), vals
}

func toInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int32:
		return int64(n), true
	case int:
		return int64(n), true
	}
	return 0, false
}
