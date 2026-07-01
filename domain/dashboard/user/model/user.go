package model

import (
	"time"

	"github.com/dgrijalva/jwt-go"

	"erp-cbqa-global/lib/base"

	"gorm.io/datatypes"
)

type (
	RequestCreateUser struct {
		Username       string  `json:"username"`
		Password       string  `json:"password"`
		EmpId          *string `json:"emp_id"`
		RoleId         string  `json:"role_id"`
		ProfileImage   string  `json:"profile_image"`
		SignatureImage string  `json:"signature_image"`
		IsActive       int     `json:"is_active"`
		CompanyId      string  `json:"company_id"`
		FullName       string  `json:"full_name"`
		Email          string  `json:"email"`
		Phone          string  `json:"phone"`
		Description    string  `json:"description"`
	}

	CreateUser struct {
		Id             string  `json:"id" gorm:"unique;default:gen_random_uuid()"`
		Username       string  `json:"username"`
		Password       string  `json:"password"`
		EmpId          *string `json:"emp_id"`
		RoleId         string  `json:"role_id"`
		ProfileImage   string  `json:"profile_image"`
		SignatureImage string  `json:"signature_image"`
		IsActive       int     `json:"is_active"`
		CompanyId      string  `json:"company_id"`
		FullName       string  `json:"full_name"`
		Email          string  `json:"email"`
		Phone          string  `json:"phone"`
		Description    string  `json:"description"`
		CreatedBy      string  `json:"created_by"`
		UpdatedBy      string  `json:"updated_by"`
	}

	UserActivationClaims struct {
		Id  string
		Otp string
		*jwt.StandardClaims
	}

	User struct {
		Id             string  `json:"id" gorm:"unique;default:gen_random_uuid()"`
		Code           string  `json:"code"`
		Username       string  `json:"username"`
		Password       string  `json:"password"`
		EmpId          *string `json:"emp_id"`
		RoleId         string  `json:"role_id"`
		Role           Role    `json:"role" gorm:"foreignKey:RoleId;references:Id"`
		ProfileImage   string  `json:"profile_image"`
		SignatureImage string  `json:"signature_image"`
		IsActive       int     `json:"is_active"`
		CompanyId      string  `json:"company_id"`
		FullName       string  `json:"full_name"`
		FirstName      string  `json:"first_name"`
		LastName       string  `json:"last_name"`
		Token          string  `json:"token"`
		Email          string  `json:"email"`
		Phone          string  `json:"phone"`
		Description    string  `json:"description"`
		base.BaseModelMaster
	}

	UserLogHistory struct {
		Id        string `json:"id" gorm:"unique;default:gen_random_uuid()"`
		Action    string
		CreatedAt time.Time
		Model     string
		NewObject datatypes.JSON
		ObjectId  string
		OldObject datatypes.JSON
		UserId    string
	}

	ResponseUser struct {
		Id             string  `json:"id" gorm:"unique;default:gen_random_uuid()"`
		Code           string  `json:"code"`
		Username       string  `json:"username"`
		EmpId          *string `json:"emp_id"`
		RoleId         string  `json:"role_id"`
		Role           Role    `json:"role" gorm:"foreignKey:RoleId;references:Id"`
		ProfileImage   string  `json:"profile_image"`
		SignatureImage string  `json:"signature_image"`
		IsActive       int     `json:"is_active"`
		CompanyId      string  `json:"company_id"`
		FullName       string  `json:"full_name"`
		FirstName      string  `json:"first_name"`
		LastName       string  `json:"last_name"`
		Email          string  `json:"email"`
		Phone          string  `json:"phone"`
		Description    string  `json:"description"`
	}

	RequestUpdateUser struct {
		Username       string `json:"username"`
		Password       string `json:"password"`
		EmpId          string `json:"emp_id"`
		RoleId         string `json:"role_id"`
		ProfileImage   string `json:"profile_image"`
		SignatureImage string `json:"signature_image"`
		IsActive       int    `json:"is_active"`
		CompanyId      string `json:"company_id"`
		FullName       string `json:"full_name"`
		Email          string `json:"email"`
		Phone          string `json:"phone"`
		Description    string `json:"description"`
	}

	UpdateUser struct {
		Id             string `json:"id" gorm:"unique;default:gen_random_uuid()"`
		Code           string `json:"code"`
		Username       string `json:"username"`
		Password       string `json:"password"`
		EmpId          string `json:"emp_id"`
		RoleId         string `json:"role_id"`
		Role           Role   `json:"role" gorm:"foreignKey:RoleId;references:Id"`
		ProfileImage   string `json:"profile_image"`
		SignatureImage string `json:"signature_image"`
		IsActive       int    `json:"is_active"`
		CompanyId      string `json:"company_id"`
		FullName       string `json:"full_name"`
		FirstName      string `json:"first_name"`
		LastName       string `json:"last_name"`
		Token          string `json:"token"`
		Email          string `json:"email"`
		Phone          string `json:"phone"`
		Description    string `json:"description"`
		UpdatedBy      string `json:"updated_by"`
	}

	RequestResendEmail struct {
		Email string `json:"email"`
	}
)

func (User) TableName() string {
	return "musers"
}
