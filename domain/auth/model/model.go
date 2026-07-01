package model

import (
	"time"

	"github.com/dgrijalva/jwt-go"
)

type Jwt struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	RoleID   string `json:"role_id"`
	*jwt.StandardClaims
}

type User struct {
	ID        string
	Username  string
	RoleID    string
	Role      Role
	Password  string
	Token     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Role struct {
	ID   string
	Name string
}

func (Role) TableName() string {
	return "mroles"
}

func (User) TableName() string {
	return "musers"
}
