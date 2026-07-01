package base

import (
	"time"

	"gorm.io/gorm"
)

// BaseModelMaster adalah base struct untuk semua model dengan audit trail
type BaseModelMaster struct {
	IsActive  uint8           `json:"is_active" gorm:"column:is_active;default:1"`
	CreatedBy string          `json:"created_by" gorm:"column:created_by;"`
	UpdatedBy string          `json:"updated_by" gorm:"column:updated_by;"`
	DeletedBy *string         `json:"deleted_by" gorm:"column:deleted_by;"`
	CreatedAt time.Time       `json:"created_at" gorm:"column:created_at;"`
	UpdatedAt time.Time       `json:"updated_at" gorm:"column:updated_at;"`
	DeletedAt *gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;"`
}
