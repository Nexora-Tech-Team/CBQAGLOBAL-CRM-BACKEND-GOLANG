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

	pmCtrl := pm.PmController(db)
	pmGroup := main.Group("pm", middleware.AuthDashboard(db))
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
		pmGroup.GET("/tickets", pmCtrl.Tickets)
		pmGroup.POST("/tickets", pmCtrl.CreateTicket)
		pmGroup.GET("/tickets/:ticketId/comments", pmCtrl.TicketComments)
		pmGroup.POST("/tickets/:ticketId/comments", pmCtrl.AddTicketComment)
		pmGroup.GET("/ticket-templates", pmCtrl.TicketTemplates)
		pmGroup.GET("/activity-logs", pmCtrl.ActivityLogs)
	}
}
