package learning_groups

import (
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	"errors"
	"fmt"

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

	params["lg.deleted_at.isnull"] = ""
	memberUserID, isMember, err := s.authenticatedMemberID(ctx)
	if err != nil {
		return nil, err
	}
	if institutionID := ctx.GetString("institution_id"); institutionID != "" && !s.isSuperAdmin(ctx) {
		params["lg.institution_id"] = institutionID
	}
	if err := helpers.ApplyRegionScope(ctx, params, "i"); err != nil {
		return nil, err
	}

	memberJoin := ""
	studentJoin := ""
	groupBy := "group by lg.id, i.name, student_lgm.user_id"
	if isMember {
		memberID, parseErr := uuid.Parse(memberUserID)
		if parseErr != nil {
			return nil, errors.New("invalid authenticated user id")
		}
		studentJoin = fmt.Sprintf("left join learning_group_members student_lgm on student_lgm.learning_group_id = lg.id and student_lgm.role_in_group = 'student' and student_lgm.user_id = '%s' and student_lgm.deleted_at is null", memberID.String())
		memberJoin = fmt.Sprintf("join learning_group_members member_scope_lgm on member_scope_lgm.learning_group_id = lg.id and member_scope_lgm.user_id = '%s' and member_scope_lgm.deleted_at is null", memberID.String())
	} else {
		studentJoin = "left join learning_group_members student_lgm on false"
	}

	baseQuery := fmt.Sprintf(`
		select
			lg.*,
			student_lgm.user_id as student_member_user_id,
			i.name as institution_name,
			string_agg(u.name, ', ') as instructor_name
		from learning_groups lg
		join institutions i on i.id = lg.institution_id
		%s
		%s
		left join learning_group_members lgm on lgm.learning_group_id = lg.id and lgm.role_in_group = 'instructor' and lgm.deleted_at is null
		left join users u on u.id = lgm.user_id
	`, studentJoin, memberJoin)

	result, err := helpers.BuildPaginatedQuery(
		ctx,
		s.DB,
		params,
		"learning_groups",
		baseQuery,
		groupBy,
		"",
		dto.SortBy,
	)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *LearningGroupService) isSuperAdmin(ctx *gin.Context) bool {
	var roleName string
	s.DB.Table("roles").Select("lower(replace(name, '-', '_'))").Where("id = ?", ctx.GetString("role_id")).Scan(&roleName)
	return helpers.IsRole(roleName, "super_admin")
}

func (s *LearningGroupService) getByID(id string) (*models.LearningGroup, error) {
	var data models.LearningGroup
	err := s.DB.Preload("Institution").First(&data, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &data, err
}

func (s *LearningGroupService) getByIDForUser(ctx *gin.Context, id string) (*models.LearningGroup, error) {
	query := s.DB.Where("learning_groups.id = ?", id)
	memberUserID, isMember, err := s.authenticatedMemberID(ctx)
	if err != nil {
		return nil, err
	}
	if isMember {
		query = query.Joins("JOIN learning_group_members member_scope_lgm ON member_scope_lgm.learning_group_id = learning_groups.id AND member_scope_lgm.user_id = ? AND member_scope_lgm.deleted_at IS NULL", memberUserID)
	}
	var data models.LearningGroup
	err = query.Preload("Institution").First(&data).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &data, err
}

func (s *LearningGroupService) authenticatedMemberID(ctx *gin.Context) (string, bool, error) {
	userID := ctx.GetString("user_id")
	if userID == "" {
		return "", false, errors.New("user_id is missing from authenticated session")
	}

	var role string
	if err := s.DB.Table("roles").
		Select("lower(replace(name, '-', '_'))").
		Where("id = ?", ctx.GetString("role_id")).
		Scan(&role).Error; err != nil {
		return "", false, err
	}
	if !helpers.IsRole(role, "student") && !helpers.IsRole(role, "instructor") && !helpers.IsRole(role, "teacher") {
		return "", false, nil
	}
	return userID, true, nil
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
		Status:        models.LearningGroupStatus(dto.Status),
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

	if dto.Status != nil {
		data.Status = models.LearningGroupStatus(*dto.Status)
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
