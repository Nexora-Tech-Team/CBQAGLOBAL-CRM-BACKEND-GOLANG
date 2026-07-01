package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"erp-cbqa-global/domain/auth"
	"erp-cbqa-global/domain/auth/repository"
	authdashboard "erp-cbqa-global/domain/dashboard/auth"
	repodashboard "erp-cbqa-global/domain/dashboard/auth/repository"
	authLib "erp-cbqa-global/lib/auth"
	"erp-cbqa-global/lib/constant"
	"erp-cbqa-global/lib/response"
)

func Auth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authService := auth.AuthService(repository.AuthRepository(db))
		user, err := authService.CheckAuth(c.Request.Header.Get("Authorization"))
		if err != nil {
			response.Error(c, http.StatusUnauthorized, err.Error())
			c.Abort()
			return
		}
		if user.Role.Name != "User" {
			response.Error(c, http.StatusUnauthorized, constant.NotAuthorize)
			c.Abort()
			return
		}

		userStr, err := json.Marshal(&authLib.AuthData{
			ID:        user.ID,
			Username:  user.Username,
			RoleID:    user.RoleID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		})
		if err != nil {
			response.Error(c, http.StatusUnauthorized, err.Error())
			c.Abort()
			return
		}
		c.Set("auth", string(userStr))
	}
}

func AuthDashboard(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authService := authdashboard.AuthService(repodashboard.AuthRepository(db))
		user, err := authService.CheckAuth(c.Request.Header.Get("Authorization"))
		if err != nil {
			response.Error(c, http.StatusUnauthorized, err.Error())
			c.Abort()
			return
		}
		userStr, err := json.Marshal(&authLib.AuthData{
			ID:       user.ID.String(),
			Username: user.Username,
			RoleID:   user.RoleID.String(),
		})
		if err != nil {
			response.Error(c, http.StatusForbidden, err.Error())
			c.Abort()
			return
		}
		c.Set("auth", string(userStr))
	}
}
