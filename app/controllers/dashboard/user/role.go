package user

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"erp-cbqa-global/domain/dashboard/user"
	"erp-cbqa-global/domain/dashboard/user/model"
	"erp-cbqa-global/domain/dashboard/user/repository"
	authLib "erp-cbqa-global/lib/auth"
	"erp-cbqa-global/lib/form"
	"erp-cbqa-global/lib/response"
)

type roleController struct {
	RoleService user.RoleServiceInterface
}

func RoleController(db *gorm.DB) *roleController {
	return &roleController{
		RoleService: user.RoleService(repository.RoleRepository(db), db),
	}
}

func (rc *roleController) IndexRole(context *gin.Context) {
	page, _ := strconv.Atoi(form.SQLInjectorNumber(context.DefaultQuery("page", "1")))
	limit, _ := strconv.Atoi(form.SQLInjectorNumber(context.DefaultQuery("limit", "10")))
	search := form.SQLInjector(context.DefaultQuery(`search`, ``))
	status := form.SQLInjectorSingleNumber(context.DefaultQuery(`status`, ``))
	sortBy := context.DefaultQuery(`sort_by`, `created_at`)
	sort := context.DefaultQuery(`sort`, `desc`)
	isAll := context.Query("is_all")
	isAllBool, _ := strconv.ParseBool(isAll)
	isShowDelete := context.Query("is_show_delete")
	isShowDeleteBool, _ := strconv.ParseBool(isShowDelete)
	if isAllBool {
		data, errStatus, err := rc.RoleService.GetRoleNoLimit(search, status, sortBy, sort, isShowDeleteBool, limit)
		if err != nil {
			response.Error(context, errStatus, err.Error())
			return
		}
		response.Json(context, http.StatusOK, data)
	} else {
		if limit == 0 {
			limit = 10
		}
		offset := (page - 1) * limit
		data, totalRow, errStatus, err := rc.RoleService.GetRoles(search, status, sortBy, sort, offset, limit, isShowDeleteBool)
		if err != nil {
			response.Error(context, errStatus, err.Error())
			return
		}
		response.JsonPagination(context, http.StatusOK, data, page, limit, totalRow)
	}
}

func (rc *roleController) GetMyRole(ctx *gin.Context) {
	authUser, err := authLib.GetAuthUserCtx(ctx)
	if err != nil {
		response.Error(ctx, http.StatusUnauthorized, err.Error())
		return
	}
	data, errStatus, err := rc.RoleService.GetMyRole(authUser.RoleID)
	if err != nil {
		response.Error(ctx, errStatus, err.Error())
		return
	}
	response.Json(ctx, http.StatusOK, data)
}

func (rc *roleController) GetRoleById(ctx *gin.Context) {
	id := ctx.Param(`id`)
	data, errStatus, err := rc.RoleService.GetMyRole(id)
	if err != nil {
		response.Error(ctx, errStatus, err.Error())
		return
	}
	response.Json(ctx, http.StatusOK, data)
}

func (rc *roleController) Create(ctx *gin.Context) {
	var req model.RequestCreate
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	authUser, err := authLib.GetAuthUserCtx(ctx)
	if err != nil {
		response.Error(ctx, http.StatusUnauthorized, err.Error())
		return
	}
	resp, errStatus, err := rc.RoleService.Create(&req, authUser)
	if err != nil {
		response.Error(ctx, errStatus, err.Error())
		return
	}
	response.Json(ctx, http.StatusOK, resp)
}

func (rc *roleController) UpdateRole(ctx *gin.Context) {
	id := ctx.Param(`id`)
	var req model.RequestUpdateRole
	if err := ctx.Bind(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	authUser, err := authLib.GetAuthUserCtx(ctx)
	if err != nil {
		response.Error(ctx, http.StatusUnauthorized, err.Error())
		return
	}
	resp, errStatus, err := rc.RoleService.UpdateRole(id, &req, authUser)
	if err != nil {
		response.Error(ctx, errStatus, err.Error())
		return
	}
	response.Json(ctx, http.StatusOK, resp)
}

func (rc *roleController) DeleteRole(context *gin.Context) {
	id := context.Param(`id`)
	authUser, err := authLib.GetAuthUserCtx(context)
	if err != nil {
		response.Error(context, http.StatusUnauthorized, err.Error())
		return
	}
	errStatus, err := rc.RoleService.DeleteRole(id, authUser)
	if err != nil {
		response.Error(context, errStatus, err.Error())
		return
	}
	response.Json(context, http.StatusOK, nil)
}

func (rc *roleController) GetRoleMenu(ctx *gin.Context) {
	isAllBool, _ := strconv.ParseBool(ctx.Query("is_all"))
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if isAllBool {
		data, resCode, err := rc.RoleService.GetMenuNoLimit()
		if err != nil {
			response.Error(ctx, resCode, err.Error())
			return
		}
		response.Json(ctx, resCode, data)
	} else {
		offset := (page - 1) * limit
		data, totalRow, resCode, err := rc.RoleService.GetMenu(offset, limit)
		if err != nil {
			response.Error(ctx, resCode, err.Error())
			return
		}
		response.JsonPagination(ctx, resCode, data, page, limit, totalRow)
	}
}
