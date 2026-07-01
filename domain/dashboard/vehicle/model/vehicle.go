package model

import (
	"gorm.io/gorm"
	"time"
)

type (
	Vehicle struct {
		Id             string          `json:"id" gorm:"unique;default:gen_random_uuid()"`
		TypeId         string          `json:"type_id"`
		Brand          string          `json:"brand"`
		PlateNumber    string          `json:"plate_number"`
		FacilityTypeId string          `json:"facility_type_id"`
		FacilityId     string          `json:"facility_id"`
		CreatedAt      time.Time       `json:"created_at"`
		UpdatedAt      time.Time       `json:"updated_at"`
		DeletedAt      *gorm.DeletedAt `json:"deleted_at"`
	}

	ResVehicle struct {
		Id             string          `json:"id"`
		TypeId         string          `json:"type_id"`
		Type           string          `json:"type"`
		Brand          string          `json:"brand"`
		PlateNumber    string          `json:"plate_number"`
		FacilityTypeId string          `json:"facility_type_id"`
		FacilityType   string          `json:"facility_type"`
		FacilityId     string          `json:"facility_id"`
		FacilityName   string          `json:"facility_name"`
		CreatedAt      time.Time       `json:"created_at"`
		UpdatedAt      time.Time       `json:"updated_at"`
		DeletedAt      *gorm.DeletedAt `json:"deleted_at"`
	}

	ReqCreateVehicle struct {
		TypeId         string `json:"type_id"`
		Brand          string `json:"brand"`
		PlateNumber    string `json:"plate_number"`
		FacilityTypeId string `json:"facility_type_id"`
		FacilityId     string `json:"facility_id"`
	}

	ReqUpdateVehicle struct {
		TypeId         string `json:"type_id"`
		Brand          string `json:"brand"`
		PlateNumber    string `json:"plate_number"`
		FacilityTypeId string `json:"facility_type_id"`
		FacilityId     string `json:"facility_id"`
	}
)
