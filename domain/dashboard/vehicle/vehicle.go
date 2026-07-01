package vehicle

import (
	"erp-cbqa-global/domain/dashboard/vehicle/model"
	"erp-cbqa-global/domain/dashboard/vehicle/repository"
	"errors"
	"net/http"
	"strings"

	"gorm.io/gorm"
)

type VehicleServiceInterface interface {
	CreateVehicle(reqCreate model.ReqCreateVehicle) (int, error)
	GetListVehicle(search, byTypeId, byFacilityTypeId, byDcId, bySpId, sortBy, sort string, offset, limit int) (*[]model.ResVehicle, int64, int, error)
	GetListVehicleNoLimit(search, byTypeId, byFacilityTypeId, byDcId, bySpId, sortBy, sort string) (*[]model.ResVehicle, int, error)
	GetVehicleById(id string) (model.ResVehicle, int, error)
	UpdateVehicle(id string, reqUpdate model.ReqUpdateVehicle) (int, error)
	DeleteVehicle(id string) (int, error)
}

type vehicleService struct {
	VehicleRepo repository.VehicleRepositoryInterface
}

func NewVehicleService(vehicleRepo repository.VehicleRepositoryInterface) VehicleServiceInterface {
	return &vehicleService{
		VehicleRepo: vehicleRepo,
	}
}

// CreateVehicle is
func (s *vehicleService) CreateVehicle(reqCreate model.ReqCreateVehicle) (int, error) {
	// mapping create vehicle
	reqCreateVehicle := model.Vehicle{
		TypeId:         reqCreate.TypeId,
		Brand:          strings.ToLower(reqCreate.Brand),
		PlateNumber:    strings.ToUpper(reqCreate.PlateNumber),
		FacilityTypeId: reqCreate.FacilityTypeId,
		FacilityId:     reqCreate.FacilityId,
	}
	// create vehicle
	err := s.VehicleRepo.CreateVehicle(reqCreateVehicle)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

// GetListVehicle is
func (s *vehicleService) GetListVehicle(search, byTypeId, byFacilityTypeId, byDcId, bySpId, sortBy, sort string, offset, limit int) (*[]model.ResVehicle, int64, int, error) {
	vehicles, totalRow, err := s.VehicleRepo.GetListVehicle(search, byTypeId, byFacilityTypeId, byDcId, bySpId, sortBy, sort, offset, limit)
	if err != nil {
		return nil, totalRow, http.StatusInternalServerError, err
	}
	return vehicles, totalRow, http.StatusOK, nil
}

// GetListVehicleNoLimit is
func (s *vehicleService) GetListVehicleNoLimit(search, byTypeId, byFacilityTypeId, byDcId, bySpId, sortBy, sort string) (*[]model.ResVehicle, int, error) {
	vehicles, err := s.VehicleRepo.GetListVehicleNoLimit(search, byTypeId, byFacilityTypeId, byDcId, bySpId, sortBy, sort)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return vehicles, http.StatusOK, nil
}

// GetVehicleById is
func (s *vehicleService) GetVehicleById(id string) (model.ResVehicle, int, error) {
	vehicle, err := s.VehicleRepo.GetVehicleById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return vehicle, http.StatusNotFound, err
		}
		return vehicle, http.StatusInternalServerError, err
	}
	return vehicle, http.StatusOK, nil
}

// UpdateVehicle is
func (s *vehicleService) UpdateVehicle(id string, reqUpdate model.ReqUpdateVehicle) (int, error) {
	// get vehicle
	_, err := s.VehicleRepo.GetVehicleById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusNotFound, err
		}
		return http.StatusInternalServerError, err
	}
	// mapping update vehicle
	reqUpdateParam := model.Vehicle{
		Id:             id,
		TypeId:         reqUpdate.TypeId,
		Brand:          reqUpdate.Brand,
		PlateNumber:    reqUpdate.PlateNumber,
		FacilityTypeId: reqUpdate.FacilityTypeId,
		FacilityId:     reqUpdate.FacilityId,
	}
	err = s.VehicleRepo.UpdateVehicle(reqUpdateParam)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

// DeleteVehicle is
func (s *vehicleService) DeleteVehicle(id string) (int, error) {
	// get vehicle
	_, err := s.VehicleRepo.GetVehicleById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusNotFound, err
		}
		return http.StatusInternalServerError, err
	}
	// delete vehicle
	err = s.VehicleRepo.DeleteVehicle(id)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}
