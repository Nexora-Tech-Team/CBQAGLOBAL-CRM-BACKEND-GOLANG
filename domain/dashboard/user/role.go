package user

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"erp-cbqa-global/domain/dashboard/user/model"
	"erp-cbqa-global/domain/dashboard/user/repository"
	"erp-cbqa-global/lib/base"
	"erp-cbqa-global/lib/constant"

	authLib "erp-cbqa-global/lib/auth"

	"github.com/google/uuid"

	"gorm.io/gorm"
)

type RoleServiceInterface interface {
	GetRoleNoLimit(search, status, sortBy, sort string, isShowDelete bool, limit int) ([]model.Role, int, error)
	GetRoles(search, status, sortBy, sort string, offset, limit int, isShowDelete bool) ([]model.Role, int64, int, error)
	GetRole(offset, limit int) ([]model.Role, int64, int, error)
	GetMyRole(userId string) (model.ResponseRole, int, error)
	GetRoleById(id string) (model.ResponseRole, int, error)
	Create(req *model.RequestCreate, user *authLib.AuthData) (*model.Role, int, error)
	UpdateRole(id string, req *model.RequestUpdateRole, user *authLib.AuthData) (*model.Role, int, error)
	DeleteRole(id string, user *authLib.AuthData) (int, error)
	GetMenu(offset, limit int) ([]model.Menu, int64, int, error)
	GetMenuNoLimit() ([]model.Menu, int, error)
}

type roleService struct {
	Repository repository.RoleRepositoryInterface
	DB         *gorm.DB
}

func RoleService(repository repository.RoleRepositoryInterface, db *gorm.DB) RoleServiceInterface {
	return &roleService{
		Repository: repository,
		DB:         db,
	}
}

func (s *roleService) GetRole(offset, limit int) ([]model.Role, int64, int, error) {
	var roles []model.Role
	roles, totalRow, err := s.Repository.GetRole(offset, limit)
	if err != nil {
		return roles, totalRow, http.StatusInternalServerError, err
	}
	return roles, totalRow, http.StatusOK, nil
}

func (s *roleService) GetMyRole(id string) (model.ResponseRole, int, error) {
	var role model.ResponseRole
	role, err := s.Repository.DetailRole(id)
	if err != nil {
		return role, http.StatusInternalServerError, err
	}
	return role, http.StatusOK, nil
}

func (s *roleService) GetRoleById(id string) (model.ResponseRole, int, error) {
	var role model.ResponseRole
	role, err := s.Repository.DetailRole(id)
	if err != nil {
		return role, http.StatusInternalServerError, err
	}
	return role, http.StatusOK, nil
}

func (s *roleService) GetRoleNoLimit(name, status, sortBy, sort string, isShowDelete bool, limit int) ([]model.Role, int, error) {
	var role []model.Role
	role, err := s.Repository.GetRoleNoLimit(name, status, sortBy, sort, isShowDelete, limit)
	if err != nil {
		return role, http.StatusInternalServerError, err
	}
	return role, http.StatusOK, nil
}

func (s *roleService) GetRoles(name, status, sortBy, sort string, offset, limit int, isShowDelete bool) ([]model.Role, int64, int, error) {
	var role []model.Role
	role, totalRow, err := s.Repository.GetRoles(name, status, sortBy, sort, offset, limit, isShowDelete)
	if err != nil {
		return role, totalRow, http.StatusInternalServerError, err
	}
	return role, totalRow, http.StatusOK, nil
}

func (s *roleService) Create(req *model.RequestCreate, user *authLib.AuthData) (*model.Role, int, error) {
	storeRoleModel := &model.Role{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		BaseModelMaster: base.BaseModelMaster{
			CreatedBy: user.ID,
			UpdatedBy: user.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	return storeRoleModel, http.StatusOK, nil
}

func (s *roleService) UpdateRole(id string, req *model.RequestUpdateRole, user *authLib.AuthData) (*model.Role, int, error) {
	role, err := s.Repository.QuerySelectRole([]string{"*"}, model.Role{ID: id})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusBadRequest, errors.New(constant.RecordNotFound)
		}
		return nil, http.StatusBadRequest, err
	}
	fmt.Println(role.Name)
	roleModel := &model.Role{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		BaseModelMaster: base.BaseModelMaster{
			UpdatedAt: time.Now(),
			UpdatedBy: user.ID,
		},
	}

	err = s.Repository.UpdateRole(id, roleModel)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return roleModel, http.StatusOK, nil
}

func (s *roleService) DeleteRole(id string, user *authLib.AuthData) (int, error) {
	data, err := s.Repository.DetailRole(id)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	_, err = s.Repository.QuerySelectUser([]string{"*"}, model.User{RoleId: id})
	if err == nil {
		return http.StatusBadRequest, errors.New(constant.RoleInUse)
	}
	if data.ID == "" {
		return http.StatusBadRequest, errors.New(constant.RecordNotFound)
	}

	err = s.Repository.DeleteByID(id)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func (s *roleService) GetMenu(offset, limit int) (menu []model.Menu, totalRow int64, resCode int, err error) {
	menu, totalRow, err = s.Repository.GetMenu(offset, limit)
	if err != nil {
		return menu, totalRow, http.StatusInternalServerError, err
	}
	return menu, totalRow, http.StatusOK, nil
}

func (s *roleService) GetMenuNoLimit() (menu []model.Menu, resCode int, err error) {
	menu, err = s.Repository.GetMenuNoLimit()
	if err != nil {
		return menu, http.StatusInternalServerError, err
	}
	return menu, http.StatusOK, nil
}
