package repository

import (
	"erp-cbqa-global/domain/dashboard/vehicle/model"
	"fmt"

	"gorm.io/gorm"
)

type VehicleRepositoryInterface interface {
	CreateVehicle(reqCreate model.Vehicle) error
	GetListVehicle(search, byTypeId, byFacilityTypeId, byDcId, bySpId, sortBy, sort string, offset, limit int) (*[]model.ResVehicle, int64, error)
	GetListVehicleNoLimit(search, byTypeId, byFacilityTypeId, byDcId, bySpId, sortBy, sort string) (*[]model.ResVehicle, error)
	GetVehicleById(id string) (model.ResVehicle, error)
	UpdateVehicle(reqUpdate model.Vehicle) error
	DeleteVehicle(id string) error
}

type vehicleRepository struct {
	DB *gorm.DB
}

func NewVehicleRepository(db *gorm.DB) VehicleRepositoryInterface {
	return &vehicleRepository{
		DB: db,
	}
}

// CreateVehicle is
func (r *vehicleRepository) CreateVehicle(reqCreate model.Vehicle) error {
	// create vehicle
	if err := r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("md_vehicles").
			Create(&reqCreate).Error; err != nil {
			tx.Rollback()
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// GetListVehicle is
func (r *vehicleRepository) GetListVehicle(search, byTypeId, byFacilityTypeId, byDcId, bySpId, sortBy, sort string, offset, limit int) (*[]model.ResVehicle, int64, error) {
	var (
		vehicle  []model.ResVehicle
		totalRow int64
	)
	// filter get vehicle list
	whereCondition := `md_vehicles.deleted_at IS NULL`
	joinTable := `LEFT JOIN scope_details facility_type ON facility_type.id = md_vehicles.facility_type_id ` +
		`LEFT JOIN scope_details type ON type.id = md_vehicles.type_id ` +
		`LEFT JOIN (SELECT dc.id   AS id,
                           dc.name AS facility_name,
                           'DC'    AS facility_type
                    FROM md_distribution_centers dc
                    WHERE dc.is_dc = 1
                    UNION ALL
                    SELECT sp.id   AS id,
                           sp.name AS facility_name,
                           'SP'    AS facility_type
                    FROM md_stockpoints sp) facility ON facility.id = md_vehicles.facility_id`
	if search != "" {
		whereCondition += fmt.Sprintf(` AND md_vehicles.plate_number ILIKE '%v'`, `%`+search+`%`)
	}
	if byTypeId != "" {
		whereCondition += fmt.Sprintf(` AND md_vehicles.type_id = '%v'`, byTypeId)
	}
	if byFacilityTypeId != "" {
		whereCondition += fmt.Sprintf(` AND md_vehicles.facility_type_id = '%v'`, byFacilityTypeId)
	}
	if byDcId != "" {
		whereCondition += fmt.Sprintf(` AND md_vehicles.facility_id = '%v'`, byDcId)
	}
	if bySpId != "" {
		whereCondition += fmt.Sprintf(` AND md_vehicles.facility_id = '%v'`, bySpId)
	}
	// count total vehicle
	if err := r.DB.
		Table("md_vehicles").
		Joins(joinTable).
		Where(whereCondition).
		Count(&totalRow).
		Error; err != nil {
		return nil, 0, err
	}
	// get vehicle list
	if err := r.DB.
		Table("md_vehicles").
		Select(`md_vehicles.*, type.name AS type, facility_type.name AS facility_type, facility.facility_name AS facility_name`).
		Joins(joinTable).
		Where(whereCondition).
		Order(sortBy + ` ` + sort).
		Limit(limit).
		Offset(offset).
		Find(&vehicle).
		Error; err != nil {
		return nil, 0, err
	}
	return &vehicle, totalRow, nil
}

// GetListVehicleNoLimit is
func (r *vehicleRepository) GetListVehicleNoLimit(search, byTypeId, byFacilityTypeId, byDcId, bySpId, sortBy, sort string) (*[]model.ResVehicle, error) {
	var vehicle []model.ResVehicle
	// filter get vehicle list
	whereCondition := `md_vehicles.deleted_at IS NULL`
	joinTable := `LEFT JOIN scope_details facility_type ON facility_type.id = md_vehicles.facility_type_id ` +
		`LEFT JOIN scope_details type ON type.id = md_vehicles.type_id ` +
		`LEFT JOIN (SELECT dc.id   AS id,
                           dc.name AS facility_name,
                           'DC'    AS facility_type
                    FROM md_distribution_centers dc
                    WHERE dc.is_dc = 1
                    UNION ALL
                    SELECT sp.id   AS id,
                           sp.name AS facility_name,
                           'SP'    AS facility_type
                    FROM md_stockpoints sp) facility ON facility.id = md_vehicles.facility_id`
	if search != "" {
		whereCondition += fmt.Sprintf(` AND md_vehicles.plate_number ILIKE '%v'`, `%`+search+`%`)
	}
	if byTypeId != "" {
		whereCondition += fmt.Sprintf(` AND md_vehicles.type_id = '%v'`, byTypeId)
	}
	if byFacilityTypeId != "" {
		whereCondition += fmt.Sprintf(` AND md_vehicles.facility_type_id = '%v'`, byFacilityTypeId)
	}
	if byDcId != "" {
		whereCondition += fmt.Sprintf(` AND md_vehicles.facility_id = '%v'`, byDcId)
	}
	if bySpId != "" {
		whereCondition += fmt.Sprintf(` AND md_vehicles.facility_id = '%v'`, bySpId)
	}
	// get vehicle list
	if err := r.DB.
		Table("md_vehicles").
		Select(`md_vehicles.*, type.name AS type, facility_type.name AS facility_type, facility.facility_name AS facility_name`).
		Joins(joinTable).
		Where(whereCondition).
		Order(sortBy + ` ` + sort).
		Find(&vehicle).
		Error; err != nil {
		return nil, err
	}
	return &vehicle, nil
}

// GetVehicleById is
func (r *vehicleRepository) GetVehicleById(id string) (model.ResVehicle, error) {
	var vehicle model.ResVehicle
	joinTable := `LEFT JOIN scope_details facility_type ON facility_type.id = md_vehicles.facility_type_id ` +
		`LEFT JOIN scope_details type ON type.id = md_vehicles.type_id ` +
		`LEFT JOIN (SELECT dc.id   AS id,
                           dc.name AS facility_name,
                           'DC'    AS facility_type
                    FROM md_distribution_centers dc
                    WHERE dc.is_dc = 1
                    UNION ALL
                    SELECT sp.id   AS id,
                           sp.name AS facility_name,
                           'SP'    AS facility_type
                    FROM md_stockpoints sp) facility ON facility.id = md_vehicles.facility_id`
	if err := r.DB.
		Table("md_vehicles").
		Select(`md_vehicles.*, type.name AS type, facility_type.name AS facility_type, facility.facility_name AS facility_name`).
		Joins(joinTable).
		Where(`md_vehicles.id = ?`, id).
		First(&vehicle).
		Error; err != nil {
		return vehicle, err
	}
	return vehicle, nil
}

// UpdateVehicle is
func (r *vehicleRepository) UpdateVehicle(reqUpdate model.Vehicle) error {
	if err := r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("md_vehicles").
			Select("id", "type_id", "brand", "plate_number", "facility_type_id", "facility_id").
			Updates(&reqUpdate).Error; err != nil {
			tx.Rollback()
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// DeleteVehicle is
func (r *vehicleRepository) DeleteVehicle(id string) error {
	if err := r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("md_vehicles").
			Where(`md_vehicles.id = ?`, id).
			Delete(&model.Vehicle{}).Error; err != nil {
			tx.Rollback()
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
