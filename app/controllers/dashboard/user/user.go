package user

import (
	"erp-cbqa-global/domain/dashboard/user/model"
	"erp-cbqa-global/lib/constant"
	"erp-cbqa-global/lib/form"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"erp-cbqa-global/domain/dashboard/user"
	"erp-cbqa-global/domain/dashboard/user/repository"
	authLib "erp-cbqa-global/lib/auth"
	"erp-cbqa-global/lib/response"
)

type userController struct {
	UserService user.UserServiceInterface
}

func UserController(db *gorm.DB) *userController {
	return &userController{
		UserService: user.UserService(repository.UserRepository(db)),
	}
}

func (uc *userController) GetMyProfile(ctx *gin.Context) {
	authUser, err := authLib.GetAuthUserCtx(ctx)
	if err != nil {
		response.Error(ctx, http.StatusUnauthorized, err.Error())
		return
	}

	data, errStatus, err := uc.UserService.GetMyProfile(authUser.ID)
	if err != nil {
		response.Error(ctx, errStatus, err.Error())
		return
	}
	response.Json(ctx, http.StatusOK, data)
}

func (uc *userController) CreateUser(ctx *gin.Context) {
	var reqBody model.RequestCreateUser
	if err := ctx.ShouldBindJSON(&reqBody); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	authUser, err := authLib.GetAuthUserCtx(ctx)
	if err != nil {
		response.Error(ctx, http.StatusUnauthorized, err.Error())
		return
	}

	check, errorCheck := uc.UserService.ValidateEmail(reqBody.Email)

	if check >= 1 {
		log.Println(errorCheck)
		response.Error(ctx, http.StatusConflict, constant.EmailConflict)
		return
	}

	resUser, resCode, err := uc.UserService.CreateUser(&reqBody, authUser)
	if err != nil {
		response.Error(ctx, resCode, err.Error())
		return
	}

	response.Json(ctx, http.StatusOK, resUser)
}

func (uc *userController) GetListUser(ctx *gin.Context) {
	page, _ := strconv.Atoi(form.SQLInjectorNumber(ctx.DefaultQuery("page", "1")))
	limit, _ := strconv.Atoi(form.SQLInjectorNumber(ctx.DefaultQuery("limit", "10")))

	search := form.SQLInjector(ctx.DefaultQuery(`search`, ``))
	status := form.SQLInjectorSingleNumber(ctx.DefaultQuery(`status`, ``))
	roleId := form.SQLInjector(ctx.DefaultQuery(`role_id`, ``))
	sortBy := ctx.DefaultQuery(`sort_by`, `created_at`)
	sort := ctx.DefaultQuery(`sort`, `desc`)
	isAll := ctx.Query("is_all")
	isAllBool, _ := strconv.ParseBool(isAll)
	isShowDelete := ctx.Query("is_show_delete")
	isShowDeleteBool, _ := strconv.ParseBool(isShowDelete)

	if isAllBool {
		data, errStatus, err := uc.UserService.GetUserNoLimit(search, status, roleId, sortBy, sort, isShowDeleteBool, limit)
		if err != nil {
			response.Error(ctx, errStatus, err.Error())
			return
		}
		response.Json(ctx, http.StatusOK, data)
	} else {
		if limit == 0 {
			limit = 10
		}
		offset := (page - 1) * limit
		data, totalRow, errStatus, err := uc.UserService.GetUsers(search, status, roleId, sortBy, sort, offset, limit, isShowDeleteBool)
		if err != nil {
			response.Error(ctx, errStatus, err.Error())
			return
		}
		response.JsonPagination(ctx, http.StatusOK, data, page, limit, totalRow)
	}
}

func (uc *userController) GetDetailUser(context *gin.Context) {
	id := form.SQLInjector(context.Param(`id`))
	data, errStatus, err := uc.UserService.GetDetailUser(id)
	if err != nil {
		response.Error(context, errStatus, err.Error())
		return
	}
	response.Json(context, http.StatusOK, data)
}

func (uc *userController) UpdateUser(ctx *gin.Context) {
	id := ctx.Param(`id`)
	var req model.RequestUpdateUser
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	authUser, err := authLib.GetAuthUserCtx(ctx)
	if err != nil {
		response.Error(ctx, http.StatusUnauthorized, err.Error())
		return
	}
	data, resCode, err := uc.UserService.GetDetailUser(id)
	if err != nil {
		response.Error(ctx, resCode, err.Error())
		return
	}
	if err = uc.validateStatus(data.IsActive, int(req.IsActive)); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	data, resCode, err = uc.UserService.UpdateUser(id, &req, authUser)
	if err != nil {
		response.Error(ctx, resCode, err.Error())
		return
	}
	response.Json(ctx, http.StatusOK, data)
}

func (uc *userController) DeleteUser(context *gin.Context) {
	id := context.Param(`id`)
	authUser, err := authLib.GetAuthUserCtx(context)
	if err != nil {
		response.Error(context, http.StatusUnauthorized, err.Error())
		return
	}
	resCode, err := uc.UserService.DeleteUser(id, authUser)
	if err != nil {
		response.Error(context, resCode, err.Error())
		return
	}
	response.Json(context, http.StatusOK, nil)
}

func (uc *userController) ResendEmailVerification(ctx *gin.Context) {
	var req model.RequestResendEmail
	if err := ctx.Bind(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	resCode, err := uc.UserService.ResendEmailVerification(req.Email)
	if err != nil {
		response.Error(ctx, resCode, err.Error())
		return
	}
	response.Json(ctx, http.StatusOK, nil)
}

func (uc *userController) validateStatus(statusNow, statusChange int) error {
	if statusNow != statusChange && statusNow == 2 {
		return errors.New("user account is unverified")
	} else if statusNow != statusChange && statusChange == 2 {
		return errors.New("user account is verified")
	}
	return nil
}
