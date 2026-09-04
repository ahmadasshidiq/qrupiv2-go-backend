package roles

import (
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	"clasenna-go-backend/libs/seeder"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type RoleService struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *RoleService {
	return &RoleService{DB: db}
}

func (s *RoleService) getAll(ctx *gin.Context, dto DefaultFindDTO) (*helpers.PaginatedResult, error) {
	payload := ctx.Request.URL.Query()
	params := make(map[string]interface{})
	for key, values := range payload {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	// filters
	params["roles.deleted_at.isnull"] = ""

	result, err := helpers.BuildPaginatedQuery(
		ctx,
		s.DB,       // DB
		params,     // filter
		"roles",    // table name
		"",         // optional base query
		"",         // optional query group by
		"",         // optional select fields
		dto.SortBy, // optional default sort
	)

	if err != nil {
		return nil, err
	}

	rows := result.Data

	for _, row := range rows {
		rawPerms, ok := row["permissions"].(string)
		if !ok || rawPerms == "" {
			row["permissions"] = []interface{}{}
			continue
		}

		var parsed []models.PermissionItem
		if err := json.Unmarshal([]byte(rawPerms), &parsed); err != nil {
			log.Printf("[RoleService] invalid permissions JSON: %v", err)
			row["permissions"] = []interface{}{}
			continue
		}
		row["permissions"] = parsed
	}

	return result, nil
}

func (s *RoleService) getMasterPermissions() ([]models.GroupedPermission, error) {
	masterPermissions := seeder.MasterPermissions

	groupMap := make(map[string][]models.PermissionItem)

	// Kelompokkan berdasarkan model
	for _, p := range masterPermissions {
		groupMap[p.Model] = append(groupMap[p.Model], p)
	}

	// Ubah ke dalam bentuk array GroupedPermission
	var grouped []models.GroupedPermission
	for model, perms := range groupMap {
		grouped = append(grouped, models.GroupedPermission{
			Title:       strings.Title(model) + " Permission",
			Permissions: perms,
		})
	}

	return grouped, nil
}

func (s *RoleService) getByID(id string) (*models.Role, error) {
	var data models.Role
	stmtQuery := fmt.Sprintf(`select * from roles where id = '%s'`, id)
	err := s.DB.Raw(stmtQuery).First(&data).Error
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *RoleService) create(dto CreateDTO) (*models.Role, error) {
	var count int64
	s.DB.Model(&models.Role{}).Where("name = ?", dto.Name).Count(&count)
	if count > 0 {
		return nil, errors.New("role name already exists")
	}

	if len(dto.Permissions) == 0 {
		return nil, errors.New("at least one permission is required")
	}

	permsJSON, err := json.Marshal(dto.Permissions)
	if err != nil {
		return nil, fmt.Errorf("invalid permissions format: %v", err)
	}

	data := models.Role{
		Name:        dto.Name,
		Permissions: datatypes.JSON(permsJSON),
		Description: dto.Description,
	}

	if err := s.DB.Create(&data).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *RoleService) update(id string, dto UpdateDTO) (*models.Role, error) {
	var data models.Role

	if err := s.DB.First(&data, "id = ?", id).Error; err != nil {
		return nil, errors.New("role not found")
	}

	if dto.Description != nil {
		data.Description = *dto.Description
	}

	if dto.Name != nil && data.Name != *dto.Name {
		var exists int64
		s.DB.Model(&models.Role{}).Where("name = ? and id != ?", dto.Name, id).Count(&exists)
		if exists > 0 {
			return nil, errors.New("role name already exists")
		}
		data.Name = *dto.Name
	}

	if dto.Permissions != nil {
		if len(*dto.Permissions) == 0 {
			return nil, errors.New("at least one permission is required")
		}
		permsJSON, err := json.Marshal(dto.Permissions)
		if err != nil {
			return nil, fmt.Errorf("invalid permissions format: %v", err)
		}
		data.Permissions = datatypes.JSON(permsJSON)
	}

	if err := s.DB.Save(&data).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *RoleService) archive(id string) (bool, error) {
	result := s.DB.Where("id = ?", id).Delete(&models.Role{})
	return result.RowsAffected > 0, result.Error
}

func (s *RoleService) delete(id string) (bool, error) {
	result := s.DB.Unscoped().Where("id = ?", id).Delete(&models.Role{})
	return result.RowsAffected > 0, result.Error
}
