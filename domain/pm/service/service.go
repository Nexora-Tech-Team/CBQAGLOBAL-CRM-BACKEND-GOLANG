package service

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"erp-cbqa-global/domain/pm/model"
	"erp-cbqa-global/domain/pm/repository"
)

// ErrInvalidWorkloadPeriod marks a bad `period`/`month` query param on
// GET /v1/pm/dashboard/team-workload — the controller maps this to 400,
// any other error from that path to 500.
var ErrInvalidWorkloadPeriod = errors.New("invalid workload period")

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

	// Work Timer (clock in/out) — pm_task_time_logs.
	ClockInTask(taskID string, userID int64, actorUserIDRaw interface{}) (model.Row, error)
	ClockOutTask(taskID string, userID int64, note *string, actorUserIDRaw interface{}) (model.Row, error)
	TaskTimeLogs(taskID string, userID int64) (model.Row, error)
	ActiveTimeLogForUser(userID int64) (model.Row, error)
	CreateManualTimeLog(taskID string, body map[string]interface{}, actorUserIDRaw interface{}) (model.Row, error)
	UpdateManualTimeLog(logID int64, body map[string]interface{}, actorUserIDRaw interface{}) (model.Row, error)
	DeleteManualTimeLog(logID int64, actorUserIDRaw interface{}) error

	CrmProjectMembers(crmProjectID int64) ([]model.Row, error)
	AddCrmProjectMember(crmProjectID int64, body map[string]interface{}, userID string) (model.Row, error)
	UpdateCrmProjectMember(crmProjectID int64, memberID string, body map[string]interface{}) (model.Row, error)
	DeleteCrmProjectMember(crmProjectID int64, memberID string) error

	Timesheets(search string, userID, crmProjectID *int64, status, period string) (model.Row, error)
	CreateTimesheet(req model.TimesheetRequest) (model.Row, error)
	UpdateTimesheetStatus(id int64, status string, approvedBy *int64) (model.Row, error)
	DeleteTimesheet(id int64) error

	// DashboardSummary powers /v1/pm/dashboard/summary — the high-level PM
	// portfolio monitoring dashboard (project health, portfolio progress,
	// team workload, active work sessions, upcoming deadlines, recent
	// activity). See the doc comment on the implementation for the exact
	// response shape and every derivation rule.
	DashboardSummary() (model.Row, error)

	// TeamWorkloadByPeriod powers /v1/pm/dashboard/team-workload — the same
	// Team Workload data as DashboardSummary's teamWorkload, but scoped to
	// an arbitrary period (this_week/this_month/custom_month) instead of
	// always being the current calendar week. See the implementation's doc
	// comment for period resolution rules and CLAUDE.md for the full
	// formula writeup.
	TeamWorkloadByPeriod(periodType, monthParam string) (model.Row, error)
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
	rows, err := s.Repository.CrmProjects(search)
	if err != nil {
		return nil, err
	}

	// One extra flat query for every active task across every project,
	// grouped in memory by crm_project_id — the alternative (fetch tasks
	// per project row) is exactly the N+1 pattern this needs to avoid.
	// PmPortfolioTasks (not the narrower ActiveTaskProgressInputs) is used
	// here because Health's overdue check needs each task's `deadline`,
	// which ActiveTaskProgressInputs doesn't select.
	tasks, err := s.Repository.PmPortfolioTasks()
	if err != nil {
		for _, row := range rows {
			row["progress"] = int64(0)
			row["health"] = "Healthy"
		}
		return rows, nil
	}
	tasksByProject := make(map[int64][]model.Row, len(rows))
	for _, t := range tasks {
		pid, _ := toInt64(t["crm_project_id"])
		tasksByProject[pid] = append(tasksByProject[pid], t)
	}
	now := time.Now()
	for _, row := range rows {
		pid, _ := toInt64(row["crm_project_id"])
		projectTasks := tasksByProject[pid]
		actualProgress := projectProgressFromTasks(projectTasks, computeTaskProgress(projectTasks))
		row["progress"] = actualProgress

		stage := str(row["stage"])
		blockedTasks, _ := toInt64(row["blocked_tasks"])
		plannedProgress := planProgressPercent(timeFromRow(row["plan_start"]), timeFromRow(row["plan_end"]), now)

		var overdueTaskCount int64
		for _, t := range projectTasks {
			if isOverdueTask(t, now) {
				overdueTaskCount++
			}
		}
		planEndPassed := false
		if planEnd := timeFromRow(row["plan_end"]); planEnd != nil && isPastCalendarDay(*planEnd, now) {
			planEndPassed = true
		}

		row["health"] = deriveProjectHealth(stage, blockedTasks, overdueTaskCount, planEndPassed, actualProgress, plannedProgress)
	}
	return rows, nil
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

	// Progress is a separate metric from Stage: Stage comes straight from
	// row["stage"], computed by the CrmProjectByID SQL query (task_stage
	// CASE) — never touched here. Progress is computed bottom-up from each
	// task's status (and, for parent tasks, their active children) via
	// computeTaskProgress — the same helper CrmProjects (list) and
	// TasksByCrmProject use, so list/detail/task-tab progress can never
	// disagree. applyTaskProgress overrides each task row's progress_pct in
	// place with its effective value and tags has_active_subtasks.
	progressByTask := applyTaskProgress(tasks)
	projectProgress := projectProgressFromTasks(tasks, progressByTask)

	result := model.Row{}
	for k, v := range row {
		result[k] = v
	}
	result["tasks"] = tasks
	result["team"] = team
	result["activity"] = activity
	result["progress"] = projectProgress
	return result, nil
}

func (s *pmService) TasksByCrmProject(id int64) ([]model.Row, error) {
	tasks, err := s.Repository.ProjectTasksByCrmProject(id)
	if err != nil {
		return nil, err
	}
	applyTaskProgress(tasks)
	return tasks, nil
}

// attachEffectiveProgress recomputes progress for a single task by pulling
// every active task in its project (one query) and running the same
// computeTaskProgress used by CrmProjectDetail/CrmProjects/TasksByCrmProject
// — guarantees a create/update/move response always matches what the very
// next list/detail fetch would show for that task, per the requirement that
// a write must be reflected by the next response, not just eventually.
func (s *pmService) attachEffectiveProgress(crmProjectID int64, row model.Row) (model.Row, error) {
	tasks, err := s.Repository.ProjectTasksByCrmProject(crmProjectID)
	if err != nil {
		return row, nil // don't fail the whole write just because the progress refresh couldn't run
	}
	progress := applyTaskProgress(tasks)
	if p, ok := progress[str(row["id"])]; ok {
		row["progress_pct"] = p.Progress
		row["has_active_subtasks"] = p.IsParent
		if p.IsParent {
			row["status"] = p.Status
			row["status_key"] = p.Status
			if meta, ok := statusMeta[p.Status]; ok {
				row["status_title"] = meta.Title
				row["status_color"] = meta.Color
			}
		}
	}
	return row, nil
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

	// A brand-new task always starts as a leaf (it can't have children yet),
	// so its initial progress follows the same leaf status rule table used
	// everywhere else — a manual progressPct in the body still wins, but
	// to_do/done/in_review/in_progress floors and ceilings still apply.
	var initialManualProgress *int64
	if _, ok := body["progressPct"]; ok {
		v := clampPercent(int64FromAny(body["progressPct"], 0))
		initialManualProgress = &v
	}
	initialProgress := leafProgressForStatus(statusKey, 0, initialManualProgress)

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
		"progress_pct":       initialProgress,
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

	created, err := s.Repository.ProjectTaskByID(id)
	if err != nil {
		return nil, err
	}
	// A new subtask changes its parent's (and the project's) effective
	// progress — attachEffectiveProgress recomputes from the whole project's
	// current task set, so this is reflected immediately, not just on the
	// next unrelated fetch.
	return s.attachEffectiveProgress(crmProjectID, created)
}

func (s *pmService) UpdateProjectTask(id string, body map[string]interface{}) (model.Row, error) {
	if !validUUID(id) {
		return nil, fmt.Errorf("invalid task id")
	}
	before, err := s.Repository.ProjectTaskByID(id)
	if err != nil {
		return nil, err
	}
	childCount, err := s.Repository.ActiveChildTaskCount(id)
	if err != nil {
		return nil, err
	}
	isParent := childCount > 0

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
	if _, ok := body["assignedTo"]; ok {
		fields["assigned_to"] = int64PtrFromAny(body["assignedTo"])
	}
	if _, ok := body["parentTaskId"]; ok {
		fields["parent_task_id"] = uuidPtrFromAny(body["parentTaskId"])
	}

	// Resolve the status this task will have AFTER this update (its new
	// value if provided, else whatever it already had) — needed even when
	// only progressPct changes, since the leaf progress rule depends on it.
	// A parent task's status is always derived from its children on read
	// (see deriveParentStatus/computeTaskProgress) exactly like its
	// progress — a manual status change aimed at one is silently dropped,
	// same reasoning as the progress guard below (the request may
	// legitimately be touching unrelated fields in the same call).
	newStatusKey := str(before["status"])
	if statusRaw, ok := body["status"].(string); ok && strings.TrimSpace(statusRaw) != "" && !isParent {
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
		newStatusKey = statusKey
	}

	// Progress: a parent task's progress is always derived from its children
	// on read (see computeTaskProgress) — a manual progressPct aimed at one
	// is silently dropped here rather than erroring, since the same request
	// may legitimately be updating unrelated fields (title, assignee, ...).
	// Leaf tasks route through leafProgressForStatus so a manual edit and/or
	// a status change both still respect the to_do/in_review/done floors
	// and ceilings, exactly like Kanban drag-move does.
	if !isParent {
		_, progressProvided := body["progressPct"]
		_, statusChanged := fields["status"]
		if progressProvided || statusChanged {
			var manual *int64
			if progressProvided {
				v := clampPercent(int64FromAny(body["progressPct"], 0))
				manual = &v
			}
			fields["progress_pct"] = leafProgressForStatus(newStatusKey, progressPctFromRow(before["progress_pct"]), manual)
		}
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

	return s.attachEffectiveProgress(projectIDFromRow(after), after)
}

func (s *pmService) MoveProjectTaskByKey(id, statusKey string, actorUserIDRaw interface{}) (model.Row, error) {
	if !validUUID(id) {
		return nil, fmt.Errorf("invalid task id")
	}
	before, err := s.Repository.ProjectTaskByID(id)
	if err != nil {
		return nil, err
	}

	// A parent task's status AND progress are always derived from its
	// children on read (deriveParentStatus / leafProgressForStatus) —
	// dragging one on the Kanban board (the UI should prevent this, but the
	// backend is the actual source of truth) is a no-op on both stored
	// columns rather than an error, consistent with how UpdateProjectTask
	// silently drops a manual progress/status aimed at a parent.
	childCount, err := s.Repository.ActiveChildTaskCount(id)
	if err != nil {
		return nil, err
	}
	if childCount == 0 {
		if err := s.Repository.MoveProjectTaskByKey(id, statusKey); err != nil {
			return nil, err
		}
		// Kanban drag never carries a manual progress value — apply the same
		// leaf status rule table as everywhere else (to_do->0, in_progress->10
		// only if it was 0, in_review->floor 90, done->100, blocked->unchanged).
		newProgress := leafProgressForStatus(statusKey, progressPctFromRow(before["progress_pct"]), nil)
		if err := s.Repository.UpdateProjectTask(id, map[string]interface{}{"progress_pct": newProgress}); err != nil {
			return nil, err
		}
	}

	after, err := s.Repository.ProjectTaskByID(id)
	if err != nil {
		return nil, err
	}
	actorUserID, actorName := s.resolveActor(actorUserIDRaw)
	s.logTaskFieldChanges(id, before, after, actorUserID, actorName)

	return s.attachEffectiveProgress(projectIDFromRow(after), after)
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

// ─── Work Timer (clock in/out) ─────────────────────────────────────────────

// formatDurationShort renders seconds as a compact "1h 25m" label for
// activity-log descriptions — mirrors the frontend's own duration formatter
// so the log reads the same way the Work Logs panel displays it.
func formatDurationShort(totalSeconds int64) string {
	if totalSeconds < 60 {
		return fmt.Sprintf("%ds", totalSeconds)
	}
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	if hours == 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

func (s *pmService) ClockInTask(taskID string, userID int64, actorUserIDRaw interface{}) (model.Row, error) {
	if !validUUID(taskID) {
		return nil, fmt.Errorf("invalid task id")
	}
	if userID == 0 {
		return nil, fmt.Errorf("userId is required")
	}
	task, err := s.Repository.ProjectTaskByID(taskID)
	if err != nil {
		return nil, err
	}
	projectID := projectIDFromRow(task)

	row, err := s.Repository.ClockIn(taskID, userID, &projectID)
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyClockedIn) {
			return nil, s.describeAlreadyClockedIn(userID, err)
		}
		return nil, err
	}

	actorUserID, actorName := s.resolveActor(actorSource(map[string]interface{}{"actorUserId": actorUserIDRaw}, ""))
	if actorUserID == nil {
		actorUserID = &userID
	}
	if actorName == "" || actorName == "System" {
		if name, nameErr := s.Repository.ResolveUserName(userID); nameErr == nil && name != "" {
			actorName = name
		}
	}
	s.logTaskActivity(taskID, projectID, actorUserID, actorName, "clock_in", "", nil, nil,
		fmt.Sprintf("%s clocked in", actorName))

	return row, nil
}

func (s *pmService) ClockOutTask(taskID string, userID int64, note *string, actorUserIDRaw interface{}) (model.Row, error) {
	if !validUUID(taskID) {
		return nil, fmt.Errorf("invalid task id")
	}
	if userID == 0 {
		return nil, fmt.Errorf("userId is required")
	}
	row, err := s.Repository.ClockOut(taskID, userID, note)
	if err != nil {
		if errors.Is(err, repository.ErrNoActiveTimeLog) {
			return nil, err
		}
		return nil, err
	}

	projectID := projectIDFromRow(row)
	actorUserID, actorName := s.resolveActor(actorSource(map[string]interface{}{"actorUserId": actorUserIDRaw}, ""))
	if actorUserID == nil {
		actorUserID = &userID
	}
	if actorName == "" || actorName == "System" {
		if name, nameErr := s.Repository.ResolveUserName(userID); nameErr == nil && name != "" {
			actorName = name
		}
	}
	duration, _ := toInt64(row["duration_seconds"])
	s.logTaskActivity(taskID, projectID, actorUserID, actorName, "clock_out", "", nil, nil,
		fmt.Sprintf("%s clocked out (%s)", actorName, formatDurationShort(duration)))

	return row, nil
}

// TaskTimeLogs bundles the full session history for a task with the two
// running totals the Work Logs panel needs (all-users total, this-user's
// own total — both over CLOSED sessions only, since an open session's
// duration isn't final yet) and, separately, the requesting user's own
// still-open session on this task (if any), so the frontend can render the
// running timer + Clock Out button without a second request.
func (s *pmService) TaskTimeLogs(taskID string, userID int64) (model.Row, error) {
	if !validUUID(taskID) {
		return nil, fmt.Errorf("invalid task id")
	}
	logs, err := s.Repository.TimeLogsByTask(taskID)
	if err != nil {
		return nil, err
	}

	var totalDuration, myDuration int64
	var activeLog model.Row
	for _, log := range logs {
		logUserID, _ := toInt64(log["user_id"])
		if log["ended_at"] == nil {
			if userID != 0 && logUserID == userID {
				activeLog = log
			}
			continue
		}
		duration, _ := toInt64(log["duration_seconds"])
		totalDuration += duration
		if userID != 0 && logUserID == userID {
			myDuration += duration
		}
	}

	return model.Row{
		"logs":                 logs,
		"totalDurationSeconds": totalDuration,
		"myDurationSeconds":    myDuration,
		"activeLog":            activeLog,
	}, nil
}

// ActiveTimeLogForUser answers "is this user clocked in anywhere right now",
// enriched with the task's title/project so the frontend can render a
// friendly "Clocked in on another task: <title>" message without a second
// lookup.
// elapsedSecondsFrom computes how long ago a DB-scanned timestamp column
// (time.Time via database/sql) was, floored at 0 so clock skew between the
// app server and DB never reports a negative running timer.
func elapsedSecondsFrom(v interface{}) int64 {
	t, ok := v.(time.Time)
	if !ok {
		return 0
	}
	elapsed := int64(time.Since(t).Seconds())
	if elapsed < 0 {
		return 0
	}
	return elapsed
}

// ActiveTimeLogForUser answers "is this user clocked in anywhere right now"
// as a flat, frontend-ready shape — {active:false} or {active:true, taskId,
// taskTitle, taskType (task/subtask), parentTaskId/Title, projectId/Title,
// startedAt, elapsedSeconds} — so the drawer/topbar can render "Clocked in
// on another task: X" (and, for a subtask, name its parent) without a
// second round trip.
func (s *pmService) ActiveTimeLogForUser(userID int64) (model.Row, error) {
	if userID == 0 {
		return nil, fmt.Errorf("userId is required")
	}
	active, err := s.Repository.ActiveTimeLogForUser(userID)
	if err != nil {
		return nil, err
	}
	if active == nil {
		return model.Row{"active": false}, nil
	}

	taskID := str(active["task_id"])
	result := model.Row{
		"active":         true,
		"taskId":         taskID,
		"startedAt":      active["started_at"],
		"elapsedSeconds": elapsedSecondsFrom(active["started_at"]),
	}

	task, err := s.Repository.ProjectTaskByID(taskID)
	if err != nil {
		// Session row outlived its task somehow (shouldn't happen — tasks
		// are soft-deleted, never hard-deleted) — still report the bare
		// active session instead of erroring the whole check.
		return result, nil
	}
	result["taskTitle"] = task["title"]

	if parentTaskID := str(task["parent_task_id"]); parentTaskID != "" {
		result["taskType"] = "subtask"
		result["parentTaskId"] = parentTaskID
		if parent, perr := s.Repository.ProjectTaskByID(parentTaskID); perr == nil {
			result["parentTaskTitle"] = parent["title"]
		}
	} else {
		result["taskType"] = "task"
	}

	crmProjectID := projectIDFromRow(task)
	result["projectId"] = crmProjectID
	if proj, perr := s.Repository.CrmProjectByID(crmProjectID); perr == nil {
		result["projectTitle"] = proj["title"]
	}

	return result, nil
}

// timeFromRow reads a DB-scanned timestamp column as *time.Time — used when
// falling back to an existing log's stored started_at/ended_at during a
// partial manual-log edit.
func timeFromRow(v interface{}) *time.Time {
	t, ok := v.(time.Time)
	if !ok {
		return nil
	}
	return &t
}

// resolveManualLogActor is the shared "who's actually performing this
// write, and what's their display name" resolution for all three manual-log
// mutations — falls back to resolving userID's own name when no distinct
// actor name comes back (e.g. actorUserId wasn't sent), same pattern
// ClockInTask/ClockOutTask already use.
func (s *pmService) resolveManualLogActor(actorUserIDRaw interface{}, fallbackUserID int64) (*int64, string) {
	actorUserID, actorName := s.resolveActor(actorSource(map[string]interface{}{"actorUserId": actorUserIDRaw}, ""))
	if actorUserID == nil {
		actorUserID = &fallbackUserID
	}
	if actorName == "" || actorName == "System" {
		if name, err := s.Repository.ResolveUserName(*actorUserID); err == nil && name != "" {
			actorName = name
		}
	}
	return actorUserID, actorName
}

// CreateManualTimeLog logs an already-closed session for the case the user
// forgot to clock in/out — same table as ClockIn/ClockOut (source='manual'
// instead of 'realtime'), but no "one active session" constraint applies
// (that's specific to *open* sessions); it's checked instead for double-
// booking against ANY of the user's existing sessions, open or closed, via
// ManualLogOverlapExists.
func (s *pmService) CreateManualTimeLog(taskID string, body map[string]interface{}, actorUserIDRaw interface{}) (model.Row, error) {
	if !validUUID(taskID) {
		return nil, fmt.Errorf("invalid task id")
	}
	userID := int64FromAny(body["userId"], 0)
	if userID == 0 {
		return nil, fmt.Errorf("userId is required")
	}
	note := strings.TrimSpace(strFromAny(body["note"], ""))
	if note == "" {
		return nil, fmt.Errorf("note is required for a manual log")
	}
	startedAt := parseDateTime(strPtrFromAny(body["startedAt"]))
	endedAt := parseDateTime(strPtrFromAny(body["endedAt"]))
	if startedAt == nil || endedAt == nil {
		return nil, fmt.Errorf("startedAt and endedAt are required")
	}
	if !startedAt.Before(*endedAt) {
		return nil, fmt.Errorf("startedAt must be before endedAt")
	}

	overlap, err := s.Repository.ManualLogOverlapExists(userID, *startedAt, *endedAt, nil)
	if err != nil {
		return nil, err
	}
	if overlap {
		return nil, fmt.Errorf("this time range overlaps an existing log for this user")
	}

	task, err := s.Repository.ProjectTaskByID(taskID)
	if err != nil {
		return nil, err
	}
	projectID := projectIDFromRow(task)
	duration := int64(endedAt.Sub(*startedAt).Seconds())

	actorUserID, actorName := s.resolveManualLogActor(actorUserIDRaw, userID)
	row, err := s.Repository.InsertManualTimeLog(map[string]interface{}{
		"task_id":          taskID,
		"project_id":       projectID,
		"user_id":          userID,
		"started_at":       *startedAt,
		"ended_at":         *endedAt,
		"duration_seconds": duration,
		"note":             note,
		"source":           "manual",
		"created_by":       *actorUserID,
		"updated_by":       *actorUserID,
	})
	if err != nil {
		return nil, err
	}

	desc := fmt.Sprintf("%s added a manual work log (%s)", actorName, formatDurationShort(duration))
	if *actorUserID != userID {
		if ownerName, nerr := s.Repository.ResolveUserName(userID); nerr == nil && ownerName != "" {
			desc = fmt.Sprintf("%s added a manual work log for %s (%s)", actorName, ownerName, formatDurationShort(duration))
		}
	}
	s.logTaskActivity(taskID, projectID, actorUserID, actorName, "manual_log_created", "", nil, nil, desc)

	return row, nil
}

// UpdateManualTimeLog edits an existing manual log's time range/note/owner.
// Only source='manual' rows are editable — a real-time session's
// started_at/ended_at are a factual record of when the user actually
// clocked in/out, not something to hand-edit after the fact.
func (s *pmService) UpdateManualTimeLog(logID int64, body map[string]interface{}, actorUserIDRaw interface{}) (model.Row, error) {
	before, err := s.Repository.TimeLogByID(logID)
	if err != nil {
		return nil, err
	}
	if str(before["source"]) != "manual" {
		return nil, fmt.Errorf("only manual logs can be edited")
	}

	startedAt := parseDateTime(strPtrFromAny(body["startedAt"]))
	endedAt := parseDateTime(strPtrFromAny(body["endedAt"]))
	effectiveStart := startedAt
	if effectiveStart == nil {
		effectiveStart = timeFromRow(before["started_at"])
	}
	effectiveEnd := endedAt
	if effectiveEnd == nil {
		effectiveEnd = timeFromRow(before["ended_at"])
	}
	if effectiveStart == nil || effectiveEnd == nil {
		return nil, fmt.Errorf("startedAt and endedAt are required")
	}
	if !effectiveStart.Before(*effectiveEnd) {
		return nil, fmt.Errorf("startedAt must be before endedAt")
	}

	userID := int64FromAny(body["userId"], 0)
	if userID == 0 {
		userID, _ = toInt64(before["user_id"])
	}

	overlap, err := s.Repository.ManualLogOverlapExists(userID, *effectiveStart, *effectiveEnd, &logID)
	if err != nil {
		return nil, err
	}
	if overlap {
		return nil, fmt.Errorf("this time range overlaps an existing log for this user")
	}

	fields := map[string]interface{}{}
	if startedAt != nil {
		fields["started_at"] = *startedAt
	}
	if endedAt != nil {
		fields["ended_at"] = *endedAt
	}
	if startedAt != nil || endedAt != nil {
		fields["duration_seconds"] = int64(effectiveEnd.Sub(*effectiveStart).Seconds())
	}
	if _, ok := body["userId"]; ok && userID > 0 {
		fields["user_id"] = userID
	}
	if noteRaw, ok := body["note"]; ok {
		note := strings.TrimSpace(strFromAny(noteRaw, ""))
		if note == "" {
			return nil, fmt.Errorf("note is required for a manual log")
		}
		fields["note"] = note
	}

	actorUserID, actorName := s.resolveManualLogActor(actorUserIDRaw, userID)
	fields["updated_by"] = *actorUserID

	if len(fields) > 0 {
		if err := s.Repository.UpdateTimeLog(logID, fields); err != nil {
			return nil, err
		}
	}
	after, err := s.Repository.TimeLogByID(logID)
	if err != nil {
		return nil, err
	}

	taskID := str(after["task_id"])
	projectID := projectIDFromRow(after)
	s.logTaskActivity(taskID, projectID, actorUserID, actorName, "manual_log_updated", "", nil, nil,
		fmt.Sprintf("%s updated a manual work log", actorName))

	return after, nil
}

// DeleteManualTimeLog soft-deletes a manual log. Real-time (clock in/out)
// sessions have no delete endpoint wired to them — this only ever operates
// on source='manual' rows.
func (s *pmService) DeleteManualTimeLog(logID int64, actorUserIDRaw interface{}) error {
	before, err := s.Repository.TimeLogByID(logID)
	if err != nil {
		return err
	}
	if str(before["source"]) != "manual" {
		return fmt.Errorf("only manual logs can be deleted")
	}
	if err := s.Repository.DeleteTimeLog(logID); err != nil {
		return err
	}

	ownerID, _ := toInt64(before["user_id"])
	actorUserID, actorName := s.resolveManualLogActor(actorUserIDRaw, ownerID)
	s.logTaskActivity(str(before["task_id"]), projectIDFromRow(before), actorUserID, actorName, "manual_log_deleted", "", nil, nil,
		fmt.Sprintf("%s deleted a manual work log", actorName))
	return nil
}

// describeAlreadyClockedIn turns the bare ErrAlreadyClockedIn into a message
// naming the task the user is already on, when that can be resolved —
// falls back to the generic error if the lookup itself fails for any reason
// (never let a best-effort enrichment mask the real 409).
func (s *pmService) describeAlreadyClockedIn(userID int64, fallback error) error {
	active, err := s.Repository.ActiveTimeLogForUser(userID)
	if err != nil || active == nil {
		return fallback
	}
	task, err := s.Repository.ProjectTaskByID(str(active["task_id"]))
	if err != nil || task["title"] == nil {
		return fallback
	}
	return fmt.Errorf("already clocked in on \"%s\" — clock out there first: %w", str(task["title"]), fallback)
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

// ─── PM Dashboard summary (/v1/pm/dashboard/summary) ───────────────────────
//
// High-level portfolio monitoring for /pm/dashboard. Every number here is
// derived from PmPortfolioProjects/PmPortfolioTasks/PmActiveWorkSessions/
// PmWeeklySecondsByUser/PmMembersForWorkload/PmRecentActivity — no field is
// ever hardcoded or estimated without a documented rule. Response is a plain
// model.Row (matching every other PM endpoint's convention) built with the
// exact key names the frontend contract expects (see CLAUDE.md).
//
// Stage is read straight off PmPortfolioProjects' `stage` column (the same
// taskStageCaseSQL CASE used by CrmProjects/CrmProjectByID) — never
// re-derived here, so /pm/dashboard and /pm/projects can never disagree on
// a project's Stage. "In Progress" was previously labeled "Fieldwork";
// taskStageCaseSQL already only ever emits "In Progress" (see the 2026-07-22
// rename note on that const) so no normalization is needed on this side.
//
// Planned progress is new (no prior metric in this codebase derives it): the
// percentage of a project's plan_start->plan_end window that has elapsed as
// of now, clamped 0-100. A project with no plan window at all (brand new,
// no tasks, no CRM start/end dates) reports 0 rather than being excluded,
// so portfolio averages still divide by every project consistently.
//
// Actual progress reuses computeTaskProgress/projectProgressFromTasks
// verbatim (the same helpers CrmProjects/CrmProjectDetail/TasksByCrmProject
// use) — the single source of truth for "how much of this project's active
// work is actually done".
func (s *pmService) DashboardSummary() (model.Row, error) {
	projects, err := s.Repository.PmPortfolioProjects()
	if err != nil {
		return nil, err
	}
	tasks, err := s.Repository.PmPortfolioTasks()
	if err != nil {
		tasks = []model.Row{}
	}
	members, err := s.Repository.PmMembersForWorkload()
	if err != nil {
		members = []model.Row{}
	}
	// Graceful fallback per the backend requirements: if the time-log or
	// activity-log tables/queries are ever unavailable, the dashboard still
	// renders with zeroed/empty sections instead of a hard 500.
	activeSessions, err := s.Repository.PmActiveWorkSessions()
	if err != nil {
		activeSessions = []model.Row{}
	}
	weeklySeconds, err := s.Repository.PmWeeklySecondsByUser()
	if err != nil {
		weeklySeconds = []model.Row{}
	}
	recentActivity, err := s.Repository.PmRecentActivity(10)
	if err != nil {
		recentActivity = []model.Row{}
	}

	now := time.Now()

	// Only PM-portfolio (Advisory/Audit Program) projects' own tasks count
	// toward stage/health/progress/attention — a task somehow pointing at a
	// crm_project_id outside that scope (shouldn't happen, but PmPortfolioTasks
	// is intentionally unscoped for reuse) never leaks into these metrics.
	scopedProjectIDs := make(map[int64]bool, len(projects))
	projectTitleByID := make(map[int64]string, len(projects))
	for _, p := range projects {
		pid, _ := toInt64(p["crm_project_id"])
		scopedProjectIDs[pid] = true
		projectTitleByID[pid] = str(p["title"])
	}

	tasksByProject := make(map[int64][]model.Row, len(projects))
	for _, t := range tasks {
		pid, _ := toInt64(t["crm_project_id"])
		if !scopedProjectIDs[pid] {
			continue
		}
		tasksByProject[pid] = append(tasksByProject[pid], t)
	}

	// weeklySecondsByUser / activeSessionByUser index the two time-log
	// queries by user id — activeSessions is guaranteed at-most-one-per-user
	// by the DB's unique partial index (migration 013), so a plain map
	// assignment (not append) is safe.
	weeklySecondsByUser := make(map[int64]int64, len(weeklySeconds))
	for _, row := range weeklySeconds {
		uid, _ := toInt64(row["user_id"])
		seconds, _ := toInt64(row["total_seconds"])
		weeklySecondsByUser[uid] = seconds
	}
	activeSessionByUser := make(map[int64]model.Row, len(activeSessions))
	for _, row := range activeSessions {
		uid, _ := toInt64(row["user_id"])
		activeSessionByUser[uid] = row
	}

	type projectMetrics struct {
		row              model.Row
		stage            string
		health           string
		actualProgress   int64
		plannedProgress  int64
		overdueTaskCount int64
		issue            string
		issueRank        int
	}

	metrics := make([]projectMetrics, 0, len(projects))
	var totalOpenTasks, totalDoneTasksAllProjects, totalTasksAllProjects int64
	var sumPlanned, sumActual int64

	for _, p := range projects {
		pid, _ := toInt64(p["crm_project_id"])
		projectTasks := tasksByProject[pid]

		stage := str(p["stage"])
		blockedTasks, _ := toInt64(p["blocked_tasks"])
		totalTasks, _ := toInt64(p["total_tasks"])
		doneTasks, _ := toInt64(p["done_tasks"])

		progress := computeTaskProgress(projectTasks)
		actualProgress := projectProgressFromTasks(projectTasks, progress)
		plannedProgress := planProgressPercent(timeFromRow(p["plan_start"]), timeFromRow(p["plan_end"]), now)

		var overdueTaskCount int64
		for _, t := range projectTasks {
			if isOverdueTask(t, now) {
				overdueTaskCount++
			}
			if normalizeTaskStatusKey(str(t["status"])) != "done" {
				totalOpenTasks++
			}
		}
		totalDoneTasksAllProjects += doneTasks
		totalTasksAllProjects += totalTasks
		sumPlanned += plannedProgress
		sumActual += actualProgress

		planEndPassed := false
		if planEnd := timeFromRow(p["plan_end"]); planEnd != nil && isPastCalendarDay(*planEnd, now) {
			planEndPassed = true
		}

		health := deriveProjectHealth(stage, blockedTasks, overdueTaskCount, planEndPassed, actualProgress, plannedProgress)

		// Issue priority for Attention Needed: Blocked > Overdue > Behind
		// plan > No recent update. Only one issue is surfaced per project
		// (the most critical applicable one) — a project can technically
		// match more than one, but showing the single worst signal is what
		// keeps the table scannable.
		issue, issueRank := "", -1
		updatedAt := timeFromRow(p["updated_at"])
		staleUpdate := updatedAt != nil && now.Sub(*updatedAt) > 14*24*time.Hour
		switch {
		case blockedTasks > 0 || stage == "Blocked":
			issue, issueRank = "Blocked", 0
		case overdueTaskCount > 0 || (planEndPassed && stage != "Completed"):
			issue, issueRank = "Overdue", 1
		case actualProgress < plannedProgress && stage != "Completed":
			issue, issueRank = "Behind plan", 2
		case staleUpdate && stage != "Completed":
			issue, issueRank = "No recent update", 3
		}

		metrics = append(metrics, projectMetrics{
			row: p, stage: stage, health: health,
			actualProgress: actualProgress, plannedProgress: plannedProgress,
			overdueTaskCount: overdueTaskCount, issue: issue, issueRank: issueRank,
		})
	}

	// ── Summary cards ──────────────────────────────────────────────────
	var activeProjects, completedProjects, blockedProjects, overdueProjects int64
	peopleSet := map[int64]bool{}
	for _, t := range tasks {
		pid, _ := toInt64(t["crm_project_id"])
		if !scopedProjectIDs[pid] {
			continue
		}
		if uid, ok := toInt64(t["assigned_to"]); ok && uid != 0 {
			peopleSet[uid] = true
		}
	}
	for _, m := range metrics {
		switch m.stage {
		case "Completed":
			completedProjects++
		case "Blocked":
			blockedProjects++
		default:
			activeProjects++
		}
		if m.overdueTaskCount > 0 || m.issue == "Overdue" {
			overdueProjects++
		}
	}
	var totalWeeklySeconds int64
	for _, seconds := range weeklySecondsByUser {
		totalWeeklySeconds += seconds
	}

	summary := model.Row{
		"totalProjects":     int64(len(projects)),
		"activeProjects":    activeProjects,
		"completedProjects": completedProjects,
		"blockedProjects":   blockedProjects,
		"overdueProjects":   overdueProjects,
		"totalOpenTasks":    totalOpenTasks,
		"peopleInvolved":    int64(len(peopleSet)),
		"workHoursThisWeek": roundTo1(float64(totalWeeklySeconds) / 3600.0),
	}

	// ── Stage / Health overview ────────────────────────────────────────
	stageCounts := map[string]int64{"Planning": 0, "In Progress": 0, "Review": 0, "Blocked": 0, "Completed": 0}
	healthCounts := map[string]int64{"Healthy": 0, "At Risk": 0, "Blocked": 0}
	for _, m := range metrics {
		if _, ok := stageCounts[m.stage]; ok {
			stageCounts[m.stage]++
		}
		healthCounts[m.health]++
	}
	stageOrder := []string{"Planning", "In Progress", "Review", "Blocked", "Completed"}
	stageOverview := make([]model.Row, 0, len(stageOrder))
	for _, st := range stageOrder {
		stageOverview = append(stageOverview, model.Row{"stage": st, "count": stageCounts[st]})
	}
	healthOrder := []string{"Healthy", "At Risk", "Blocked"}
	healthOverview := make([]model.Row, 0, len(healthOrder))
	for _, h := range healthOrder {
		healthOverview = append(healthOverview, model.Row{"health": h, "count": healthCounts[h]})
	}

	// ── Portfolio progress ─────────────────────────────────────────────
	var avgPlanned, avgActual float64
	if len(metrics) > 0 {
		avgPlanned = float64(sumPlanned) / float64(len(metrics))
		avgActual = float64(sumActual) / float64(len(metrics))
	}
	portfolioProgress := model.Row{
		"averagePlannedProgress": roundTo1(avgPlanned),
		"averageActualProgress":  roundTo1(avgActual),
		"averageVariance":        roundTo1(avgActual - avgPlanned),
		"completedTasks":         totalDoneTasksAllProjects,
		"totalTasks":             totalTasksAllProjects,
	}

	// ── Attention Needed (top 10, most critical first) ─────────────────
	attention := make([]projectMetrics, 0, len(metrics))
	for _, m := range metrics {
		if m.issue != "" {
			attention = append(attention, m)
		}
	}
	sort.SliceStable(attention, func(i, j int) bool { return attention[i].issueRank < attention[j].issueRank })
	if len(attention) > 10 {
		attention = attention[:10]
	}
	attentionNeeded := make([]model.Row, 0, len(attention))
	for _, m := range attention {
		pid, _ := toInt64(m.row["crm_project_id"])
		attentionNeeded = append(attentionNeeded, model.Row{
			"projectId":    pid,
			"projectTitle": m.row["title"],
			"picName":      strFromAny(m.row["pic"], strFromAny(m.row["owner"], strFromAny(m.row["owner_name"], ""))),
			"stage":        m.stage,
			"progress":     m.actualProgress,
			"deadline":     m.row["plan_end"],
			"issue":        m.issue,
		})
	}

	// ── Team workload ───────────────────────────────────────────────────
	tasksByUser := make(map[int64][]model.Row, len(members))
	for _, t := range tasks {
		uid, ok := toInt64(t["assigned_to"])
		if !ok || uid == 0 {
			continue
		}
		tasksByUser[uid] = append(tasksByUser[uid], t)
	}
	teamWorkload := make([]model.Row, 0, len(members))
	for _, member := range members {
		uid, _ := toInt64(member["id"])
		userTasks := tasksByUser[uid]
		var active, inProgress, overdue int64
		for _, t := range userTasks {
			statusKey := normalizeTaskStatusKey(str(t["status"]))
			if statusKey != "done" {
				active++
			}
			if statusKey == "in_progress" {
				inProgress++
			}
			if isOverdueTask(t, now) {
				overdue++
			}
		}
		hoursThisWeek := roundTo1(float64(weeklySecondsByUser[uid]) / 3600.0)

		var currentClockIn interface{}
		if session, ok := activeSessionByUser[uid]; ok {
			taskType := "task"
			if toBool(session["is_subtask"]) {
				taskType = "subtask"
			}
			spid, _ := toInt64(session["project_id"])
			currentClockIn = model.Row{
				"taskId":       session["task_id"],
				"taskTitle":    session["task_title"],
				"projectId":    spid,
				"projectTitle": session["project_title"],
				"startedAt":    session["started_at"],
				"taskType":     taskType,
			}
		}

		teamWorkload = append(teamWorkload, model.Row{
			"userId":          uid,
			"memberName":      member["name"],
			"jobTitle":        strFromAny(member["job_title"], ""),
			"activeTasks":     active,
			"inProgressTasks": inProgress,
			"overdueTasks":    overdue,
			"hoursThisWeek":   hoursThisWeek,
			"currentClockIn":  currentClockIn,
			"loadStatus":      loadStatus(active, overdue, hoursThisWeek),
		})
	}

	// ── Active work sessions ────────────────────────────────────────────
	activeWorkSessions := make([]model.Row, 0, len(activeSessions))
	for _, session := range activeSessions {
		uid, _ := toInt64(session["user_id"])
		pid, _ := toInt64(session["project_id"])
		taskType := "task"
		if toBool(session["is_subtask"]) {
			taskType = "subtask"
		}
		activeWorkSessions = append(activeWorkSessions, model.Row{
			"userId":         uid,
			"memberName":     session["member_name"],
			"projectId":      pid,
			"projectTitle":   session["project_title"],
			"taskId":         session["task_id"],
			"taskTitle":      session["task_title"],
			"taskType":       taskType,
			"startedAt":      session["started_at"],
			"elapsedSeconds": elapsedSecondsFrom(session["started_at"]),
		})
	}

	// ── Upcoming deadlines (tasks only; capped at 15) ──────────────────
	type deadlineItem struct {
		row    model.Row
		due    time.Time
		status string
		rank   int
	}
	deadlineItems := make([]deadlineItem, 0)
	weekFromNow := now.Add(7 * 24 * time.Hour)
	for _, t := range tasks {
		if normalizeTaskStatusKey(str(t["status"])) == "done" {
			continue
		}
		due := timeFromRow(t["deadline"])
		if due == nil {
			continue
		}
		var status string
		var rank int
		switch {
		case isPastCalendarDay(*due, now):
			status, rank = "Overdue", 0
		case isSameCalendarDay(*due, now):
			status, rank = "Due Today", 1
		case due.Before(weekFromNow):
			status, rank = "Due This Week", 2
		default:
			continue
		}
		deadlineItems = append(deadlineItems, deadlineItem{row: t, due: *due, status: status, rank: rank})
	}
	sort.SliceStable(deadlineItems, func(i, j int) bool {
		if deadlineItems[i].rank != deadlineItems[j].rank {
			return deadlineItems[i].rank < deadlineItems[j].rank
		}
		return deadlineItems[i].due.Before(deadlineItems[j].due)
	})
	if len(deadlineItems) > 15 {
		deadlineItems = deadlineItems[:15]
	}
	upcomingDeadlines := make([]model.Row, 0, len(deadlineItems))
	for _, d := range deadlineItems {
		pid, _ := toInt64(d.row["crm_project_id"])
		itemType := "task"
		if str(d.row["parent_task_id"]) != "" {
			itemType = "subtask"
		}
		upcomingDeadlines = append(upcomingDeadlines, model.Row{
			"type":         itemType,
			"itemId":       d.row["id"],
			"itemTitle":    d.row["title"],
			"projectId":    pid,
			"projectTitle": projectTitleByID[pid],
			"ownerName":    strFromAny(d.row["assignee_name"], ""),
			"dueDate":      d.row["deadline"],
			"status":       d.status,
		})
	}

	// ── Recent activity ────────────────────────────────────────────────
	recent := make([]model.Row, 0, len(recentActivity))
	for _, a := range recentActivity {
		pid, _ := toInt64(a["project_id"])
		recent = append(recent, model.Row{
			"id":          a["id"],
			"actorName":   strFromAny(a["actor_name"], "System"),
			"action":      a["action"],
			"description": a["description"],
			"projectId":   pid,
			"taskId":      a["task_id"],
			"createdAt":   a["created_at"],
		})
	}

	return model.Row{
		"summary":            summary,
		"stageOverview":      stageOverview,
		"healthOverview":     healthOverview,
		"portfolioProgress":  portfolioProgress,
		"attentionNeeded":    attentionNeeded,
		"teamWorkload":       teamWorkload,
		"activeWorkSessions": activeWorkSessions,
		"upcomingDeadlines":  upcomingDeadlines,
		"recentActivity":     recent,
	}, nil
}

// ─── Team Workload period filter (/v1/pm/dashboard/team-workload) ─────────
//
// TeamWorkloadByPeriod is DashboardSummary's teamWorkload section, widened
// to accept a period instead of always being the current calendar week —
// see CLAUDE.md for the full requirements writeup. Per-member fields and
// their period-sensitivity (also documented in CLAUDE.md):
//   - activeTasks / inProgressTasks / overdueTasks / currentClockIn: always
//     CURRENT snapshot, never historical — the task explicitly asked for
//     this ("agar dashboard tetap actionable"): "how much is on this
//     person's plate right now" doesn't make sense as a stale July number
//     while looking at a June report.
//   - completedTasks / hoursLogged / averageHoursPerDay: scoped to the
//     selected period.
//   - loadStatus: uses the period's hoursLogged against weekly or monthly
//     thresholds depending on periodKind (see loadStatusForPeriod).
func (s *pmService) TeamWorkloadByPeriod(periodType, monthParam string) (model.Row, error) {
	now := time.Now()
	start, end, label, periodKind, isOngoing, err := resolvePeriodRange(periodType, monthParam, now)
	if err != nil {
		return nil, err
	}

	members, err := s.Repository.PmMembersForWorkload()
	if err != nil {
		members = []model.Row{}
	}
	tasks, err := s.Repository.PmPortfolioTasks()
	if err != nil {
		tasks = []model.Row{}
	}
	activeSessions, err := s.Repository.PmActiveWorkSessions()
	if err != nil {
		activeSessions = []model.Row{}
	}
	periodSeconds, err := s.Repository.PmWorkloadSecondsByUserForRange(start, end)
	if err != nil {
		periodSeconds = []model.Row{}
	}
	periodCompleted, err := s.Repository.PmCompletedTasksByUserForRange(start, end)
	if err != nil {
		periodCompleted = []model.Row{}
	}

	tasksByUser := make(map[int64][]model.Row, len(members))
	for _, t := range tasks {
		uid, ok := toInt64(t["assigned_to"])
		if !ok || uid == 0 {
			continue
		}
		tasksByUser[uid] = append(tasksByUser[uid], t)
	}
	secondsByUser := make(map[int64]int64, len(periodSeconds))
	for _, row := range periodSeconds {
		uid, _ := toInt64(row["user_id"])
		seconds, _ := toInt64(row["period_seconds"])
		secondsByUser[uid] = seconds
	}
	completedByUser := make(map[int64]int64, len(periodCompleted))
	for _, row := range periodCompleted {
		uid, _ := toInt64(row["user_id"])
		count, _ := toInt64(row["completed_tasks"])
		completedByUser[uid] = count
	}
	activeSessionByUser := make(map[int64]model.Row, len(activeSessions))
	for _, row := range activeSessions {
		uid, _ := toInt64(row["user_id"])
		activeSessionByUser[uid] = row
	}

	// workdaysElapsed: for a period still in progress (isOngoing), count
	// Mon-Fri only up to and including today; for a fully-elapsed past
	// custom_month, count every weekday in the whole month (`end` is
	// already the exclusive month boundary in that case, so no `now` cap
	// is needed).
	workdaysEndExclusive := end
	if isOngoing {
		workdaysEndExclusive = now.AddDate(0, 0, 1)
	}
	workdaysElapsed := countWeekdays(start, workdaysEndExclusive)

	teamWorkload := make([]model.Row, 0, len(members))
	for _, member := range members {
		uid, _ := toInt64(member["id"])
		userTasks := tasksByUser[uid]
		var active, inProgress, overdue int64
		for _, t := range userTasks {
			statusKey := normalizeTaskStatusKey(str(t["status"]))
			if statusKey != "done" {
				active++
			}
			if statusKey == "in_progress" {
				inProgress++
			}
			if isOverdueTask(t, now) {
				overdue++
			}
		}

		periodHours := roundTo1(float64(secondsByUser[uid]) / 3600.0)
		var avgHoursPerDay float64
		if workdaysElapsed > 0 {
			avgHoursPerDay = roundTo1(periodHours / float64(workdaysElapsed))
		}

		var currentClockIn interface{}
		if session, ok := activeSessionByUser[uid]; ok {
			taskType := "task"
			if toBool(session["is_subtask"]) {
				taskType = "subtask"
			}
			spid, _ := toInt64(session["project_id"])
			currentClockIn = model.Row{
				"taskId":       session["task_id"],
				"taskTitle":    session["task_title"],
				"projectId":    spid,
				"projectTitle": session["project_title"],
				"startedAt":    session["started_at"],
				"taskType":     taskType,
			}
		}

		teamWorkload = append(teamWorkload, model.Row{
			"userId":          uid,
			"memberName":      member["name"],
			"jobTitle":        strFromAny(member["job_title"], ""),
			"activeTasks":     active,
			"inProgressTasks": inProgress,
			"completedTasks":  completedByUser[uid],
			"overdueTasks":    overdue,
			"hoursLogged":     periodHours,
			"avgHoursPerDay":  avgHoursPerDay,
			"currentClockIn":  currentClockIn,
			"loadStatus":      loadStatusForPeriod(active, overdue, periodHours, periodKind),
		})
	}

	return model.Row{
		"period": model.Row{
			"type":            periodType,
			"label":           label,
			"startDate":       start,
			"endDate":         end,
			"workdaysElapsed": workdaysElapsed,
		},
		"teamWorkload": teamWorkload,
	}, nil
}

// resolvePeriodRange turns a period type (+ optional YYYY-MM month for
// custom_month) into a concrete [start, end) query range, a display label,
// whether monthly or weekly loadStatus thresholds apply, and whether the
// period is still "ongoing" (end == now, so workday-elapsed counting should
// stop at today) vs a fully-elapsed past (or, edge case, future) month.
//   - this_week: Monday 00:00 of the current week -> now. Always ongoing.
//   - this_month: the 1st of the current month 00:00 -> now. Always ongoing.
//   - custom_month: the 1st of the requested month 00:00 -> either `now`
//     (ongoing, if the requested month IS the current month) or the 1st of
//     the FOLLOWING month (not ongoing — a past, or edge-case future, month
//     is treated as fully elapsed), i.e. the full calendar month.
func resolvePeriodRange(periodType, monthParam string, now time.Time) (start, end time.Time, label, periodKind string, isOngoing bool, err error) {
	switch periodType {
	case "", "this_week":
		start = startOfWeek(now)
		end = now
		label = "This Week"
		periodKind = "weekly"
		isOngoing = true
	case "this_month":
		start = startOfMonth(now)
		end = now
		label = now.Format("January 2006")
		periodKind = "monthly"
		isOngoing = true
	case "custom_month":
		if strings.TrimSpace(monthParam) == "" {
			return start, end, "", "", false, fmt.Errorf("%w: month is required for custom_month period (format YYYY-MM)", ErrInvalidWorkloadPeriod)
		}
		parsed, perr := time.ParseInLocation("2006-01", monthParam, now.Location())
		if perr != nil {
			return start, end, "", "", false, fmt.Errorf("%w: invalid month %q, expected YYYY-MM", ErrInvalidWorkloadPeriod, monthParam)
		}
		start = time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, now.Location())
		nextMonth := start.AddDate(0, 1, 0)
		if start.Year() == now.Year() && start.Month() == now.Month() {
			end = now
			isOngoing = true
		} else {
			end = nextMonth
			isOngoing = false
		}
		label = start.Format("January 2006")
		periodKind = "monthly"
	default:
		return start, end, "", "", false, fmt.Errorf("%w: unknown period %q (expected this_week, this_month, or custom_month)", ErrInvalidWorkloadPeriod, periodType)
	}
	return start, end, label, periodKind, isOngoing, nil
}

// startOfWeek returns Monday 00:00 of t's ISO week (Mon-Fri are the only
// days that ever count as "workdays" in this feature).
func startOfWeek(t time.Time) time.Time {
	daysSinceMonday := (int(t.Weekday()) + 6) % 7 // Sunday=0 -> 6, Monday=1 -> 0, ...
	d := t.AddDate(0, 0, -daysSinceMonday)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, t.Location())
}

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// countWeekdays counts Mon-Fri calendar days in [startInclusive, endExclusive).
func countWeekdays(startInclusive, endExclusive time.Time) int {
	d := time.Date(startInclusive.Year(), startInclusive.Month(), startInclusive.Day(), 0, 0, 0, 0, startInclusive.Location())
	endDay := time.Date(endExclusive.Year(), endExclusive.Month(), endExclusive.Day(), 0, 0, 0, 0, endExclusive.Location())
	count := 0
	for d.Before(endDay) {
		if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday {
			count++
		}
		d = d.AddDate(0, 0, 1)
	}
	return count
}

// loadStatusForPeriod is loadStatus's period-aware sibling — same priority
// order (heaviest signal wins, checked first), but the hours thresholds
// scale with periodKind ("weekly" thresholds for this_week; "monthly" —
// roughly 4x — for this_month/custom_month, per the task's explicit rule
// table). Deliberately a separate function from loadStatus (used by
// DashboardSummary, always weekly) rather than a shared one with a bypass
// flag — keeps the always-weekly summary card path simple and unable to
// accidentally regress if this one changes.
func loadStatusForPeriod(activeTasks, overdueTasks int64, hoursLogged float64, periodKind string) string {
	heavyHours, moderateHours := 35.0, 20.0
	if periodKind == "monthly" {
		heavyHours, moderateHours = 140.0, 80.0
	}
	switch {
	case activeTasks > 5 || hoursLogged > heavyHours || overdueTasks > 0:
		return "Berat"
	case activeTasks >= 3 || hoursLogged >= moderateHours:
		return "Sedang"
	default:
		return "Ringan"
	}
}

// planProgressPercent is the "planned progress" side of Portfolio Progress:
// how far through a project's plan_start->plan_end window `now` falls,
// clamped 0-100. Returns 0 if either bound is missing or the window is
// zero/negative-length (can't divide by zero, and a same-day plan window
// isn't meaningful to interpolate) — a project with no usable plan window
// simply contributes 0 rather than being excluded from portfolio averages.
// deriveProjectHealth is the single source of truth for a project's Health
// status — Blocked > At Risk > Healthy, checked in that order so a project
// only ever reports its single worst signal. Used by both DashboardSummary
// (via PmPortfolioProjects) and CrmProjects, so the two list/detail/
// dashboard surfaces can never disagree about a project's health:
//   - Blocked: any active task is blocked, or Stage itself is already
//     "Blocked" (the two are usually the same signal, kept as an OR for
//     safety in case Stage and the live blocked-task count ever drift).
//   - At Risk: any task is overdue, OR the plan window's end date has
//     already passed while the project isn't Completed yet, OR actual
//     progress (tasks really done) is behind planned progress (where the
//     plan window's start/end date says it should be by now).
//   - Healthy: none of the above.
func deriveProjectHealth(stage string, blockedTasks, overdueTaskCount int64, planEndPassed bool, actualProgress, plannedProgress int64) string {
	switch {
	case blockedTasks > 0 || stage == "Blocked":
		return "Blocked"
	case overdueTaskCount > 0 || (planEndPassed && stage != "Completed") || actualProgress < plannedProgress:
		return "At Risk"
	default:
		return "Healthy"
	}
}

func planProgressPercent(planStart, planEnd *time.Time, now time.Time) int64 {
	if planStart == nil || planEnd == nil {
		return 0
	}
	total := planEnd.Sub(*planStart)
	if total <= 0 {
		return 0
	}
	if now.Before(*planStart) {
		return 0
	}
	if !now.Before(*planEnd) {
		return 100
	}
	elapsed := now.Sub(*planStart)
	pct := int64((elapsed.Seconds() / total.Seconds()) * 100)
	return clampPercent(pct)
}

// isOverdueTask: has a deadline whose calendar day has already fully passed
// and isn't done yet. Mirrors the LOWER(status) tolerance taskStageJoinSQL
// uses elsewhere in this file, via normalizeTaskStatusKey (folds 'completed'
// into 'done'). Calendar-day (not exact-timestamp) comparison matters
// because task deadlines are stored as midnight-of-day values — an exact
// due.Before(now) would flag a task as "overdue" the instant any time at
// all has passed today, making "Due Today" effectively unreachable.
func isOverdueTask(t model.Row, now time.Time) bool {
	if normalizeTaskStatusKey(str(t["status"])) == "done" {
		return false
	}
	due := timeFromRow(t["deadline"])
	return due != nil && isPastCalendarDay(*due, now)
}

// isPastCalendarDay/isSameCalendarDay compare only the Y/M/D of two
// timestamps (ignoring time-of-day) — see isOverdueTask's doc comment for
// why this matters given midnight-stored deadlines.
func isPastCalendarDay(t, now time.Time) bool {
	ty, tm, td := t.Date()
	ny, nm, nd := now.Date()
	if ty != ny {
		return ty < ny
	}
	if tm != nm {
		return tm < nm
	}
	return td < nd
}

func isSameCalendarDay(t, now time.Time) bool {
	ty, tm, td := t.Date()
	ny, nm, nd := now.Date()
	return ty == ny && tm == nm && td == nd
}

// loadStatus implements the dashboard's initial Team Workload thresholds
// (Ringan/Sedang/Berat), checked heaviest-first so any single qualifying
// condition is enough to escalate — e.g. zero active tasks but 40 tracked
// hours this week (a big single task) still reports Berat.
func loadStatus(activeTasks, overdueTasks int64, hoursThisWeek float64) string {
	switch {
	case activeTasks > 5 || hoursThisWeek > 35 || overdueTasks > 0:
		return "Berat"
	case activeTasks >= 3 || hoursThisWeek >= 20:
		return "Sedang"
	default:
		return "Ringan"
	}
}

func roundTo1(v float64) float64 {
	return math.Round(v*10) / 10
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
	case int16:
		// Postgres smallint columns (e.g. pm_project_tasks.progress_pct)
		// scan into map[string]interface{} as Go int16, not int64/int32 —
		// confirmed via runtime type check, not assumption.
		return int64(n), true
	case int:
		return int64(n), true
	}
	return 0, false
}

// ─── Task/project progress computation (single source of truth) ───────────
//
// Effective progress is never trusted purely from the stored progress_pct
// column — it's recomputed on every read from status + child tasks, so a
// row written before this feature shipped (or edited any other way) always
// self-corrects instead of showing a stale number. See CrmProjectDetail,
// CrmProjects, TasksByCrmProject, and the task write paths (CreateTask/
// UpdateTask/MoveTaskByKey below) for where this plugs in.

// taskProgress holds one task's derived effective progress/status plus
// whether it has any active direct subtasks. A task with subtasks is a
// "parent": its progress AND status are always derived from its children,
// never a manual value — see leafProgressForStatus (progress) and
// deriveParentStatus (status) for the two rule tables.
type taskProgress struct {
	Progress int64
	Status   string // normalized key: to_do/in_progress/in_review/blocked/done
	IsParent bool
}

// normalizeTaskStatusKey lowercases/trims a raw status value and folds
// 'completed' into 'done' — the one place both leafProgressForStatus and
// deriveParentStatus agree on what a status string actually means.
func normalizeTaskStatusKey(raw string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "completed" {
		return "done"
	}
	if key == "" {
		return "to_do"
	}
	return key
}

// deriveParentStatus is the status equivalent of leafProgressForStatus's
// progress rule, applied to a parent's direct children (already resolved to
// their own EFFECTIVE status — a nested parent contributes its own derived
// status, not its raw stored one). Priority (highest first): Blocked >
// In Review > In Progress > Done > To Do. A child counts toward "Done" if
// its own status is done OR its effective progress is 100 (matches the
// leaf rule where in_review can already sit at 100) — Done only fires once
// EVERY child qualifies that way, never partially.
func deriveParentStatus(children []taskProgress) string {
	total := len(children)
	if total == 0 {
		return "to_do"
	}
	var blocked, review, inProgress, complete int
	for _, c := range children {
		switch c.Status {
		case "blocked":
			blocked++
		case "in_review":
			review++
		case "in_progress":
			inProgress++
		}
		if c.Status == "done" || c.Progress >= 100 {
			complete++
		}
	}
	switch {
	case blocked > 0:
		return "blocked"
	case review > 0:
		return "in_review"
	case inProgress > 0:
		return "in_progress"
	case complete == total:
		return "done"
	default:
		return "to_do"
	}
}

// projectIDFromRow reads crm_project_id out of a DB-scanned model.Row —
// same toInt64-first-then-int64FromAny fallback already used by
// logTaskFieldChanges, since the driver can hand this back as a native
// int64 (toInt64) or, less commonly, something int64FromAny (float64/
// string) can parse instead.
func projectIDFromRow(row model.Row) int64 {
	if id, ok := toInt64(row["crm_project_id"]); ok && id != 0 {
		return id
	}
	return int64FromAny(row["crm_project_id"], 0)
}

// progressPctFromRow reads progress_pct out of a DB-scanned model.Row.
// Postgres smallint columns typically arrive as int64 via database/sql, but
// this tolerates int32/float64/string too rather than assuming one driver
// behavior.
func progressPctFromRow(v interface{}) int64 {
	if n, ok := toInt64(v); ok {
		return clampPercent(n)
	}
	switch n := v.(type) {
	case float64:
		return clampPercent(int64(n))
	case string:
		if i, err := strconv.ParseInt(n, 10, 64); err == nil {
			return clampPercent(i)
		}
	}
	return 0
}

// leafProgressForStatus is the single rule table for how a LEAF task's
// progress reacts to its status — shared by task create, task update, and
// Kanban drag-move, so the three can never diverge from each other.
// manualProgress is a value the caller explicitly wants to set on this call
// (nil if none, e.g. a plain drag-move never carries one); prevProgress is
// the task's progress before this change.
//
//	to_do       -> always 0
//	done        -> always 100 ('completed' is treated as a synonym)
//	blocked     -> never auto-advances; only a genuine manual edit changes it
//	in_review   -> floors at 90 (never lowers an already-higher value)
//	in_progress -> 10 if it was 0, otherwise left alone
//	anything else -> manual value if given, else unchanged
func leafProgressForStatus(statusKey string, prevProgress int64, manualProgress *int64) int64 {
	base := prevProgress
	if manualProgress != nil {
		base = clampPercent(*manualProgress)
	}
	switch strings.ToLower(strings.TrimSpace(statusKey)) {
	case "to_do":
		return 0
	case "done", "completed":
		return 100
	case "blocked":
		return base
	case "in_review":
		if base < 90 {
			return 90
		}
		return base
	case "in_progress":
		if base == 0 {
			return 10
		}
		return base
	default:
		return base
	}
}

// computeTaskProgress derives every task's EFFECTIVE progress AND status
// bottom-up: leaves come from leafProgressForStatus (progress) and their own
// stored status (normalized); a task with active children gets the rounded
// average of its children's own effective progress (leafProgressForStatus's
// "recursive" case) and deriveParentStatus's priority rule over its
// children's own effective status — computed recursively so nested subtasks
// fold correctly however deep the tree goes. `tasks` should be every active
// task sharing a scope (one project, or all projects at once for the list
// endpoint) — this is one pass over an already-fetched slice, never an extra
// query per task. A malformed/cyclic parent_task_id chain (never seen in
// practice) falls back to that task's own stored progress/status instead of
// recursing forever.
func computeTaskProgress(tasks []model.Row) map[string]taskProgress {
	childrenOf := make(map[string][]string, len(tasks))
	byID := make(map[string]model.Row, len(tasks))
	for _, t := range tasks {
		id := str(t["id"])
		byID[id] = t
		if pid := str(t["parent_task_id"]); pid != "" {
			childrenOf[pid] = append(childrenOf[pid], id)
		}
	}

	result := make(map[string]taskProgress, len(tasks))
	visiting := make(map[string]bool, len(tasks))

	var resolve func(id string) taskProgress
	resolve = func(id string) taskProgress {
		if r, ok := result[id]; ok {
			return r
		}
		t, known := byID[id]
		if !known {
			return taskProgress{Status: "to_do"}
		}
		if visiting[id] {
			return taskProgress{
				Progress: progressPctFromRow(t["progress_pct"]),
				Status:   normalizeTaskStatusKey(str(t["status"])),
			}
		}
		visiting[id] = true
		kids := childrenOf[id]
		if len(kids) == 0 {
			p := leafProgressForStatus(str(t["status"]), progressPctFromRow(t["progress_pct"]), nil)
			tp := taskProgress{Progress: p, Status: normalizeTaskStatusKey(str(t["status"])), IsParent: false}
			result[id] = tp
			visiting[id] = false
			return tp
		}
		var sum int64
		children := make([]taskProgress, 0, len(kids))
		for _, kid := range kids {
			c := resolve(kid)
			sum += c.Progress
			children = append(children, c)
		}
		avg := (sum + int64(len(kids))/2) / int64(len(kids)) // round-half-up
		tp := taskProgress{Progress: avg, Status: deriveParentStatus(children), IsParent: true}
		result[id] = tp
		visiting[id] = false
		return tp
	}

	for id := range byID {
		resolve(id)
	}
	return result
}

// statusMeta mirrors the frontend's KANBAN_COLS (title-cased label per
// status key) so a parent task's overridden status carries a correct
// display title even though it never went through the pm_task_statuses
// join that set the ORIGINAL status_title/status_color on the row.
var statusMeta = map[string]struct{ Title, Color string }{
	"to_do":       {"To Do", "#F9A52D"},
	"in_progress": {"In progress", "#1672B9"},
	"in_review":   {"In Review", "#6f42c1"},
	"blocked":     {"Blocked", "#D63939"},
	"done":        {"Done", "#00A679"},
}

// applyTaskProgress overrides tasks' progress_pct/status in place with their
// computed effective values and tags each with has_active_subtasks (so the
// frontend can disable manual editing for parent tasks), then returns the
// progress map — also used by callers to derive the owning project's
// overall progress via projectProgressFromTasks. A parent's status_title/
// status_color are overridden alongside status/status_key so the frontend's
// badge (which prefers status_title when present) never shows a stale label
// next to the new derived status.
func applyTaskProgress(tasks []model.Row) map[string]taskProgress {
	progress := computeTaskProgress(tasks)
	for _, t := range tasks {
		id := str(t["id"])
		p := progress[id]
		t["progress_pct"] = p.Progress
		t["has_active_subtasks"] = p.IsParent
		if p.IsParent {
			t["status"] = p.Status
			t["status_key"] = p.Status
			if meta, ok := statusMeta[p.Status]; ok {
				t["status_title"] = meta.Title
				t["status_color"] = meta.Color
			}
		}
	}
	return progress
}

// projectProgressFromTasks is the project-level progress rule: average
// effective progress of ROOT tasks (parent_task_id empty) only — subtasks
// are already folded into their parent's own effective value by
// computeTaskProgress, so averaging every task directly would double-count
// them. When no task has a parent at all, every task IS a root, so this
// reduces exactly to "average of all active tasks" — the documented
// no-hierarchy fallback. Zero active tasks -> 0.
func projectProgressFromTasks(tasks []model.Row, progress map[string]taskProgress) int64 {
	var sum, count int64
	for _, t := range tasks {
		if str(t["parent_task_id"]) != "" {
			continue
		}
		sum += progress[str(t["id"])].Progress
		count++
	}
	if count == 0 {
		return 0
	}
	return (sum + count/2) / count
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
	projectID := projectIDFromRow(before)
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
