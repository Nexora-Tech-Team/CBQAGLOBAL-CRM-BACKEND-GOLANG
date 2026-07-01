package model

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"

	"erp-cbqa-global/lib/base"
)

type Jwt struct {
	*AuthUser
	*jwt.StandardClaims
}

type User struct {
	ID                  uuid.UUID
	Code                string
	Description         string
	Username            string
	FirstName           string
	LastName            string
	Email               string
	Password            string
	EmpID               uuid.UUID
	RoleID              uuid.UUID
	Role                Role
	ProfileImage        string
	SignatureImage      string
	CompanyID           uuid.UUID
	FullName            string
	Token               string
	Otp                 *string
	ForgotPasswordToken *string
	ResetPasswordToken  *string
	ResetPasswordOtp    *string
	TokenFcm            *string
	base.BaseModelMaster
}

type Role struct {
	ID          uuid.UUID
	Code        string
	Name        string
	Description string
	base.BaseModelMaster
}

type AuthUser struct {
	ID       uuid.UUID
	Username string
	RoleID   uuid.UUID
}

func (User) TableName() string {
	return "musers"
}

func (Role) TableName() string {
	return "mroles"
}
