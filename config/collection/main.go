package collection

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"erp-cbqa-global/app/controllers/auth"
	"erp-cbqa-global/config/middleware"
)

func MainRouter(db *gorm.DB, main *gin.RouterGroup) {
	authCtrl := auth.AuthController(db)
	auth := main.Group("auth")
	{
		auth.POST("/login", authCtrl.Login)
		auth.POST("/logout", middleware.Auth(db), authCtrl.Logout)
	}
}
