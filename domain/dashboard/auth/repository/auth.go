package repository

import (
	"erp-cbqa-global/domain/dashboard/auth/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthRepositoryInterface interface {
	FirstByUsername(username string) (model.User, error)
	UpdateToken(model.User, string) error
	FirstByID(string) (model.User, error)
	DeleteToken(user model.User) error
	QuerySelectUser(selectParams []string, conditions interface{}) (model.User, error)
	UpdateEncryptedPasswordAndActivateUser(id, encryptedPassword *string) error
	UpdateForgotPasswordToken(id, token, otp *string) error
	UpdateEncryptedPasswordAndResetToken(id, encryptedPassword *string) error
	QuerySelectRole(selectParams []string, conditions interface{}) (model.Role, error)
	UpdateTokenFcm(id string, tokenFcm string) error
}

type authRepository struct {
	DB *gorm.DB
}

func AuthRepository(DB *gorm.DB) AuthRepositoryInterface {
	return &authRepository{
		DB: DB,
	}
}

func (r *authRepository) FirstByUsername(username string) (model.User, error) {
	var user model.User
	if err := r.DB.Preload("Role").Where("username = ?", username).First(&user).Error; err != nil {
		return user, err
	}
	return user, nil
}

func (r *authRepository) UpdateToken(user model.User, ss string) error {
	user.Token = ss
	return r.DB.Save(&user).Error
}

func (r *authRepository) FirstByID(id string) (model.User, error) {
	var user model.User
	if err := r.DB.Preload("Role").Where("id = ?", id).First(&user).Error; err != nil {
		return user, err
	}
	return user, nil
}

func (r *authRepository) DeleteToken(user model.User) error {
	return r.DB.Model(&user).Update("token", "").Error
}

func (r *authRepository) QuerySelectUser(selectParams []string, conditions interface{}) (model.User, error) {
	var user model.User
	return user, r.DB.Select(selectParams).Take(&user, conditions).Error
}

func (r *authRepository) UpdateEncryptedPasswordAndActivateUser(id, encryptedPassword *string) error {
	parsedID, _ := uuid.Parse(*id)
	user := model.User{ID: parsedID}
	if err := r.DB.Model(&user).Updates(map[string]any{
		"password":             *encryptedPassword,
		"is_active":            1,
		"reset_password_token": nil,
		"token":                "",
		"reset_password_otp":   nil,
	}).Error; err != nil {
		return err
	}
	return nil
}

func (r *authRepository) UpdateForgotPasswordToken(id, token, otp *string) error {
	parsedID, _ := uuid.Parse(*id)
	user := model.User{ID: parsedID}
	if err := r.DB.Model(&user).Updates(model.User{ForgotPasswordToken: token, Otp: otp}).Error; err != nil {
		return err
	}
	return nil
}

func (r *authRepository) UpdateEncryptedPasswordAndResetToken(id, encryptedPassword *string) error {
	parsedID, _ := uuid.Parse(*id)
	user := model.User{ID: parsedID}
	if err := r.DB.Model(&user).Updates(map[string]any{
		"password":              *encryptedPassword,
		"forgot_password_token": nil,
		"token":                 "",
		"otp":                   nil,
	}).Error; err != nil {
		return err
	}
	return nil
}

func (r *authRepository) QuerySelectRole(selectParams []string, conditions interface{}) (model.Role, error) {
	var role model.Role
	return role, r.DB.Select(selectParams).Take(&role, conditions).Error
}

func (r *authRepository) UpdateTokenFcm(id string, tokenFcm string) error {
	return r.DB.Model(&model.User{}).Where("id = ?", id).Update("token_fcm", tokenFcm).Error
}
