package service

import (
	"fmt"
	"strings"
	"time"

	"erp-cbqa-global/domain/pm/model"
	"erp-cbqa-global/domain/pm/repository"
)

type ServiceInterface interface {
	Dashboard() (model.Row, error)
	Statuses() ([]model.Row, error)
	Kanban() (model.Row, error)
	Projects(search string, clientID *int64, status string) ([]model.Row, error)
	Clients(search string) ([]model.Row, error)
	Members(projectID *int64, search string) ([]model.Row, error)

	Tasks(search string, projectID *int64, assignedTo string) ([]model.Row, error)
	CreateTask(req model.TaskRequest, userID string) (model.Row, error)
	UpdateTask(id int64, req model.TaskRequest, userID string) (model.Row, error)
	MoveTaskStatus(id, statusID int64, userID string) (model.Row, error)
	MoveTaskByKey(id int64, statusKey string, userID string) (model.Row, error)
	DeleteTask(id int64, userID string) error

	Tickets(status, search string) ([]model.Row, error)
	CreateTicket(req model.TicketRequest, userID string) (model.Row, error)
	TicketComments(ticketID int64) ([]model.Row, error)
	AddTicketComment(ticketID int64, req model.TicketCommentRequest, userID string) (model.Row, error)
	TicketTemplates() ([]model.Row, error)
	ActivityLogs() ([]model.Row, error)
}

type pmService struct {
	Repository repository.RepositoryInterface
}

func Service(repo repository.RepositoryInterface) ServiceInterface {
	return &pmService{Repository: repo}
}

func (s *pmService) Dashboard() (model.Row, error) {
	totals, err := s.Repository.DashboardTotals()
	if err != nil {
		return nil, err
	}
	byStatus, err := s.Repository.DashboardProjectsByStatus()
	if err != nil {
		return nil, err
	}
	topClients, err := s.Repository.DashboardTopClients()
	if err != nil {
		return nil, err
	}
	recentProjects, err := s.Repository.DashboardRecentProjects()
	if err != nil {
		return nil, err
	}
	result := model.Row{}
	for k, v := range totals {
		result[k] = v
	}
	result["projects_by_status"] = byStatus
	result["top_clients"] = topClients
	result["recent_projects"] = recentProjects
	return result, nil
}

func (s *pmService) Statuses() ([]model.Row, error) {
	return s.Repository.Statuses()
}

func (s *pmService) Kanban() (model.Row, error) {
	statuses, err := s.Repository.Statuses()
	if err != nil {
		return nil, err
	}
	columns := make([]model.Row, 0, len(statuses))
	for _, status := range statuses {
		if toBool(status["hide_from_kanban"]) {
			continue
		}
		id, _ := toInt64(status["id"])
		tasks, err := s.Repository.TasksByStatus(id)
		if err != nil {
			return nil, err
		}
		column := model.Row{}
		for k, v := range status {
			column[k] = v
		}
		column["tasks"] = tasks
		columns = append(columns, column)
	}
	return model.Row{"columns": columns}, nil
}

func (s *pmService) Projects(search string, clientID *int64, status string) ([]model.Row, error) {
	return s.Repository.Projects(search, clientID, status)
}

func (s *pmService) Clients(search string) ([]model.Row, error) {
	return s.Repository.Clients(search)
}

func (s *pmService) Members(projectID *int64, search string) ([]model.Row, error) {
	return s.Repository.Members(projectID, search)
}

func (s *pmService) Tasks(search string, projectID *int64, assignedTo string) ([]model.Row, error) {
	return s.Repository.Tasks(search, projectID, assignedTo)
}

func (s *pmService) taskFields(req model.TaskRequest, statusID int64, statusKey string) map[string]interface{} {
	var parentTaskID interface{}
	if req.ParentTaskID != nil && *req.ParentTaskID > 0 {
		parentTaskID = *req.ParentTaskID
	}
	return map[string]interface{}{
		"title":          req.Title,
		"description":    req.Description,
		"project_id":     req.ProjectID,
		"assigned_to":    req.AssignedTo,
		"deadline":       parseDateTime(req.Deadline),
		"labels":         req.Labels,
		"points":         intOrDefault(req.Points, 1),
		"status":         statusKey,
		"status_id":      statusID,
		"priority_id":    int64OrDefault(req.PriorityID, 0),
		"start_date":     parseDateTime(req.StartDate),
		"parent_task_id": parentTaskID,
		"collaborators":  strOrDefault(req.Collaborators, ""),
		"sort_order":     intOrDefault(req.SortOrder, 0),
		"context":        strOrDefault(req.Context, "general"),
	}
}

func (s *pmService) CreateTask(req model.TaskRequest, userID string) (model.Row, error) {
	statusKeyInput := strOrDefault(req.Status, "to_do")
	statusID, err := s.Repository.ResolveStatusID(req.StatusID, statusKeyInput)
	if err != nil {
		return nil, err
	}
	statusKey, err := s.Repository.StatusKeyByID(statusID)
	if err != nil {
		return nil, err
	}
	fields := s.taskFields(req, statusID, statusKey)
	fields["created_by"] = userID
	id, err := s.Repository.InsertTask(fields)
	if err != nil {
		return nil, err
	}
	changesNil := (*string)(nil)
	_ = s.Repository.LogActivity("created", "task", req.Title, id, "project", int64OrDefault(req.ProjectID, 0), changesNil, &userID)
	return s.Repository.TaskByID(id)
}

func (s *pmService) UpdateTask(id int64, req model.TaskRequest, userID string) (model.Row, error) {
	before, err := s.Repository.TaskByID(id)
	if err != nil {
		return nil, err
	}
	statusKeyInput := strOrDefault(req.Status, "to_do")
	statusID, err := s.Repository.ResolveStatusID(req.StatusID, statusKeyInput)
	if err != nil {
		return nil, err
	}
	statusKey, err := s.Repository.StatusKeyByID(statusID)
	if err != nil {
		return nil, err
	}
	fields := s.taskFields(req, statusID, statusKey)
	if err := s.Repository.UpdateTask(id, fields); err != nil {
		return nil, err
	}
	after, err := s.Repository.TaskByID(id)
	if err != nil {
		return nil, err
	}
	changes := fmt.Sprintf("before=%v; after=%v", before, after)
	_ = s.Repository.LogActivity("updated", "task", req.Title, id, "project", int64OrDefault(req.ProjectID, 0), &changes, &userID)
	return after, nil
}

func (s *pmService) MoveTaskStatus(id, statusID int64, userID string) (model.Row, error) {
	before, err := s.Repository.TaskByID(id)
	if err != nil {
		return nil, err
	}
	statusKey, err := s.Repository.StatusKeyByID(statusID)
	if err != nil {
		return nil, err
	}
	if err := s.Repository.UpdateTask(id, map[string]interface{}{
		"status_id":         statusID,
		"status":            statusKey,
		"status_changed_at": time.Now().UTC(),
	}); err != nil {
		return nil, err
	}
	after, err := s.Repository.TaskByID(id)
	if err != nil {
		return nil, err
	}
	changes := fmt.Sprintf("before=%v; after=%v", before, after)
	projectID, _ := toInt64(after["project_id"])
	_ = s.Repository.LogActivity("updated", "task", str(after["title"]), id, "project", projectID, &changes, &userID)
	return after, nil
}

func (s *pmService) MoveTaskByKey(id int64, statusKey string, userID string) (model.Row, error) {
	statusID, err := s.Repository.StatusIDByKey(statusKey)
	if err != nil {
		return nil, err
	}
	fields := map[string]interface{}{
		"status":            statusKey,
		"status_changed_at": time.Now().UTC(),
	}
	if statusID != nil {
		fields["status_id"] = *statusID
	}
	if err := s.Repository.UpdateTask(id, fields); err != nil {
		return nil, err
	}
	after, err := s.Repository.TaskByID(id)
	if err != nil {
		return nil, err
	}
	projectID, _ := toInt64(after["project_id"])
	_ = s.Repository.LogActivity("moved", "task", str(after["title"]), id, "project", projectID, nil, &userID)
	return after, nil
}

func (s *pmService) DeleteTask(id int64, userID string) error {
	before, err := s.Repository.TaskByID(id)
	if err != nil {
		return err
	}
	if err := s.Repository.DeleteTask(id); err != nil {
		return err
	}
	projectID, _ := toInt64(before["project_id"])
	return s.Repository.LogActivity("deleted", "task", str(before["title"]), id, "project", projectID, nil, &userID)
}

func (s *pmService) Tickets(status, search string) ([]model.Row, error) {
	return s.Repository.Tickets(status, search)
}

func (s *pmService) CreateTicket(req model.TicketRequest, userID string) (model.Row, error) {
	fields := map[string]interface{}{
		"title":                  req.Title,
		"client_id":              int64OrDefault(req.ClientID, 0),
		"project_id":             int64OrDefault(req.ProjectID, 0),
		"ticket_type_id":         int64OrDefault(req.TicketTypeID, 0),
		"created_by":             userID,
		"requested_by":           req.RequestedBy,
		"status":                 strOrDefault(req.Status, "new"),
		"assigned_to":            req.AssignedTo,
		"creator_name":           strOrDefault(req.CreatorName, ""),
		"creator_email":          strOrDefault(req.CreatorEmail, ""),
		"labels":                 req.Labels,
		"task_id":                int64OrDefault(req.TaskID, 0),
		"cc_contacts_and_emails": req.CcContactsAndEmails,
	}
	id, err := s.Repository.InsertTicket(fields)
	if err != nil {
		return nil, err
	}
	_ = s.Repository.LogActivity("created", "ticket", req.Title, id, "project", int64OrDefault(req.ProjectID, 0), nil, &userID)
	return s.Repository.TicketByID(id)
}

func (s *pmService) TicketComments(ticketID int64) ([]model.Row, error) {
	return s.Repository.TicketComments(ticketID)
}

func (s *pmService) AddTicketComment(ticketID int64, req model.TicketCommentRequest, userID string) (model.Row, error) {
	fields := map[string]interface{}{
		"ticket_id":   ticketID,
		"description": req.Description,
		"created_by":  userID,
		"files":       req.Files,
		"is_note":     req.IsNote,
	}
	id, err := s.Repository.InsertTicketComment(fields)
	if err != nil {
		return nil, err
	}
	if err := s.Repository.TouchTicketActivity(ticketID); err != nil {
		return nil, err
	}
	ticket, err := s.Repository.TicketByID(ticketID)
	if err != nil {
		return nil, err
	}
	changes := "comment added"
	_ = s.Repository.LogActivity("updated", "ticket", str(ticket["title"]), ticketID, "ticket_comment", id, &changes, &userID)
	return s.Repository.TicketCommentByID(id)
}

func (s *pmService) TicketTemplates() ([]model.Row, error) {
	return s.Repository.TicketTemplates()
}

func (s *pmService) ActivityLogs() ([]model.Row, error) {
	return s.Repository.ActivityLogs()
}

func strOrDefault(v *string, fallback string) string {
	if v == nil || strings.TrimSpace(*v) == "" {
		return fallback
	}
	return *v
}

func intOrDefault(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
}

func int64OrDefault(v *int64, fallback int64) int64 {
	if v == nil {
		return fallback
	}
	return *v
}

func str(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func toBool(v interface{}) bool {
	b, ok := v.(bool)
	return ok && b
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

func parseDateTime(v *string) *time.Time {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	value := strings.TrimSpace(*v)
	layouts := []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return &t
		}
	}
	return nil
}
