package regions

import (
	"errors"

	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	"gorm.io/gorm"
)

type RegionService struct{ DB *gorm.DB }

func NewService(db *gorm.DB) *RegionService { return &RegionService{DB: db} }

func (s *RegionService) getProvinces(dto FindDTO) (*helpers.PaginatedResult, error) {
	return s.list(dto, "provinces", "p", "", "")
}

func (s *RegionService) getRegencies(dto FindDTO) (*helpers.PaginatedResult, error) {
	return s.list(dto, "regencies", "r", "province_code", dto.ProvinceCode)
}

func (s *RegionService) getDistricts(dto FindDTO) (*helpers.PaginatedResult, error) {
	return s.list(dto, "districts", "d", "regency_code", dto.RegencyCode)
}

func (s *RegionService) getVillages(dto FindDTO) (*helpers.PaginatedResult, error) {
	return s.list(dto, "villages", "v", "district_code", dto.DistrictCode)
}

func (s *RegionService) list(dto FindDTO, table, alias, parentField, parentCode string) (*helpers.PaginatedResult, error) {
	limit := dto.Limit
	if limit == 0 {
		limit = 100
	}
	sortOrder := dto.SortOrder
	if sortOrder == "" {
		sortOrder = "asc"
	}
	params := map[string]interface{}{
		alias + ".is_active": true,
		"limit":              limit,
		"sortOrder":          sortOrder,
	}
	if dto.Page > 0 {
		params["page"] = dto.Page
	}
	if dto.SortBy != "" {
		params["sortBy"] = dto.SortBy
	}
	if dto.Search != "" {
		params[alias+".name.ilike"] = dto.Search
	}
	if parentCode != "" {
		params[alias+"."+parentField] = parentCode
	}
	return helpers.BuildPaginatedQuery(nil, s.DB, params, table, "select "+alias+".* from "+table+" "+alias, "", "", "name")
}

func (s *RegionService) getProvince(code string) (*models.Province, error) {
	data := models.Province{}
	return dataOrNil(&data, s.DB.Where("code = ? AND is_active = true", code).First(&data).Error)
}

func (s *RegionService) getRegency(code string) (*models.Regency, error) {
	data := models.Regency{}
	return dataOrNil(&data, s.DB.Where("code = ? AND is_active = true", code).First(&data).Error)
}

func (s *RegionService) getDistrict(code string) (*models.District, error) {
	data := models.District{}
	return dataOrNil(&data, s.DB.Where("code = ? AND is_active = true", code).First(&data).Error)
}

func (s *RegionService) getVillage(code string) (*models.Village, error) {
	data := models.Village{}
	return dataOrNil(&data, s.DB.Where("code = ? AND is_active = true", code).First(&data).Error)
}

func dataOrNil[T any](data *T, err error) (*T, error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}
