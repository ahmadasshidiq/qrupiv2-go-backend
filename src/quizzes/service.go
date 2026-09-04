package quizzes

import (
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type QuizService struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *QuizService {
	return &QuizService{DB: db}
}

func (s *QuizService) getAll(ctx *gin.Context, dto DefaultFindDTO) (*helpers.PaginatedResult, error) {
	payload := ctx.Request.URL.Query()
	params := make(map[string]interface{})
	for key, values := range payload {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	// filters
	params["q.deleted_at.isnull"] = ""

	baseQuery := `
		select
			q.*,
			i.name as institution_name,
			lg.name as learning_group_name,
			u.name as created_user_name,
			qs.score as quiz_score,
			qs.start_time as quiz_start_time,
			qs.end_time as quiz_end_time,
			coalesce(jsonb_array_length(q.quiz_questions), 0) as quiz_count_question,
			coalesce(jsonb_array_length(qs.answers), 0) as quiz_count_answer
		from quizzes q
		join institutions i on i.id = q.institution_id
		join learning_groups lg on lg.id = q.learning_group_id
		join users u on u.id = q.created_user_id
		left join quiz_sessions qs on qs.quiz_id = q.id and qs.status = 'submitted'
	`

	result, err := helpers.BuildPaginatedQuery(
		ctx,
		s.DB,       // DB
		params,     // filter
		"quizzes",  // table name
		baseQuery,  // optional base query
		"",         // optional query group by
		"",         // optional select fields
		dto.SortBy, // optional default sort
	)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *QuizService) getByID(id string) (*models.Quiz, error) {
	var data models.Quiz
	err := s.DB.Preload("Institution").Preload("LearningGroup").Preload("CreatedUser").First(&data, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &data, err
}

func (s *QuizService) create(dto CreateDTO) (*models.Quiz, error) {
	var count int64
	s.DB.Model(&models.Quiz{}).Where("title = ? and learning_group_id = ?", dto.Title, dto.LearningGroupID).Count(&count)
	if count > 0 {
		return nil, errors.New("quiz with the same title already exists in this learning group")
	}

	if !dto.StartTime.Before(dto.EndTime) {
		return nil, errors.New("start_time must be before end_time")
	}

	lgID, err := uuid.Parse(dto.LearningGroupID)
	if err != nil {
		return nil, errors.New("invalid learning_group_id format")
	}

	instID, err := uuid.Parse(dto.InstitutionID)
	if err != nil {
		return nil, errors.New("invalid institution_id format")
	}

	userID, err := uuid.Parse(dto.CreatedUserID)
	if err != nil {
		return nil, errors.New("invalid created_user_id format")
	}

	questions := make(models.QuizQuestions, len(dto.QuizQuestions))

	for i, q := range dto.QuizQuestions {
		options := make([]models.QuizOption, len(q.Options))

		for j, opt := range q.Options {
			options[j] = models.QuizOption{
				Label: opt.Label,
				Value: opt.Value,
			}
		}

		questions[i] = models.QuizQuestion{
			QuestionText:  q.QuestionText,
			Options:       options,
			CorrectAnswer: q.CorrectAnswer,
			Type:          q.Type,
			Points:        q.Points,
		}
	}

	data := models.Quiz{
		InstitutionID:          instID,
		LearningGroupID:        lgID,
		CreatedUserID:          userID,
		Title:                  dto.Title,
		Description:            dto.Description,
		QuizQuestions:          questions,
		StartTime:              dto.StartTime,
		EndTime:                dto.EndTime,
		DurationMinutes:        dto.DurationMinutes,
		MaxCheatingWarnings:    dto.MaxCheatingWarnings,
		CheatingPenaltyMinutes: dto.CheatingPenaltyMinutes,
		Type:                   models.QuizType(dto.Type),
	}

	if err := s.DB.Create(&data).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *QuizService) update(id string, dto UpdateDTO) (*models.Quiz, error) {
	var data models.Quiz

	if err := s.DB.First(&data, "id = ?", id).Error; err != nil {
		return nil, errors.New("quiz not found")
	}

	if dto.Title != nil {
		// Cek duplikat title di learning_group yang sama
		var exists int64
		s.DB.Model(&models.Quiz{}).
			Where("title = ? AND learning_group_id = ? AND id != ?", *dto.Title, data.LearningGroupID, id).
			Count(&exists)
		if exists > 0 {
			return nil, errors.New("quiz title already exists in this learning group")
		}
		data.Title = *dto.Title
	}

	if dto.Description != nil {
		data.Description = *dto.Description
	}

	if dto.QuizQuestions != nil {
		b, err := json.Marshal(dto.QuizQuestions)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal quiz questions: %w", err)
		}
		if err := json.Unmarshal(b, &data.QuizQuestions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal quiz questions: %w", err)
		}
	}

	if dto.StartTime != nil {
		data.StartTime = *dto.StartTime
	}

	if dto.EndTime != nil {
		data.EndTime = *dto.EndTime
	}

	// Pastikan start < end
	if !data.StartTime.Before(data.EndTime) {
		return nil, errors.New("start_time must be before end_time")
	}

	if dto.DurationMinutes != nil {
		data.DurationMinutes = *dto.DurationMinutes
	}

	if dto.MaxCheatingWarnings != nil {
		data.MaxCheatingWarnings = *dto.MaxCheatingWarnings
	}

	if dto.CheatingPenaltyMinutes != nil {
		data.CheatingPenaltyMinutes = *dto.CheatingPenaltyMinutes
	}

	if dto.Type != nil {
		data.Type = models.QuizType(*dto.Type)
	}

	if err := s.DB.Save(&data).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *QuizService) archive(id string) (bool, error) {
	result := s.DB.Where("id = ?", id).Delete(&models.Quiz{})
	return result.RowsAffected > 0, result.Error
}

func (s *QuizService) delete(id string) (bool, error) {
	result := s.DB.Unscoped().Where("id = ?", id).Delete(&models.Quiz{})
	return result.RowsAffected > 0, result.Error
}
