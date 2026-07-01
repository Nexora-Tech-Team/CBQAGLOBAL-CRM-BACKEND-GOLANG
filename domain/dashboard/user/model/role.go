package model

import (
	"erp-cbqa-global/lib/base"
)

type (
	Role struct {
		ID          string `json:"id" gorm:"unique;default:gen_random_uuid()"`
		Code        string `json:"code"`
		Name        string `json:"name"`
		Description string `json:"description"`
		base.BaseModelMaster
	}

	RequestCreate struct {
		Name        string `json:"name"`
		IsActive    uint8  `json:"is_active"`
		Description string `json:"description"`
	}

	RequestUpdateRole struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		IsActive    uint8  `json:"is_active"`
		Description string `json:"description"`
	}

	ResponseRole struct {
		ID          string `json:"id" gorm:"unique;default:gen_random_uuid()"`
		Code        string `json:"code"`
		Name        string `json:"name"`
		Description string `json:"description"`
		IsActive    uint8  `json:"is_active"`
		base.BaseModelMaster
	}
)

func (Role) TableName() string {
	return "mroles"
}
