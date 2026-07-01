package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"erp-cbqa-global/domain/dashboard/auth"
	"erp-cbqa-global/domain/dashboard/auth/model"
	"erp-cbqa-global/domain/dashboard/auth/repository"
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

func (ac *authController) ActivateAccount(ctx *gin.Context) {
	var reqBody model.ActivateUserReqBody
	if err := ctx.ShouldBindJSON(&reqBody); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	resCode, err := ac.AuthService.ActivateUser(&reqBody)
	if err != nil {
		response.Error(ctx, resCode, err.Error())
		return
	}
	response.Json(ctx, resCode, nil)
}

func (ac *authController) ForgotPassword(ctx *gin.Context) {
	var reqBody model.AuthForgotPasswordReqBody
	if err := ctx.ShouldBindJSON(&reqBody); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	resBody, resCode, err := ac.AuthService.ForgotPassword(&reqBody)
	if err != nil {
		response.Error(ctx, resCode, err.Error())
		return
	}
	response.Json(ctx, resCode, resBody)
}

func (ac *authController) ResetPassword(ctx *gin.Context) {
	var reqBody model.UserResetPasswordReqBody
	if err := ctx.ShouldBindJSON(&reqBody); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	statusCode, err := ac.AuthService.ResetPassword(&reqBody)
	if err != nil {
		response.Error(ctx, statusCode, err.Error())
		return
	}
	response.Json(ctx, statusCode, nil)
}

func (ac *authController) TokenFcm(context *gin.Context) {
	var reqBody model.ReqTokenFcm
	if err := context.ShouldBindJSON(&reqBody); err != nil {
		response.Error(context, http.StatusBadRequest, err.Error())
		return
	}
	user, err := ac.AuthService.CheckAuth(context.Request.Header.Get("Authorization"))
	if err != nil {
		response.Error(context, http.StatusUnauthorized, err.Error())
		return
	}
	_, errStatus, err := ac.AuthService.TokenFcm(user.ID.String(), reqBody)
	if err != nil {
		response.Error(context, errStatus, err.Error())
		return
	}
	response.Json(context, http.StatusOK, nil)
}
