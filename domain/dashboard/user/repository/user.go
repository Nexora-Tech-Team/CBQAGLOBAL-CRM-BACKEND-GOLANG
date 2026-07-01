package repository

import (
	"erp-cbqa-global/domain/dashboard/user/model"
	"fmt"
	"net/http"
	"strings"

	"gorm.io/gorm"
)

type UserRepositoryInterface interface {
	GetMyProfile(userId string) (model.ResponseUser, error)
	CreateUser(reqBody model.CreateUser, firstName, lastName string) (*model.User, int, error)
	QueryUpdateValuesById(id *string, values interface{}) error
	QuerySelectUser(selectParams []string, conditions interface{}) (*model.User, error)
	GetUserNoLimit(search, status, roleId, sortBy, sort string, isShowDelete bool, limit int) ([]model.ResponseUser, error)
	GetUsers(search, status, roleId, sortBy, sort string, offset, limit int, isShowDelete bool) ([]model.ResponseUser, int64, error)
	CreateUserLogHistory(req model.UserLogHistory) error
	DeleteById(id string) error
	UpdateUser(req model.UpdateUser) error
	ValidateEmail(email string) (int, error)
}

type userRepository struct {
	DB *gorm.DB
}

func UserRepository(db *gorm.DB) UserRepositoryInterface {
	return &userRepository{
		DB: db,
	}
}

func (ur *userRepository) GetMyProfile(userId string) (model.ResponseUser, error) {
	var responseUser model.ResponseUser
	var user model.User

	if err := ur.DB.Table("musers").
		Select("musers.*").
		Preload("Role").
		Where("musers.id= ?", userId).
		Find(&user).Error; err != nil {
		return responseUser, err
	}

	firstName, lastName := splitName(user.FirstName, user.LastName, user.FullName)

	responseUser = model.ResponseUser{
		Id:             user.Id,
		Code:           user.Code,
		Username:       user.Username,
		EmpId:          user.EmpId,
		RoleId:         user.RoleId,
		Role:           user.Role,
		ProfileImage:   user.ProfileImage,
		SignatureImage: user.SignatureImage,
		IsActive:       user.IsActive,
		CompanyId:      user.CompanyId,
		FullName:       user.FullName,
		FirstName:      firstName,
		LastName:       lastName,
		Email:          user.Email,
		Phone:          user.Phone,
		Description:    user.Description,
	}

	return responseUser, nil
}

func splitName(firstName, lastName, fullName string) (string, string) {
	if firstName != "" || lastName != "" {
		return firstName, lastName
	}

	parts := strings.Fields(fullName)
	if len(parts) == 0 {
		return "", ""
	}

	if len(parts) == 1 {
		return parts[0], ""
	}

	return parts[0], strings.Join(parts[1:], " ")
}

func (ur *userRepository) CreateUser(reqBody model.CreateUser, firstName, lastName string) (*model.User, int, error) {
	if err := ur.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Table("musers").Create(&reqBody).Error
		if err != nil {
			tx.Rollback()
			return err
		}
		// Update firstName and lastName after insert
		if err := tx.Table("musers").Where("id = ?", reqBody.Id).Updates(map[string]interface{}{
			"first_name": firstName,
			"last_name":  lastName,
		}).Error; err != nil {
			tx.Rollback()
			return err
		}
		return nil
	}); err != nil {
		return nil, http.StatusBadRequest, err
	}

	resUser, err := ur.QuerySelectUser([]string{"musers.*"}, &model.User{Id: reqBody.Id})
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return resUser, http.StatusOK, nil
}

func (ur *userRepository) QuerySelectUser(selectParams []string, conditions interface{}) (*model.User, error) {
	var user model.User
	if err := ur.DB.Table("musers").
		Select(selectParams).
		Preload("Role").
		Take(&user, conditions).Error; err != nil {
		return &user, err
	}

	return &user, nil
}

func (ur *userRepository) QueryUpdateValuesById(id *string, values interface{}) error {
	return ur.DB.Model(model.User{}).Where("id", id).Updates(values).Error
}

func (ur *userRepository) GetUserNoLimit(search, status, roleId, sortBy, sort string, isShowDelete bool, limit int) ([]model.ResponseUser, error) {
	var users []model.ResponseUser
	whereCondition := `musers.deleted_at IS NULL`
	if isShowDelete {
		whereCondition = `(musers.deleted_at IS NULL OR musers.deleted_at IS NOT NULL)`
	}
	if search != `` {
		whereCondition += fmt.Sprintf(` AND (musers.code ILIKE '%v' OR musers.full_name ILIKE '%v')`, `%`+search+`%`, `%`+search+`%`)
	}
	if status != `` {
		whereCondition += fmt.Sprintf(` AND musers.is_active = %v`, status)
	}
	if roleId != `` {
		whereCondition += fmt.Sprintf(` AND musers.role_id = '%v'`, roleId)
	}
	if err := ur.DB.Table("musers").
		Select("musers.*").
		Where(whereCondition).
		Order(sortBy + ` ` + sort).
		Limit(limit).
		Find(&users).Error; err != nil {
		return users, err
	}
	return users, nil
}

func (ur *userRepository) GetUsers(search, status, roleId, sortBy, sort string, offset, limit int, isShowDelete bool) ([]model.ResponseUser, int64, error) {
	var users []model.ResponseUser
	var totalRow int64
	whereCondition := `musers.deleted_at IS NULL`
	if isShowDelete {
		whereCondition = `(musers.deleted_at IS NULL OR musers.deleted_at IS NOT NULL)`
	}
	if search != `` {
		whereCondition += fmt.Sprintf(` AND (musers.full_name ILIKE '%v' OR musers.code ILIKE '%v')`, `%`+search+`%`, `%`+search+`%`)
	}
	if status != `` {
		whereCondition += fmt.Sprintf(` AND musers.is_active = %v`, status)
	}
	if roleId != `` {
		whereCondition += fmt.Sprintf(` AND musers.role_id = '%v'`, roleId)
	}

	if err := ur.DB.Table("musers").
		Select("musers.*").
		Joins("LEFT JOIN mroles ON mroles.id = musers.role_id").
		Where(whereCondition).
		Count(&totalRow).Error; err != nil {
		return users, totalRow, err
	}
	if err := ur.DB.Table("musers").
		Select("musers.*").
		Where(whereCondition).
		Offset(offset).
		Limit(limit).
		Order(sortBy + ` ` + sort).
		Find(&users).Error; err != nil {
		return users, totalRow, err
	}
	return users, totalRow, nil
}

func (ur *userRepository) CreateUserLogHistory(req model.UserLogHistory) error {
	if err := ur.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("muser_logs").Create(&req).Error; err != nil {
			tx.Rollback()
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (ur *userRepository) DeleteById(id string) error {
	err := ur.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Table("musers").Delete(&model.User{Id: id}).Error
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

func (ur *userRepository) UpdateUser(req model.UpdateUser) error {
	err := ur.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("musers").Select("id", "code", "username", "password", "emp_id", "role_id", "profile_image", "signature_image", "is_active", "company_id", "full_name", "first_name", "last_name", "email", "phone", "description", "updated_by").
			Updates(&req).Error; err != nil {
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

func (ur *userRepository) ValidateEmail(email string) (int, error) {
	var count int64
	err := ur.DB.Model(&model.User{}).Where("email = ? AND deleted_at IS NULL", email).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}
