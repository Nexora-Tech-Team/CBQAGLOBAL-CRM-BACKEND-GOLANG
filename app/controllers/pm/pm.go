package pm

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"erp-cbqa-global/domain/pm/model"
	"erp-cbqa-global/domain/pm/repository"
	"erp-cbqa-global/domain/pm/service"
	authLib "erp-cbqa-global/lib/auth"
	"erp-cbqa-global/lib/response"
)

type pmController struct {
	Service service.ServiceInterface
}

func PmController(db *gorm.DB) *pmController {
	return &pmController{
		Service: service.Service(repository.Repository(db)),
	}
}

func parseIDParam(ctx *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(ctx.Param(name), 10, 64)
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func parseInt64Query(ctx *gin.Context, name string) *int64 {
	raw := ctx.Query(name)
	if raw == "" {
		return nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	return &id
}

// currentUserID returns the acting user's id when auth middleware set one,
// or "" for the PM group, which intentionally runs without auth for now.
// Callers treat "" as anonymous rather than failing the request.
func currentUserID(ctx *gin.Context) (string, error) {
	authUser, err := authLib.GetAuthUserCtx(ctx)
	if err != nil {
		return "", nil
	}
	return authUser.ID, nil
}

func (pc *pmController) Dashboard(ctx *gin.Context) {
	data, err := pc.Service.Dashboard(parseInt64Query(ctx, "projectId"), parseInt64Query(ctx, "memberId"))
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) TaskStatuses(ctx *gin.Context) {
	data, err := pc.Service.Statuses()
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) Kanban(ctx *gin.Context) {
	data, err := pc.Service.Kanban()
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) Projects(ctx *gin.Context) {
	data, err := pc.Service.Projects(ctx.Query("search"), parseInt64Query(ctx, "client_id"), ctx.Query("status"))
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) Clients(ctx *gin.Context) {
	data, err := pc.Service.Clients(ctx.Query("search"))
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) Members(ctx *gin.Context) {
	data, err := pc.Service.Members(parseInt64Query(ctx, "project_id"), ctx.Query("search"))
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) Tasks(ctx *gin.Context) {
	data, err := pc.Service.Tasks(ctx.Query("search"), parseInt64Query(ctx, "project_id"), ctx.Query("assigned_to"))
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) CreateTask(ctx *gin.Context) {
	var req model.TaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	userID, err := currentUserID(ctx)
	if err != nil {
		response.Error(ctx, http.StatusUnauthorized, err.Error())
		return
	}
	data, err := pc.Service.CreateTask(req, userID)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

// UpdateTask handles PUT /tasks/:id for both the legacy int64-keyed pm_tasks
// board and the new UUID-keyed pm_project_tasks (CRM project) board, mirroring
// the same dual-dispatch pattern as MoveTaskByKey/DeleteTask below.
func (pc *pmController) UpdateTask(ctx *gin.Context) {
	idParam := ctx.Param("id")
	if id, convErr := strconv.ParseInt(idParam, 10, 64); convErr == nil {
		var req model.TaskRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			response.Error(ctx, http.StatusBadRequest, err.Error())
			return
		}
		userID, err := currentUserID(ctx)
		if err != nil {
			response.Error(ctx, http.StatusUnauthorized, err.Error())
			return
		}
		data, err := pc.Service.UpdateTask(id, req, userID)
		if err != nil {
			response.Error(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		ctx.JSON(http.StatusOK, data)
		return
	}

	var body map[string]interface{}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	data, err := pc.Service.UpdateProjectTask(idParam, body)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) MoveTaskStatus(ctx *gin.Context) {
	id, ok := parseIDParam(ctx, "id")
	if !ok {
		return
	}
	var req model.MoveTaskStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	userID, err := currentUserID(ctx)
	if err != nil {
		response.Error(ctx, http.StatusUnauthorized, err.Error())
		return
	}
	data, err := pc.Service.MoveTaskStatus(id, req.StatusID, userID)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

// MoveTaskByKey handles PATCH /tasks/:id/move for both the legacy int64-keyed
// pm_tasks board and the new UUID-keyed pm_project_tasks (CRM project) board,
// since the frontend uses the same path for both. It also tolerates the
// frontend's camelCase `statusKey` body alongside `status_key`.
func (pc *pmController) MoveTaskByKey(ctx *gin.Context) {
	idParam := ctx.Param("id")
	var body map[string]interface{}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	statusKey, _ := body["statusKey"].(string)
	if statusKey == "" {
		statusKey, _ = body["status_key"].(string)
	}
	if statusKey == "" {
		response.Error(ctx, http.StatusBadRequest, "status_key is required")
		return
	}

	if id, convErr := strconv.ParseInt(idParam, 10, 64); convErr == nil {
		userID, err := currentUserID(ctx)
		if err != nil {
			response.Error(ctx, http.StatusUnauthorized, err.Error())
			return
		}
		data, err := pc.Service.MoveTaskByKey(id, statusKey, userID)
		if err != nil {
			response.Error(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		ctx.JSON(http.StatusOK, data)
		return
	}

	data, err := pc.Service.MoveProjectTaskByKey(idParam, statusKey, body["actorUserId"])
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

// DeleteTask handles DELETE /tasks/:id for both the legacy int64-keyed
// pm_tasks board and the new UUID-keyed pm_project_tasks board. DELETE
// requests carry no JSON body from the frontend, so the acting user (for
// the activity log) is passed as a query param instead: ?actorUserId=67.
func (pc *pmController) DeleteTask(ctx *gin.Context) {
	idParam := ctx.Param("id")
	if id, convErr := strconv.ParseInt(idParam, 10, 64); convErr == nil {
		userID, err := currentUserID(ctx)
		if err != nil {
			response.Error(ctx, http.StatusUnauthorized, err.Error())
			return
		}
		if err := pc.Service.DeleteTask(id, userID); err != nil {
			response.Error(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		ctx.JSON(http.StatusOK, nil)
		return
	}

	if err := pc.Service.DeleteProjectTask(idParam, ctx.Query("actorUserId")); err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, nil)
}

// TaskActivity handles GET /tasks/:id/activity — the CRM-linked task board's
// audit trail (Task Detail drawer's "Activity" section).
func (pc *pmController) TaskActivity(ctx *gin.Context) {
	data, err := pc.Service.TaskActivityLogs(ctx.Param("id"))
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) CrmProjects(ctx *gin.Context) {
	data, err := pc.Service.CrmProjects(ctx.Query("search"))
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) CrmProjectDetail(ctx *gin.Context) {
	id, ok := parseIDParam(ctx, "id")
	if !ok {
		return
	}
	data, err := pc.Service.CrmProjectDetail(id)
	if err != nil {
		response.Error(ctx, http.StatusNotFound, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) UpdateCrmProjectOverview(ctx *gin.Context) {
	id, ok := parseIDParam(ctx, "id")
	if !ok {
		return
	}
	var body map[string]interface{}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	data, err := pc.Service.UpdateCrmProjectOverview(id, body)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) CrmProjectTasks(ctx *gin.Context) {
	id, ok := parseIDParam(ctx, "id")
	if !ok {
		return
	}
	data, err := pc.Service.TasksByCrmProject(id)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) CreateCrmProjectTask(ctx *gin.Context) {
	id, ok := parseIDParam(ctx, "id")
	if !ok {
		return
	}
	var body map[string]interface{}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	userID, err := currentUserID(ctx)
	if err != nil {
		response.Error(ctx, http.StatusUnauthorized, err.Error())
		return
	}
	data, err := pc.Service.CreateTaskForCrmProject(id, body, userID)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) CrmProjectMembers(ctx *gin.Context) {
	id, ok := parseIDParam(ctx, "id")
	if !ok {
		return
	}
	data, err := pc.Service.CrmProjectMembers(id)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) AddCrmProjectMember(ctx *gin.Context) {
	id, ok := parseIDParam(ctx, "id")
	if !ok {
		return
	}
	var body map[string]interface{}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	userID, err := currentUserID(ctx)
	if err != nil {
		response.Error(ctx, http.StatusUnauthorized, err.Error())
		return
	}
	data, err := pc.Service.AddCrmProjectMember(id, body, userID)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) UpdateCrmProjectMember(ctx *gin.Context) {
	id, ok := parseIDParam(ctx, "id")
	if !ok {
		return
	}
	memberID := ctx.Param("memberId")
	var body map[string]interface{}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	data, err := pc.Service.UpdateCrmProjectMember(id, memberID, body)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) DeleteCrmProjectMember(ctx *gin.Context) {
	id, ok := parseIDParam(ctx, "id")
	if !ok {
		return
	}
	memberID := ctx.Param("memberId")
	if err := pc.Service.DeleteCrmProjectMember(id, memberID); err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, nil)
}

func (pc *pmController) GanttMembers(ctx *gin.Context) {
	data, err := pc.Service.GanttMembers()
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

// DashboardSummary powers /pm/dashboard — the high-level PM portfolio
// monitoring dashboard (project health, portfolio progress, team workload,
// active work sessions, upcoming deadlines, recent activity).
func (pc *pmController) DashboardSummary(ctx *gin.Context) {
	data, err := pc.Service.DashboardSummary()
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

// TeamWorkloadByPeriod powers the Team Workload section's period filter —
// GET /dashboard/team-workload?period=this_week|this_month|custom_month
// (&month=YYYY-MM when period=custom_month). `period` defaults to
// this_week when omitted, matching the frontend's default selection.
func (pc *pmController) TeamWorkloadByPeriod(ctx *gin.Context) {
	period := ctx.DefaultQuery("period", "this_week")
	month := ctx.Query("month")
	data, err := pc.Service.TeamWorkloadByPeriod(period, month)
	if err != nil {
		if errors.Is(err, service.ErrInvalidWorkloadPeriod) {
			response.Error(ctx, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

// ClockInTask handles POST /tasks/:id/clock-in — body: { userId, actorUserId? }.
func (pc *pmController) ClockInTask(ctx *gin.Context) {
	taskID := ctx.Param("id")
	var body map[string]interface{}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	userID := int64FromBody(body["userId"])
	if userID == 0 {
		response.Error(ctx, http.StatusBadRequest, "userId is required")
		return
	}
	data, err := pc.Service.ClockInTask(taskID, userID, body["actorUserId"])
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyClockedIn) {
			response.Error(ctx, http.StatusConflict, err.Error())
			return
		}
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

// ClockOutTask handles POST /tasks/:id/clock-out — body: { userId, note?, actorUserId? }.
func (pc *pmController) ClockOutTask(ctx *gin.Context) {
	taskID := ctx.Param("id")
	var body map[string]interface{}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	userID := int64FromBody(body["userId"])
	if userID == 0 {
		response.Error(ctx, http.StatusBadRequest, "userId is required")
		return
	}
	var note *string
	if n, ok := body["note"].(string); ok && n != "" {
		note = &n
	}
	data, err := pc.Service.ClockOutTask(taskID, userID, note, body["actorUserId"])
	if err != nil {
		if errors.Is(err, repository.ErrNoActiveTimeLog) {
			response.Error(ctx, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

// TaskTimeLogs handles GET /tasks/:id/time-logs.
func (pc *pmController) TaskTimeLogs(ctx *gin.Context) {
	taskID := ctx.Param("id")
	var userID int64
	if raw := ctx.Query("userId"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			userID = parsed
		}
	}
	data, err := pc.Service.TaskTimeLogs(taskID, userID)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

// ActiveTimeLog handles GET /time-logs/active?userId=... — lets the
// frontend check "is this user clocked in somewhere" before it even opens
// a task (e.g. to show the lock state up front on a different task's drawer).
func (pc *pmController) ActiveTimeLog(ctx *gin.Context) {
	userID, ok := parseInt64QueryRequired(ctx, "userId")
	if !ok {
		return
	}
	data, err := pc.Service.ActiveTimeLogForUser(userID)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

// CreateManualTimeLog handles POST /tasks/:id/time-logs/manual — body:
// { userId, startedAt, endedAt, note, actorUserId? }.
func (pc *pmController) CreateManualTimeLog(ctx *gin.Context) {
	taskID := ctx.Param("id")
	var body map[string]interface{}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	data, err := pc.Service.CreateManualTimeLog(taskID, body, body["actorUserId"])
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

// UpdateTimeLog handles PATCH /time-logs/:id — body: any of
// { userId, startedAt, endedAt, note, actorUserId? }, all optional.
func (pc *pmController) UpdateTimeLog(ctx *gin.Context) {
	id, ok := parseIDParam(ctx, "id")
	if !ok {
		return
	}
	var body map[string]interface{}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	data, err := pc.Service.UpdateManualTimeLog(id, body, body["actorUserId"])
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

// DeleteTimeLog handles DELETE /time-logs/:id?actorUserId=... (DELETE
// requests carry no body from the frontend, same convention as DeleteTask).
func (pc *pmController) DeleteTimeLog(ctx *gin.Context) {
	id, ok := parseIDParam(ctx, "id")
	if !ok {
		return
	}
	if err := pc.Service.DeleteManualTimeLog(id, ctx.Query("actorUserId")); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, nil)
}

func parseInt64QueryRequired(ctx *gin.Context, name string) (int64, bool) {
	raw := ctx.Query(name)
	if raw == "" {
		response.Error(ctx, http.StatusBadRequest, name+" is required")
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, "invalid "+name)
		return 0, false
	}
	return id, true
}

// int64FromBody mirrors the service layer's int64FromAny for a JSON request
// body value (numbers decode as float64) — kept local to the controller so
// it doesn't need to reach into the service package's unexported helpers.
func int64FromBody(v interface{}) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case string:
		if i, err := strconv.ParseInt(n, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

func (pc *pmController) Tickets(ctx *gin.Context) {
	data, err := pc.Service.Tickets(ctx.Query("status"), ctx.Query("search"))
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) CreateTicket(ctx *gin.Context) {
	var req model.TicketRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	userID, err := currentUserID(ctx)
	if err != nil {
		response.Error(ctx, http.StatusUnauthorized, err.Error())
		return
	}
	data, err := pc.Service.CreateTicket(req, userID)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) TicketComments(ctx *gin.Context) {
	id, ok := parseIDParam(ctx, "ticketId")
	if !ok {
		return
	}
	data, err := pc.Service.TicketComments(id)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) AddTicketComment(ctx *gin.Context) {
	id, ok := parseIDParam(ctx, "ticketId")
	if !ok {
		return
	}
	var req model.TicketCommentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	userID, err := currentUserID(ctx)
	if err != nil {
		response.Error(ctx, http.StatusUnauthorized, err.Error())
		return
	}
	data, err := pc.Service.AddTicketComment(id, req, userID)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) TicketTemplates(ctx *gin.Context) {
	data, err := pc.Service.TicketTemplates()
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) ActivityLogs(ctx *gin.Context) {
	data, err := pc.Service.ActivityLogs()
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) Timesheets(ctx *gin.Context) {
	data, err := pc.Service.Timesheets(
		ctx.Query("search"),
		parseInt64Query(ctx, "user_id"),
		parseInt64Query(ctx, "crm_project_id"),
		ctx.Query("status"),
		ctx.Query("period"),
	)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) CreateTimesheet(ctx *gin.Context) {
	var req model.TimesheetRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	data, err := pc.Service.CreateTimesheet(req)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) UpdateTimesheetStatus(ctx *gin.Context) {
	id, ok := parseIDParam(ctx, "id")
	if !ok {
		return
	}
	var req model.TimesheetStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	data, err := pc.Service.UpdateTimesheetStatus(id, req.Status, req.ApprovedBy)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

func (pc *pmController) DeleteTimesheet(ctx *gin.Context) {
	id, ok := parseIDParam(ctx, "id")
	if !ok {
		return
	}
	if err := pc.Service.DeleteTimesheet(id); err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, nil)
}
