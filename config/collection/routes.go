package collection

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"erp-cbqa-global/app/controllers/dashboard/auth"
	"erp-cbqa-global/app/controllers/dashboard/user"
	"erp-cbqa-global/app/controllers/pm"
	"erp-cbqa-global/config/middleware"
)

func Router(db *gorm.DB, main *gin.RouterGroup) {
	authCtrl := auth.AuthController(db)
	auth := main.Group("auth")
	{
		auth.POST("/login", authCtrl.Login)
		auth.GET("/logout", middleware.AuthDashboard(db), authCtrl.Logout)
		auth.POST("/activate-user", authCtrl.ActivateAccount)
		auth.POST("/forgot-password", authCtrl.ForgotPassword)
		auth.POST("reset-password", authCtrl.ResetPassword)
		auth.POST("/token-fcm", middleware.AuthDashboard(db), authCtrl.TokenFcm)
	}

	userCtrl := user.UserController(db)
	user := main.Group("user")
	{
		user.GET("/my-profile", middleware.AuthDashboard(db), userCtrl.GetMyProfile)
		user.POST("", middleware.AuthDashboard(db), userCtrl.CreateUser)
		user.GET("", middleware.AuthDashboard(db), userCtrl.GetListUser)
		user.GET("/detail/:id", middleware.AuthDashboard(db), userCtrl.GetDetailUser)
		user.PUT(":id", middleware.AuthDashboard(db), userCtrl.UpdateUser)
		user.DELETE(":id", middleware.AuthDashboard(db), userCtrl.DeleteUser)
		user.POST("/resend-email-verification", middleware.AuthDashboard(db), userCtrl.ResendEmailVerification)
	}

	// Frontend calls these under /v1/pm (matching the Java backend's
	// @RequestMapping("/api/v1/pm") convention), so mount under /v1 here too.
	// No auth middleware: this module runs against a local/standalone Go
	// instance for now and intentionally doesn't require its own login.
	pmCtrl := pm.PmController(db)
	v1 := main.Group("v1")
	pmGroup := v1.Group("pm")
	{
		pmGroup.GET("/dashboard", pmCtrl.Dashboard)
		pmGroup.GET("/task-statuses", pmCtrl.TaskStatuses)
		pmGroup.GET("/projects", pmCtrl.Projects)
		pmGroup.GET("/kanban", pmCtrl.Kanban)
		pmGroup.GET("/clients", pmCtrl.Clients)
		pmGroup.GET("/members", pmCtrl.Members)
		pmGroup.GET("/tasks", pmCtrl.Tasks)
		pmGroup.POST("/tasks", pmCtrl.CreateTask)
		pmGroup.PUT("/tasks/:id", pmCtrl.UpdateTask)
		pmGroup.PATCH("/tasks/:id/status", pmCtrl.MoveTaskStatus)
		pmGroup.PATCH("/tasks/:id/move", pmCtrl.MoveTaskByKey)
		pmGroup.DELETE("/tasks/:id", pmCtrl.DeleteTask)
		pmGroup.GET("/tasks/:id/activity", pmCtrl.TaskActivity)
		pmGroup.GET("/tickets", pmCtrl.Tickets)
		pmGroup.POST("/tickets", pmCtrl.CreateTicket)
		pmGroup.GET("/tickets/:ticketId/comments", pmCtrl.TicketComments)
		pmGroup.POST("/tickets/:ticketId/comments", pmCtrl.AddTicketComment)
		pmGroup.GET("/ticket-templates", pmCtrl.TicketTemplates)
		pmGroup.GET("/activity-logs", pmCtrl.ActivityLogs)
		pmGroup.GET("/crm-projects", pmCtrl.CrmProjects)
		pmGroup.GET("/crm-projects/:id", pmCtrl.CrmProjectDetail)
		pmGroup.PUT("/crm-projects/:id/overview", pmCtrl.UpdateCrmProjectOverview)
		pmGroup.GET("/crm-projects/:id/tasks", pmCtrl.CrmProjectTasks)
		pmGroup.POST("/crm-projects/:id/tasks", pmCtrl.CreateCrmProjectTask)
		pmGroup.GET("/crm-projects/:id/members", pmCtrl.CrmProjectMembers)
		pmGroup.POST("/crm-projects/:id/members", pmCtrl.AddCrmProjectMember)
		pmGroup.PUT("/crm-projects/:id/members/:memberId", pmCtrl.UpdateCrmProjectMember)
		pmGroup.DELETE("/crm-projects/:id/members/:memberId", pmCtrl.DeleteCrmProjectMember)
		pmGroup.GET("/gantt/members", pmCtrl.GanttMembers)
		pmGroup.GET("/dashboard/summary", pmCtrl.DashboardSummary)
		pmGroup.GET("/dashboard/team-workload", pmCtrl.TeamWorkloadByPeriod)
		pmGroup.POST("/tasks/:id/clock-in", pmCtrl.ClockInTask)
		pmGroup.POST("/tasks/:id/clock-out", pmCtrl.ClockOutTask)
		pmGroup.GET("/tasks/:id/time-logs", pmCtrl.TaskTimeLogs)
		pmGroup.GET("/time-logs/active", pmCtrl.ActiveTimeLog)
		pmGroup.POST("/tasks/:id/time-logs/manual", pmCtrl.CreateManualTimeLog)
		pmGroup.PATCH("/time-logs/:id", pmCtrl.UpdateTimeLog)
		pmGroup.DELETE("/time-logs/:id", pmCtrl.DeleteTimeLog)
		pmGroup.GET("/timesheets", pmCtrl.Timesheets)
		pmGroup.POST("/timesheets", pmCtrl.CreateTimesheet)
		pmGroup.PATCH("/timesheets/:id/status", pmCtrl.UpdateTimesheetStatus)
		pmGroup.DELETE("/timesheets/:id", pmCtrl.DeleteTimesheet)
	}
}
