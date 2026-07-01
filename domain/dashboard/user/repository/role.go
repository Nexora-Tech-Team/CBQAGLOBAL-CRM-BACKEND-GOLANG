package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"erp-cbqa-global/domain/dashboard/user/model"
)

type RoleRepositoryInterface interface {
	GetRoleNoLimit(search, status, sortBy, sort string, isShowDelete bool, limit int) ([]model.Role, error)          // read all
	GetRoles(search, status, sortBy, sort string, offset, limit int, isShowDelete bool) ([]model.Role, int64, error) // read pagination
	GetRole(offset, limit int) ([]model.Role, int64, error)
	DetailRole(id string) (model.ResponseRole, error)
	QuerySelectUser(selectParams []string, conditions interface{}) (model.User, error)
	DeleteByID(id string) error
	GetLatestRoleCode() (model.Role, error)
	CreateUserLogHistory(req model.UserLogHistory) error
	GetMenu(offset, limit int) ([]model.Menu, int64, error)
	GetMenuNoLimit() ([]model.Menu, error)
	QuerySelectRole(selectParams []string, conditions interface{}) (model.Role, error)
	GetMenuById(id string) ([]model.Menu, int64, error)
	UpdateRole(id string, role *model.Role) error
}

type roleRepository struct {
	DB *gorm.DB
}

func RoleRepository(db *gorm.DB) RoleRepositoryInterface {
	return &roleRepository{
		DB: db,
	}
}

func (r *roleRepository) GetRole(offset, limit int) ([]model.Role, int64, error) {
	var role []model.Role
	var totalRow int64

	if err := r.DB.Table("roles").Count(&totalRow).Offset(offset).Limit(limit).Scan(&role).Error; err != nil {
		return role, totalRow, errors.New("failed to get role : " + err.Error())
	}

	return role, totalRow, nil
}

// GET Role tanpa paginatino
func (r *roleRepository) GetRoleNoLimit(search, status, sortBy, sort string, isShowDelete bool, limit int) ([]model.Role, error) {
	var role []model.Role
	whereCondition := `roles.deleted_at IS NULL`
	if isShowDelete {
		whereCondition = `(roles.deleted_at IS NULL OR roles.deleted_at IS NOT NULL)`
	}
	if search != `` {
		whereCondition += fmt.Sprintf(` AND (roles.name ILIKE '%v')`, `%`+search+`%`)
	}
	if status != `` {
		whereCondition += fmt.Sprintf(` AND roles.status = %v`, status)
	}
	if err := r.DB.
		Table("roles").
		Where(whereCondition).
		Order(sortBy + ` ` + sort).
		Limit(limit).
		Find(&role).Error; err != nil {
		return role, err
	}
	return role, nil
}

// GET Role dengan pagination
func (r *roleRepository) GetRoles(search, status, sortBy, sort string, offset, limit int, isShowDelete bool) ([]model.Role, int64, error) {
	var role []model.Role
	var totalRow int64
	whereCondition := `roles.deleted_at IS NULL`
	if isShowDelete {
		whereCondition = `(roles.deleted_at IS NULL OR roles.deleted_at IS NOT NULL)`
	}
	if search != `` {
		whereCondition += fmt.Sprintf(` AND (roles.name ILIKE '%v')`, `%`+search+`%`)
	}
	if status != `` {
		whereCondition += fmt.Sprintf(` AND roles.status = %v`, status)
	}
	if err := r.DB.Table("roles").
		Where(whereCondition).
		Count(&totalRow).Error; err != nil {
		return role, totalRow, err
	}

	if err := r.DB.
		Table("roles").
		Select("roles.*").
		Where(whereCondition).
		Offset(offset).
		Limit(limit).
		Order(sortBy + ` ` + sort).
		Find(&role).Error; err != nil {
		return role, totalRow, err
	}

	return role, totalRow, nil
}

// GET Role by id
func (r *roleRepository) DetailRole(id string) (model.ResponseRole, error) {
	var responseRole model.ResponseRole
	var role model.Role
	if err := r.DB.Where("id = ?", id).First(&role).Error; err != nil {
		return responseRole, err
	}
	responseRole = model.ResponseRole{
		ID:          role.ID,
		Name:        role.Name,
		Code:        role.Code,
		IsActive:    role.IsActive,
		Description: role.Description,
	}
	return responseRole, nil
}

// GET User
func (r *roleRepository) QuerySelectUser(selectParams []string, conditions interface{}) (model.User, error) {
	var user model.User
	return user, r.DB.Select(selectParams).Take(&user, conditions).Error
}

// DELETE Role dan update status role
func (r *roleRepository) DeleteByID(id string) error {
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Table("roles").
			Where("id = ?", id).
			Update("status", 0).Error
		if err != nil {
			tx.Rollback()
			return err
		}
		err = tx.Table("roles").Delete(&model.Role{ID: id}).Error
		if err != nil {
			tx.Rollback()
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// GET Role terakhir untuk generate Role Code
func (r *roleRepository) GetLatestRoleCode() (model.Role, error) {
	var role model.Role
	if err := r.DB.Table("roles").
		Select(`role_code, created_at`).
		Order(`created_at desc`).
		Limit(1).
		Find(&role).Error; err != nil {
		return role, err
	}
	return role, nil
}

func (r *roleRepository) CreateUserLogHistory(req model.UserLogHistory) error {
	if err := r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("user_log_histories").Create(&req).Error; err != nil {
			tx.Rollback()
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// GetRoleManagementMenu is repository to get list role management menu
func (r *roleRepository) GetMenu(offset, limit int) (menu []model.Menu, totalRow int64, err error) {
	if err := r.DB.Table("menus").Order("priority asc").Count(&totalRow).Offset(offset).Limit(limit).Scan(&menu).Error; err != nil {
		return menu, totalRow, errors.New("failed to get menu list : " + err.Error())
	}

	return menu, totalRow, nil
}

// GetRoleManagementMenuNoLimit is repository to get all list role management menu
func (r *roleRepository) GetMenuNoLimit() (menu []model.Menu, err error) {
	if err := r.DB.Table("menus").Order("priority asc").Scan(&menu).Error; err != nil {
		return menu, errors.New("failed to get all menu list : " + err.Error())
	}

	return menu, nil
}

// QuerySelectRole is repository to get role
func (r *roleRepository) QuerySelectRole(selectParams []string, conditions interface{}) (model.Role, error) {
	var role model.Role
	return role, r.DB.Select(selectParams).Take(&role, conditions).Error
}

// UPDATE Role
func (r *roleRepository) UpdateRole(id string, role *model.Role) error {
	return r.DB.Table("roles").Where("id = ?", id).Updates(role).Error
}

func (r *roleRepository) GetMenuById(id string) ([]model.Menu, int64, error) {
	var menu []model.Menu
	var totalRow int64

	if err := r.DB.
		Where("id = ?", id).Find(&menu).Count(&totalRow).Error; err != nil {
		return menu, totalRow, errors.New("failed to get menu count : " + err.Error())
	}

	return menu, totalRow, nil

}
