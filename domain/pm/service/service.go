package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"erp-cbqa-global/domain/pm/model"
	"erp-cbqa-global/domain/pm/repository"
)

type ServiceInterface interface {
	Dashboard(crmProjectID, memberID *int64) (model.Row, error)
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

	CrmProjects(search string) ([]model.Row, error)
	CrmProjectDetail(id int64) (model.Row, error)
	UpdateCrmProjectOverview(id int64, body map[string]interface{}) (model.Row, error)
	TasksByCrmProject(id int64) ([]model.Row, error)
	CreateTaskForCrmProject(crmProjectID int64, body map[string]interface{}, userID string) (model.Row, error)
	UpdateProjectTask(id string, body map[string]interface{}) (model.Row, error)
	MoveProjectTaskByKey(id, statusKey string, actorUserIDRaw interface{}) (model.Row, error)
	DeleteProjectTask(id string, actorUserIDRaw interface{}) error
	TaskActivityLogs(taskID string) ([]model.Row, error)
	GanttMembers() ([]model.Row, error)

	CrmProjectMembers(crmProjectID int64) ([]model.Row, error)
	AddCrmProjectMember(crmProjectID int64, body map[string]interface{}, userID string) (model.Row, error)
	UpdateCrmProjectMember(crmProjectID int64, memberID string, body map[string]interface{}) (model.Row, error)
	DeleteCrmProjectMember(crmProjectID int64, memberID string) error

	Timesheets(search string, userID, crmProjectID *int64, status, period string) (model.Row, error)
	CreateTimesheet(req model.TimesheetRequest) (model.Row, error)
	UpdateTimesheetStatus(id int64, status string, approvedBy *int64) (model.Row, error)
	DeleteTimesheet(id int64) error
}

type pmService struct {
	Repository repository.RepositoryInterface
}

func Service(repo repository.RepositoryInterface) ServiceInterface {
	return &pmService{Repository: repo}
}

func (s *pmService) Dashboard(crmProjectID, memberID *int64) (model.Row, error) {
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
	taskDistribution, err := s.Repository.DashboardTaskDistribution(crmProjectID, memberID)
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
	result["taskDistribution"] = taskDistribution
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
	fields["created_by"] = nilIfEmpty(userID)
	id, err := s.Repository.InsertTask(fields)
	if err != nil {
		return nil, err
	}
	changesNil := (*string)(nil)
	_ = s.Repository.LogActivity("created", "task", req.Title, id, "project", int64OrDefault(req.ProjectID, 0), changesNil, strPtrOrNil(userID))
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
	_ = s.Repository.LogActivity("updated", "task", req.Title, id, "project", int64OrDefault(req.ProjectID, 0), &changes, strPtrOrNil(userID))
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
	_ = s.Repository.LogActivity("updated", "task", str(after["title"]), id, "project", projectID, &changes, strPtrOrNil(userID))
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
	_ = s.Repository.LogActivity("moved", "task", str(after["title"]), id, "project", projectID, nil, strPtrOrNil(userID))
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
	return s.Repository.LogActivity("deleted", "task", str(before["title"]), id, "project", projectID, nil, strPtrOrNil(userID))
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
		"created_by":             nilIfEmpty(userID),
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
	_ = s.Repository.LogActivity("created", "ticket", req.Title, id, "project", int64OrDefault(req.ProjectID, 0), nil, strPtrOrNil(userID))
	return s.Repository.TicketByID(id)
}

func (s *pmService) TicketComments(ticketID int64) ([]model.Row, error) {
	return s.Repository.TicketComments(ticketID)
}

func (s *pmService) AddTicketComment(ticketID int64, req model.TicketCommentRequest, userID string) (model.Row, error) {
	fields := map[string]interface{}{
		"ticket_id":   ticketID,
		"description": req.Description,
		"created_by":  nilIfEmpty(userID),
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
	_ = s.Repository.LogActivity("updated", "ticket", str(ticket["title"]), ticketID, "ticket_comment", id, &changes, strPtrOrNil(userID))
	return s.Repository.TicketCommentByID(id)
}

func (s *pmService) TicketTemplates() ([]model.Row, error) {
	return s.Repository.TicketTemplates()
}

func (s *pmService) ActivityLogs() ([]model.Row, error) {
	return s.Repository.ActivityLogs()
}

func (s *pmService) CrmProjects(search string) ([]model.Row, error) {
	return s.Repository.CrmProjects(search)
}

// UpdateCrmProjectOverview saves the PM Overview tab's editable fields:
// owner + project dates live on the CRM `projects` table, while picUserId
// (the PM-side "PIC", distinct from owner) lives on `pm_projects`. Returns
// the same shape as CrmProjectDetail so the frontend can swap in the fresh
// data straight from the PUT response.
func (s *pmService) UpdateCrmProjectOverview(id int64, body map[string]interface{}) (model.Row, error) {
	fields := map[string]interface{}{}
	if _, ok := body["owner"]; ok {
		fields["assigned_to"] = strPtrFromAny(body["owner"])
	}
	if _, ok := body["projectStartDate"]; ok {
		if d := parseDateTime(strPtrFromAny(body["projectStartDate"])); d != nil {
			fields["project_date"] = d
		}
	}
	if _, ok := body["projectEndDate"]; ok {
		if d := parseDateTime(strPtrFromAny(body["projectEndDate"])); d != nil {
			fields["valid_until"] = d
		}
	}
	if len(fields) > 0 {
		if err := s.Repository.UpdateCrmProject(id, fields); err != nil {
			return nil, err
		}
	}

	if _, ok := body["picUserId"]; ok {
		if err := s.Repository.UpsertPmProjectPic(id, int64PtrFromAny(body["picUserId"])); err != nil {
			return nil, err
		}
	}

	return s.CrmProjectDetail(id)
}

func (s *pmService) CrmProjectDetail(id int64) (model.Row, error) {
	row, err := s.Repository.CrmProjectByID(id)
	if err != nil {
		return nil, err
	}
	tasks, err := s.Repository.ProjectTasksByCrmProject(id)
	if err != nil {
		tasks = []model.Row{}
	}
	team, err := s.Repository.TeamByCrmProject(id)
	if err != nil {
		team = []model.Row{}
	}
	activity, err := s.Repository.ActivityByCrmProject(id)
	if err != nil {
		activity = []model.Row{}
	}

	// No dedicated "stage" column exists for CRM-linked PM projects yet, so
	// derive a reasonable approximation from real task completion instead of
	// a fixed value.
	total := len(tasks)
	done := 0
	for _, t := range tasks {
		if str(t["status"]) == "done" {
			done++
		}
	}
	progress := 0
	if total > 0 {
		progress = done * 100 / total
	}
	stage := "Planning"
	if total > 0 {
		if progress == 100 {
			stage = "Closed"
		} else {
			stage = "Fieldwork"
		}
	}

	result := model.Row{}
	for k, v := range row {
		result[k] = v
	}
	result["tasks"] = tasks
	result["team"] = team
	result["activity"] = activity
	result["stage"] = stage
	result["progress"] = progress
	return result, nil
}

func (s *pmService) TasksByCrmProject(id int64) ([]model.Row, error) {
	return s.Repository.ProjectTasksByCrmProject(id)
}

func (s *pmService) CrmProjectMembers(crmProjectID int64) ([]model.Row, error) {
	return s.Repository.TeamByCrmProject(crmProjectID)
}

func (s *pmService) AddCrmProjectMember(crmProjectID int64, body map[string]interface{}, userID string) (model.Row, error) {
	memberUserID := int64FromAny(body["userId"], 0)
	if memberUserID == 0 {
		return nil, fmt.Errorf("userId is required")
	}
	isLeader, _ := body["isLeader"].(bool)

	id := uuid.NewString()
	fields := map[string]interface{}{
		"id":             id,
		"crm_project_id": crmProjectID,
		"user_id":        memberUserID,
		"role":           strPtrFromAny(body["role"]),
		"is_leader":      isLeader,
		"created_by":     nilIfEmpty(userID),
	}
	if err := s.Repository.InsertCrmProjectMember(fields); err != nil {
		return nil, err
	}
	return s.Repository.CrmProjectMemberByID(id, crmProjectID)
}

func (s *pmService) UpdateCrmProjectMember(crmProjectID int64, memberID string, body map[string]interface{}) (model.Row, error) {
	if !validUUID(memberID) {
		return nil, fmt.Errorf("invalid member id")
	}
	fields := map[string]interface{}{}
	if _, ok := body["role"]; ok {
		fields["role"] = strPtrFromAny(body["role"])
	}
	if v, ok := body["isLeader"]; ok {
		if isLeader, ok2 := v.(bool); ok2 {
			fields["is_leader"] = isLeader
		}
	}
	if len(fields) > 0 {
		if err := s.Repository.UpdateCrmProjectMember(memberID, crmProjectID, fields); err != nil {
			return nil, err
		}
	}
	return s.Repository.CrmProjectMemberByID(memberID, crmProjectID)
}

func (s *pmService) DeleteCrmProjectMember(crmProjectID int64, memberID string) error {
	if !validUUID(memberID) {
		return fmt.Errorf("invalid member id")
	}
	return s.Repository.DeleteCrmProjectMember(memberID, crmProjectID)
}

func (s *pmService) CreateTaskForCrmProject(crmProjectID int64, body map[string]interface{}, userID string) (model.Row, error) {
	title, _ := body["title"].(string)
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	statusKeyInput := strFromAny(body["status"], "to_do")
	statusID, err := s.Repository.ResolveStatusID(nil, statusKeyInput)
	if err != nil {
		return nil, err
	}
	statusKey, err := s.Repository.StatusKeyByID(statusID)
	if err != nil {
		return nil, err
	}

	id := uuid.NewString()
	fields := map[string]interface{}{
		"id":                 id,
		"crm_project_id":     crmProjectID,
		"title":              title,
		"description":        strPtrFromAny(body["description"]),
		"start_date":         parseDateTime(strPtrFromAny(body["startDate"])),
		"deadline":           parseDateTime(strPtrFromAny(body["deadline"])),
		"actual_start_date":  parseDateTime(strPtrFromAny(body["actualStartDate"])),
		"actual_finish_date": parseDateTime(strPtrFromAny(body["actualFinishDate"])),
		"collaborators":      strPtrFromAny(body["collaborators"]),
		"priority_id":        int64FromAny(body["priorityId"], 0),
		"progress_pct":       clampPercent(int64FromAny(body["progressPct"], 0)),
		"status":             statusKey,
		"status_id":          statusID,
		"parent_task_id":     uuidPtrFromAny(body["parentTaskId"]),
		"assigned_to":        int64PtrFromAny(body["assignedTo"]),
		"created_by":         nilIfEmpty(userID),
	}
	if err := s.Repository.InsertProjectTask(fields); err != nil {
		return nil, err
	}

	actorUserID, actorName := s.resolveActor(actorSource(body, userID))
	isSubtask := uuidPtrFromAny(body["parentTaskId"]) != nil
	action, description := "created", fmt.Sprintf("%s created task: %s", actorName, title)
	if isSubtask {
		action, description = "subtask_created", fmt.Sprintf("%s created subtask: %s", actorName, title)
	}
	s.logTaskActivity(id, crmProjectID, actorUserID, actorName, action, "", nil, strPtr(title), description)

	return s.Repository.ProjectTaskByID(id)
}

func (s *pmService) UpdateProjectTask(id string, body map[string]interface{}) (model.Row, error) {
	if !validUUID(id) {
		return nil, fmt.Errorf("invalid task id")
	}
	before, err := s.Repository.ProjectTaskByID(id)
	if err != nil {
		return nil, err
	}
	fields := map[string]interface{}{}
	if title, ok := body["title"].(string); ok && strings.TrimSpace(title) != "" {
		fields["title"] = title
	}
	if _, ok := body["description"]; ok {
		fields["description"] = strPtrFromAny(body["description"])
	}
	if _, ok := body["startDate"]; ok {
		fields["start_date"] = parseDateTime(strPtrFromAny(body["startDate"]))
	}
	if _, ok := body["deadline"]; ok {
		fields["deadline"] = parseDateTime(strPtrFromAny(body["deadline"]))
	}
	if _, ok := body["actualStartDate"]; ok {
		fields["actual_start_date"] = parseDateTime(strPtrFromAny(body["actualStartDate"]))
	}
	if _, ok := body["actualFinishDate"]; ok {
		fields["actual_finish_date"] = parseDateTime(strPtrFromAny(body["actualFinishDate"]))
	}
	if _, ok := body["collaborators"]; ok {
		fields["collaborators"] = strPtrFromAny(body["collaborators"])
	}
	if _, ok := body["priorityId"]; ok {
		fields["priority_id"] = int64FromAny(body["priorityId"], 0)
	}
	if _, ok := body["progressPct"]; ok {
		fields["progress_pct"] = clampPercent(int64FromAny(body["progressPct"], 0))
	}
	if _, ok := body["assignedTo"]; ok {
		fields["assigned_to"] = int64PtrFromAny(body["assignedTo"])
	}
	if _, ok := body["parentTaskId"]; ok {
		fields["parent_task_id"] = uuidPtrFromAny(body["parentTaskId"])
	}
	if statusRaw, ok := body["status"].(string); ok && strings.TrimSpace(statusRaw) != "" {
		statusID, err := s.Repository.ResolveStatusID(nil, statusRaw)
		if err != nil {
			return nil, err
		}
		statusKey, err := s.Repository.StatusKeyByID(statusID)
		if err != nil {
			return nil, err
		}
		fields["status"] = statusKey
		fields["status_id"] = statusID
		fields["status_changed_at"] = time.Now().UTC()
	}
	if len(fields) > 0 {
		if err := s.Repository.UpdateProjectTask(id, fields); err != nil {
			return nil, err
		}
	}
	after, err := s.Repository.ProjectTaskByID(id)
	if err != nil {
		return nil, err
	}
	actorUserID, actorName := s.resolveActor(actorSource(body, ""))
	s.logTaskFieldChanges(id, before, after, actorUserID, actorName)
	return after, nil
}

func (s *pmService) MoveProjectTaskByKey(id, statusKey string, actorUserIDRaw interface{}) (model.Row, error) {
	if !validUUID(id) {
		return nil, fmt.Errorf("invalid task id")
	}
	before, err := s.Repository.ProjectTaskByID(id)
	if err != nil {
		return nil, err
	}
	if err := s.Repository.MoveProjectTaskByKey(id, statusKey); err != nil {
		return nil, err
	}
	after, err := s.Repository.ProjectTaskByID(id)
	if err != nil {
		return nil, err
	}
	actorUserID, actorName := s.resolveActor(actorUserIDRaw)
	s.logTaskFieldChanges(id, before, after, actorUserID, actorName)
	return after, nil
}

func (s *pmService) DeleteProjectTask(id string, actorUserIDRaw interface{}) error {
	if !validUUID(id) {
		return fmt.Errorf("invalid task id")
	}
	before, err := s.Repository.ProjectTaskByID(id)
	if err == nil {
		projectID := int64FromAny(before["crm_project_id"], 0)
		actorUserID, actorName := s.resolveActor(actorUserIDRaw)
		title := str(before["title"])
		// Log BEFORE the delete, per spec: never lose the delete record even
		// though the task itself becomes invisible to ProjectTaskByID
		// (deleted = TRUE) right after this.
		s.logTaskActivity(id, projectID, actorUserID, actorName, "deleted", "", strPtr(title), nil,
			fmt.Sprintf("%s deleted task: %s", actorName, title))
	}
	return s.Repository.DeleteProjectTask(id)
}

func (s *pmService) TaskActivityLogs(taskID string) ([]model.Row, error) {
	if !validUUID(taskID) {
		return nil, fmt.Errorf("invalid task id")
	}
	return s.Repository.TaskActivityLogs(taskID)
}

func (s *pmService) GanttMembers() ([]model.Row, error) {
	return s.Repository.GanttMembers()
}

func periodRange(period string) (*string, *string) {
	now := time.Now().UTC()
	switch strings.ToLower(strings.TrimSpace(period)) {
	case "today":
		d := now.Format("2006-01-02")
		return &d, &d
	case "this_week":
		offset := int(now.Weekday()) - 1
		if offset < 0 {
			offset = 6
		}
		monday := now.AddDate(0, 0, -offset)
		from := monday.Format("2006-01-02")
		to := monday.AddDate(0, 0, 6).Format("2006-01-02")
		return &from, &to
	case "this_month":
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
		to := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
		return &from, &to
	default:
		return nil, nil
	}
}

func (s *pmService) Timesheets(search string, userID, crmProjectID *int64, status, period string) (model.Row, error) {
	from, to := periodRange(period)
	entries, err := s.Repository.Timesheets(search, userID, crmProjectID, status, from, to)
	if err != nil {
		return nil, err
	}
	summary, err := s.Repository.TimesheetSummary(userID)
	if err != nil {
		return nil, err
	}
	return model.Row{"entries": entries, "summary": summary}, nil
}

func (s *pmService) CreateTimesheet(req model.TimesheetRequest) (model.Row, error) {
	if req.Hours <= 0 {
		return nil, fmt.Errorf("hours must be greater than 0")
	}
	workDate := parseDateTime(&req.WorkDate)
	if workDate == nil {
		return nil, fmt.Errorf("invalid work_date: %s", req.WorkDate)
	}
	var taskID interface{}
	if req.TaskID != nil {
		if _, err := uuid.Parse(*req.TaskID); err == nil {
			taskID = *req.TaskID
		}
	}
	fields := map[string]interface{}{
		"crm_project_id": req.CrmProjectID,
		"task_id":        taskID,
		"user_id":        req.UserID,
		"work_date":      workDate.Format("2006-01-02"),
		"hours":          req.Hours,
		"description":    req.Description,
		"status":         "pending",
		"created_by":     req.UserID,
	}
	id, err := s.Repository.InsertTimesheet(fields)
	if err != nil {
		return nil, err
	}
	return s.Repository.TimesheetByID(id)
}

func (s *pmService) UpdateTimesheetStatus(id int64, status string, approvedBy *int64) (model.Row, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "pending" && status != "approved" && status != "rejected" {
		return nil, fmt.Errorf("invalid status: %s", status)
	}
	if err := s.Repository.UpdateTimesheetStatus(id, status, approvedBy); err != nil {
		return nil, err
	}
	return s.Repository.TimesheetByID(id)
}

func (s *pmService) DeleteTimesheet(id int64) error {
	return s.Repository.DeleteTimesheet(id)
}

func strFromAny(v interface{}, fallback string) string {
	if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
		return s
	}
	return fallback
}

func strPtrFromAny(v interface{}) *string {
	s, ok := v.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func int64FromAny(v interface{}, fallback int64) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case string:
		if i, err := strconv.ParseInt(n, 10, 64); err == nil {
			return i
		}
	}
	return fallback
}

// validUUID guards against a stale/malformed id (e.g. a browser tab that
// cached a task id from before the id-encoding bug fix) reaching Postgres as
// a raw `WHERE id = ?` on a uuid column, which fails with a cryptic
// "invalid input syntax for type uuid" error instead of a clean 4xx.
func validUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

func clampPercent(v int64) int64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// int64PtrFromAny parses a bigint `users.id` reference (assignee, PIC,
// activity-log actor — matching GanttMembers), unlike the legacy pm_tasks
// board which uses a musers UUID. 0 is treated the same as "not provided":
// Postgres SERIAL ids here always start at 1, so 0 can never be a real user,
// but the frontend's "no selection" state for a required-number field often
// serializes to 0 rather than omitting the key or sending null. Returning
// &0 in that case previously reached the DB as assigned_to = 0, which
// violates pm_project_tasks_assigned_to_fkey (500 on every save where the
// assignee was left blank).
func int64PtrFromAny(v interface{}) *int64 {
	switch n := v.(type) {
	case float64:
		if n == 0 {
			return nil
		}
		id := int64(n)
		return &id
	case string:
		if i, err := strconv.ParseInt(n, 10, 64); err == nil && i != 0 {
			return &i
		}
	}
	return nil
}

func uuidPtrFromAny(v interface{}) *string {
	s, ok := v.(string)
	if !ok {
		return nil
	}
	if _, err := uuid.Parse(s); err != nil {
		return nil
	}
	return &s
}

// nilIfEmpty guards UUID columns (created_by, etc.) against an anonymous
// (empty-string) actor, since the PM group currently runs without auth.
func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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

// ─── Task activity log helpers ─────────────────────────────────────────────

var priorityLabels = map[int64]string{1: "Low", 2: "Normal", 3: "High", 4: "Critical"}

func priorityLabel(v interface{}) string {
	id, _ := toInt64(v)
	if id == 0 {
		if f, ok := v.(float64); ok {
			id = int64(f)
		}
	}
	if label, ok := priorityLabels[id]; ok {
		return label
	}
	return "Normal"
}

// actorSource prefers an explicit actorUserId from the request body (what
// the frontend sends, sourced from the logged-in CRM user in localStorage)
// over the JWT-derived userID, since the PM API group currently runs
// without enforced auth and userID is usually empty in practice.
func actorSource(body map[string]interface{}, userID string) interface{} {
	if v, ok := body["actorUserId"]; ok && v != nil {
		if s, isStr := v.(string); !isStr || strings.TrimSpace(s) != "" {
			return v
		}
	}
	if strings.TrimSpace(userID) != "" {
		return userID
	}
	return nil
}

// resolveActor turns an actorUserId (from actorSource, a request query
// param, etc.) into a stored user id + a human-readable name snapshot.
// Falls back to "System" when no actor was identified — expected for now
// since the PM API has no enforced auth.
func (s *pmService) resolveActor(raw interface{}) (*int64, string) {
	uid := int64PtrFromAny(raw)
	if uid == nil {
		return nil, "System"
	}
	name, err := s.Repository.ResolveUserName(*uid)
	if err != nil || strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("User #%d", *uid)
	}
	return uid, name
}

func (s *pmService) logTaskActivity(taskID string, projectID int64, actorUserID *int64, actorName, action, fieldName string, oldValue, newValue *string, description string) {
	fields := map[string]interface{}{
		"task_id":       taskID,
		"project_id":    projectID,
		"actor_user_id": actorUserID,
		"actor_name":    actorName,
		"action":        action,
		"field_name":    nilIfEmptyStr(fieldName),
		"old_value":     oldValue,
		"new_value":     newValue,
		"description":   description,
	}
	// Activity logging must never block or fail the task mutation it's
	// attached to — swallow the error (there's nothing actionable the
	// caller could do with a failed audit-log write anyway).
	_ = s.Repository.InsertTaskActivityLog(fields)
}

func nilIfEmptyStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func strPtr(s string) *string { return &s }

// formatTaskDate renders a Row's date/time value (however the driver scanned
// it — time.Time is typical, but this tolerates strings/byte slices too) as
// a plain calendar date for human-readable activity descriptions.
func formatTaskDate(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case time.Time:
		return t.Format("2006-01-02")
	case *time.Time:
		if t == nil {
			return ""
		}
		return t.Format("2006-01-02")
	case []byte:
		return strings.SplitN(string(t), "T", 2)[0]
	case string:
		return strings.SplitN(t, "T", 2)[0]
	default:
		return fmt.Sprintf("%v", v)
	}
}

// taskActivityField describes one trackable column for diffing a task
// update: how to read + format it, and the label used in the log.
type taskActivityField struct {
	label    string // used in field_name + description, e.g. "status"
	value    func(row model.Row) string
	describe func(actor, oldVal, newVal string) string
}

var taskActivityFields = []taskActivityField{
	{
		label: "title",
		value: func(row model.Row) string { return str(row["title"]) },
		describe: func(actor, oldVal, newVal string) string {
			return fmt.Sprintf("%s renamed task from \"%s\" to \"%s\"", actor, oldVal, newVal)
		},
	},
	{
		label: "status",
		value: func(row model.Row) string { return strFromAny(row["status_title"], str(row["status"])) },
		describe: func(actor, oldVal, newVal string) string {
			return fmt.Sprintf("%s changed status from %s to %s", actor, oldVal, newVal)
		},
	},
	{
		label: "assignee",
		value: func(row model.Row) string { return strFromAny(row["assignee_name"], "Unassigned") },
		describe: func(actor, oldVal, newVal string) string {
			return fmt.Sprintf("%s changed assignee from %s to %s", actor, oldVal, newVal)
		},
	},
	{
		label: "progress",
		value: func(row model.Row) string { return fmt.Sprintf("%v%%", row["progress_pct"]) },
		describe: func(actor, oldVal, newVal string) string {
			return fmt.Sprintf("%s updated progress from %s to %s", actor, oldVal, newVal)
		},
	},
	{
		label: "priority",
		value: func(row model.Row) string { return priorityLabel(row["priority_id"]) },
		describe: func(actor, oldVal, newVal string) string {
			return fmt.Sprintf("%s changed priority from %s to %s", actor, oldVal, newVal)
		},
	},
	{
		label: "plan_start",
		value: func(row model.Row) string { return formatTaskDate(row["start_date"]) },
		describe: func(actor, oldVal, newVal string) string {
			return fmt.Sprintf("%s changed plan start date from %s to %s", actor, dateOrUnset(oldVal), dateOrUnset(newVal))
		},
	},
	{
		label: "plan_finish",
		value: func(row model.Row) string { return formatTaskDate(row["deadline"]) },
		describe: func(actor, oldVal, newVal string) string {
			return fmt.Sprintf("%s changed plan finish date from %s to %s", actor, dateOrUnset(oldVal), dateOrUnset(newVal))
		},
	},
	{
		label: "actual_start",
		value: func(row model.Row) string { return formatTaskDate(row["actual_start_date"]) },
		describe: func(actor, oldVal, newVal string) string {
			return fmt.Sprintf("%s changed actual start date from %s to %s", actor, dateOrUnset(oldVal), dateOrUnset(newVal))
		},
	},
	{
		label: "actual_finish",
		value: func(row model.Row) string { return formatTaskDate(row["actual_finish_date"]) },
		describe: func(actor, oldVal, newVal string) string {
			return fmt.Sprintf("%s changed actual finish date from %s to %s", actor, dateOrUnset(oldVal), dateOrUnset(newVal))
		},
	},
}

func dateOrUnset(v string) string {
	if v == "" {
		return "(not set)"
	}
	return v
}

// logTaskFieldChanges diffs before/after snapshots of a task row (both from
// ProjectTaskByID, same shape) and writes one activity log entry per changed
// tracked field — matches the granular examples in the spec ("changed status
// from To Do to In progress") rather than one noisy combined entry.
func (s *pmService) logTaskFieldChanges(taskID string, before, after model.Row, actorUserID *int64, actorName string) {
	projectID, _ := toInt64(before["crm_project_id"])
	if projectID == 0 {
		projectID = int64FromAny(before["crm_project_id"], 0)
	}
	for _, field := range taskActivityFields {
		oldVal := field.value(before)
		newVal := field.value(after)
		if oldVal == newVal {
			continue
		}
		s.logTaskActivity(taskID, projectID, actorUserID, actorName, "updated", field.label,
			strPtr(oldVal), strPtr(newVal), field.describe(actorName, oldVal, newVal))
	}
}
