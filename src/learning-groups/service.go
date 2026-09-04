package learning_groups

import (
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LearningGroupService struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *LearningGroupService {
	return &LearningGroupService{DB: db}
}

func (s *LearningGroupService) getAll(ctx *gin.Context, dto DefaultFindDTO) (*helpers.PaginatedResult, error) {
	payload := ctx.Request.URL.Query()
	params := make(map[string]interface{})
	for key, values := range payload {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	// filters
	params["lg.deleted_at.isnull"] = ""

	baseQuery := `
		select
			lg.*,
			i.name as institution_name,
			string_agg(u.name, ', ') as instructor_name
		from learning_groups lg
		join institutions i on i.id = lg.institution_id
		left join learning_group_members lgm on lgm.learning_group_id = lg.id and role_in_group = 'instructor'
		left join users u on u.id = lgm.user_id
	`

	result, err := helpers.BuildPaginatedQuery(
		ctx,
		s.DB,                     // DB
		params,                   // filter
		"learning_groups",        // table name
		baseQuery,                // optional base query
		"group by lg.id, i.name", // optional query group by
		"",                       // optional select fields
		dto.SortBy,               // optional default sort
	)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *LearningGroupService) getByID(id string) (*models.LearningGroup, error) {
	var data models.LearningGroup
	err := s.DB.Preload("Institution").First(&data, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &data, err
}

func (s *LearningGroupService) create(dto CreateDTO) (*models.LearningGroup, error) {
	var count int64
	s.DB.Model(&models.LearningGroup{}).Where("name = ?", dto.Name).Count(&count)
	if count > 0 {
		return nil, errors.New("learning group name already exists")
	}

	InstID, err := uuid.Parse(dto.InstitutionID)
	if err != nil {
		return nil, errors.New("invalid role_id format")
	}

	data := models.LearningGroup{
		Name:          dto.Name,
		InstitutionID: InstID,
		Code:          dto.Code,
		Type:          models.LearningGroupType(dto.Type),
		Level:         dto.Level,
		Major:         dto.Major,
		Department:    dto.Department,
		AcademicYear:  dto.AcademicYear,
		IsActive:      dto.IsActive,
	}

	if err := s.DB.Create(&data).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *LearningGroupService) update(id string, dto UpdateDTO) (*models.LearningGroup, error) {
	var data models.LearningGroup

	if err := s.DB.First(&data, "id = ?", id).Error; err != nil {
		return nil, errors.New("learning group not found")
	}

	if dto.InstitutionID != nil {
		id, err := uuid.Parse(*dto.InstitutionID)
		if err != nil {
			return nil, errors.New("invalid institution_id format")
		}

		data.InstitutionID = id
	}

	if dto.Code != nil {
		data.Code = *dto.Code
	}

	if dto.Type != nil {
		data.Type = models.LearningGroupType(*dto.Type)
	}

	if dto.Level != nil {
		data.Level = *dto.Level
	}

	if dto.Major != nil {
		data.Major = *dto.Major
	}

	if dto.Department != nil {
		data.Department = *dto.Department
	}

	if dto.AcademicYear != nil {
		data.AcademicYear = *dto.AcademicYear
	}

	if dto.IsActive != nil {
		data.IsActive = *dto.IsActive
	}

	if dto.Name != nil && data.Name != *dto.Name {
		var exists int64
		s.DB.Model(&models.LearningGroup{}).Where("name = ? and id != ?", dto.Name, id).Count(&exists)
		if exists > 0 {
			return nil, errors.New("learning group name already exists")
		}
		data.Name = *dto.Name
	}

	if err := s.DB.Save(&data).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *LearningGroupService) archive(id string) (bool, error) {
	result := s.DB.Where("id = ?", id).Delete(&models.LearningGroup{})
	return result.RowsAffected > 0, result.Error
}

func (s *LearningGroupService) delete(id string) (bool, error) {
	result := s.DB.Unscoped().Where("id = ?", id).Delete(&models.LearningGroup{})
	return result.RowsAffected > 0, result.Error
}
