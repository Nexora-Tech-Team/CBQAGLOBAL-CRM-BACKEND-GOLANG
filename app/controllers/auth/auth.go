package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"erp-cbqa-global/domain/auth"
	"erp-cbqa-global/domain/auth/model"
	"erp-cbqa-global/domain/auth/repository"
	"erp-cbqa-global/lib/response"
)

type authController struct {
	AuthService auth.AuthServiceInterface
}

func AuthController(db *gorm.DB) *authController {
	return &authController{
		AuthService: auth.AuthService(repository.AuthRepository(db)),
	}
}

func (ac *authController) Login(context *gin.Context) {
	var reqBody model.ReqBody
	if err := context.ShouldBindJSON(&reqBody); err != nil {
		response.Error(context, http.StatusBadRequest, err.Error())
		return
	}
	resBody, errStatus, err := ac.AuthService.Login(reqBody)
	if err != nil {
		response.Error(context, errStatus, err.Error())
		return
	}
	response.Json(context, http.StatusOK, resBody)
}

func (ac *authController) Logout(context *gin.Context) {
	user, err := ac.AuthService.CheckAuth(context.Request.Header.Get("Authorization"))
	if err != nil {
		response.Error(context, http.StatusUnauthorized, err.Error())
		return
	}
	if errStatus, err := ac.AuthService.Logout(*user); err != nil {
		response.Error(context, errStatus, err.Error())
		return
	}
	response.Json(context, http.StatusOK, nil)
}
