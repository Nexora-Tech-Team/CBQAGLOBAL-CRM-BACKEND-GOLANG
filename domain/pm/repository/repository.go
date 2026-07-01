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
		  AND (?::uuid IS NULL OR t.assigned_to = ?::uuid)
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
