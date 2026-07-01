package repository

import (
	"erp-cbqa-global/domain/auth/model"

	"gorm.io/gorm"
)

type AuthRepositoryInterface interface {
	FirstByUsername(username string) (model.User, error)
	UpdateToken(model.User, string) error
	FirstByID(string) (model.User, error)
	DeleteToken(model.User) error
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
	if err := r.DB.Where("username = ?", username).First(&user).Error; err != nil {
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
	if err := r.DB.Preload("Role").First(&user, id).Error; err != nil {
		return user, err
	}
	return user, nil
}

func (r *authRepository) DeleteToken(user model.User) error {
	user.Token = ""
	return r.DB.Save(&user).Error
}
