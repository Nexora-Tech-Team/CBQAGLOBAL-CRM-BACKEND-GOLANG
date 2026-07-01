package model

import "time"

type Module struct {
	ID        string    `json:"id" gorm:"unique;default:gen_random_uuid()"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	OrderNo   int       `json:"order_no"`
	IsActive  int       `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Module) TableName() string {
	return "mmodules"
}

type Menu struct {
	ID        string    `json:"id" gorm:"unique;default:gen_random_uuid()"`
	ModuleID  string    `json:"module_id"`
	ParentID  string    `json:"parent_id"`
	Name      string    `json:"name"`
	Route     string    `json:"route"`
	Icon      string    `json:"icon"`
	OrderNo   int       `json:"order_no"`
	IsActive  int       `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Menu) TableName() string {
	return "mmenus"
}
