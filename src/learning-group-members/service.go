package learning_group_members

import (
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LearningGroupMemberService struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *LearningGroupMemberService {
	return &LearningGroupMemberService{DB: db}
}

func (s *LearningGroupMemberService) getAll(ctx *gin.Context, dto DefaultFindDTO) (*helpers.PaginatedResult, error) {
	payload := ctx.Request.URL.Query()
	params := make(map[string]interface{})
	for key, values := range payload {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	// filters
	params["lgm.deleted_at.isnull"] = ""

	baseQuery := `
		select
			lgm.*,
			lg.name as learning_group_name,
			u.name as user_name,
			u.avatar_url as user_avatar_url,
			u.email as user_email,
			i.name as institution_name
		from learning_group_members lgm
		join learning_groups lg on lg.id = lgm.learning_group_id
		join users u on u.id = lgm.user_id
		left join institutions i on i.id = lg.institution_id 
	`

	result, err := helpers.BuildPaginatedQuery(
		ctx,
		s.DB,                     // DB
		params,                   // filter
		"learning_group_members", // table name
		baseQuery,                // optional base query
		"",                       // optional query group by
		"",                       // optional select fields
		dto.SortBy,               // optional default sort
	)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *LearningGroupMemberService) getByID(id string) (*models.LearningGroupMember, error) {
	var data models.LearningGroupMember
	err := s.DB.Preload("LearningGroup").Preload("User").First(&data, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &data, err
}

func (s *LearningGroupMemberService) create(dto CreateDTO) (*models.LearningGroupMember, error) {
	lgID, err := uuid.Parse(dto.LearningGroupID)
	if err != nil {
		return nil, errors.New("invalid learning_group_id format")
	}

	userID, err := uuid.Parse(dto.UserID)
	if err != nil {
		return nil, errors.New("invalid user_id format")
	}

	var count int64
	s.DB.Model(&models.LearningGroupMember{}).Where(
		"learning_group_id = ? and user_id = ?",
		lgID,
		userID,
	).Count(&count)

	if count > 0 {
		return nil, errors.New("user already exists in this learning group")
	}

	data := models.LearningGroupMember{
		LearningGroupID: lgID,
		UserID:          userID,
		RoleInGroup:     models.RoleInGroup(dto.RoleInGroup),
	}

	if err := s.DB.Create(&data).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *LearningGroupMemberService) update(id string, dto UpdateDTO) (*models.LearningGroupMember, error) {
	var data models.LearningGroupMember

	if err := s.DB.First(&data, "id = ?", id).Error; err != nil {
		return nil, errors.New("learning group member not found")
	}

	if dto.LearningGroupID != nil {
		id, err := uuid.Parse(*dto.LearningGroupID)
		if err != nil {
			return nil, errors.New("invalid learning_group_id format")
		}

		data.LearningGroupID = id
	}

	if dto.UserID != nil {
		id, err := uuid.Parse(*dto.UserID)
		if err != nil {
			return nil, errors.New("invalid user_id format")
		}

		data.UserID = id
	}

	if dto.RoleInGroup != nil {
		data.RoleInGroup = models.RoleInGroup(*dto.RoleInGroup)
	}

	var count int64
	s.DB.Model(&models.LearningGroupMember{}).
		Where("learning_group_id = ? and user_id = ? AND id <> ?", data.LearningGroupID, data.UserID, data.ID).
		Count(&count)
	if count > 0 {
		return nil, errors.New("user already exists in this learning group")
	}

	if err := s.DB.Save(&data).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *LearningGroupMemberService) archive(id string) (bool, error) {
	result := s.DB.Where("id = ?", id).Delete(&models.LearningGroupMember{})
	return result.RowsAffected > 0, result.Error
}

func (s *LearningGroupMemberService) delete(id string) (bool, error) {
	result := s.DB.Unscoped().Where("id = ?", id).Delete(&models.LearningGroupMember{})
	return result.RowsAffected > 0, result.Error
}
