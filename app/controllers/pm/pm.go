package pm

import (
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

	data, err := pc.Service.MoveProjectTaskByKey(idParam, statusKey)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, data)
}

// DeleteTask handles DELETE /tasks/:id for both the legacy int64-keyed
// pm_tasks board and the new UUID-keyed pm_project_tasks board.
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

	if err := pc.Service.DeleteProjectTask(idParam); err != nil {
		response.Error(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, nil)
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
