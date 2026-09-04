package quiz_sessions

import (
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type QuizSessionService struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *QuizSessionService {
	return &QuizSessionService{DB: db}
}

func (s *QuizSessionService) getAll(ctx *gin.Context, dto DefaultFindDTO) (*helpers.PaginatedResult, error) {
	payload := ctx.Request.URL.Query()
	params := make(map[string]interface{})
	for key, values := range payload {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	// filters
	params["qs.deleted_at.isnull"] = ""

	baseQuery := `
		select
			qs.*,
			q.title as quiz_title,
			u.name as user_name,
			i.name as institution_name,
			lg.name as learning_group_name
		from quiz_sessions qs
		join quizzes q on q.id = qs.quiz_id
		join users u on u.id = qs.user_id
		join learning_groups lg on lg.id = q.learning_group_id
		join institutions i on i.id = lg.institution_id
	`

	result, err := helpers.BuildPaginatedQuery(
		ctx,
		s.DB,            // DB
		params,          // filter
		"quiz_sessions", // table name
		baseQuery,       // optional base query
		"",              // optional query group by
		"",              // optional select fields
		dto.SortBy,      // optional default sort
	)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *QuizSessionService) getByID(id string) (*models.QuizSession, error) {
	var data models.QuizSession
	err := s.DB.Preload("Quiz").Preload("User").First(&data, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &data, err
}

func (s *QuizSessionService) create(dto CreateDTO) (*models.QuizSession, error) {
	var count int64
	s.DB.Model(&models.QuizSession{}).Where("quiz_id = ? and user_id = ?", dto.QuizID, dto.UserID).Count(&count)
	if count > 0 {
		return nil, errors.New("quiz session already exists")
	}

	quizID, err := uuid.Parse(dto.QuizID)
	if err != nil {
		return nil, errors.New("invalid quiz_id format")
	}

	userID, err := uuid.Parse(dto.UserID)
	if err != nil {
		return nil, errors.New("invalid created_user_id format")
	}

	data := models.QuizSession{
		QuizID:     quizID,
		UserID:     userID,
		StartTime:  dto.StartTime,
		DeviceInfo: dto.DeviceInfo,
	}

	if err := s.DB.Create(&data).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *QuizSessionService) archive(id string) (bool, error) {
	result := s.DB.Where("id = ?", id).Delete(&models.QuizSession{})
	return result.RowsAffected > 0, result.Error
}

func (s *QuizSessionService) delete(id string) (bool, error) {
	result := s.DB.Unscoped().Where("id = ?", id).Delete(&models.QuizSession{})
	return result.RowsAffected > 0, result.Error
}

func (s *QuizSessionService) update(id string, dto UpdateDTO) (*models.QuizSession, error) {
	var data models.QuizSession

	if err := s.DB.First(&data, "id = ?", id).Error; err != nil {
		return nil, errors.New("quiz session not found")
	}

	if dto.EndTime != nil {
		data.EndTime = *dto.EndTime
	}

	if dto.Status != nil {
		data.Status = models.QuizSessionStatus(*dto.Status)
	}

	if dto.CheatingCount != nil {
		data.CheatingCount = *dto.CheatingCount
	}

	if dto.CheatingLogs != nil {
		b, err := json.Marshal(dto.CheatingLogs)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal cheating logs: %w", err)
		}
		data.CheatingLogs = datatypes.JSON(b)
	}

	if dto.Answers != nil {
		b, err := json.Marshal(dto.Answers)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal answers: %w", err)
		}
		data.Answers = datatypes.JSON(b)
	}

	if dto.Score != nil {
		data.Score = *dto.Score
	}

	if dto.DeviceInfo != nil {
		data.DeviceInfo = *dto.DeviceInfo
	}

	if !data.EndTime.IsZero() && !data.StartTime.Before(data.EndTime) {
		return nil, errors.New("end_time must be after start_time")
	}

	if err := s.DB.Save(&data).Error; err != nil {
		return nil, err
	}

	return &data, nil
}
