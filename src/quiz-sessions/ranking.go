package quiz_sessions

import (
	"errors"
	"time"

	"clasenna-go-backend/libs/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StudentRanking struct {
	Rank         int64     `json:"rank"`
	UserID       uuid.UUID `json:"user_id"`
	UserName     string    `json:"user_name"`
	AvatarURL    string    `json:"avatar_url"`
	TotalQuizzes int64     `json:"total_quizzes"`
	TotalScore   float64   `json:"total_score"`
	AverageScore float64   `json:"average_score"`
	HighestScore float64   `json:"highest_score"`
}

type QuizRankingResponse struct {
	Scope             string           `json:"scope"`
	InstitutionID     uuid.UUID        `json:"institution_id"`
	LearningGroupID   *uuid.UUID       `json:"learning_group_id,omitempty"`
	LearningGroupName string           `json:"learning_group_name,omitempty"`
	Rankings          []StudentRanking `json:"rankings"`
}

type parsedRankingFilter struct {
	institutionID   uuid.UUID
	learningGroupID *uuid.UUID
	quizID          *uuid.UUID
	startDate       *time.Time
	endDate         *time.Time
	limit           int
}

func (s *QuizSessionService) getRankings(ctx *gin.Context, dto RankingFilterDTO) (*QuizRankingResponse, error) {
	filter, err := parseRankingFilter(ctx, dto)
	if err != nil {
		return nil, err
	}

	result := &QuizRankingResponse{
		Scope:           dto.Scope,
		InstitutionID:   filter.institutionID,
		LearningGroupID: filter.learningGroupID,
		Rankings:        make([]StudentRanking, 0),
	}
	if filter.learningGroupID != nil {
		var group models.LearningGroup
		query := s.DB.Select("id", "name", "type").Where("id = ? AND institution_id = ?", *filter.learningGroupID, filter.institutionID)
		if dto.Scope == "class" {
			query = query.Where("type IN ?", []models.LearningGroupType{
				models.LearningGroupTypeSchoolClass,
				models.LearningGroupTypeUniversityClass,
			})
		}
		if err := query.First(&group).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("learning group not found for the requested scope")
		} else if err != nil {
			return nil, err
		}
		result.LearningGroupName = group.Name
	}

	aggregate := rankingAggregateQuery(s.DB.WithContext(ctx.Request.Context()), filter).
		Select(`u.id AS user_id, u.name AS user_name, u.avatar_url,
			COUNT(*) AS total_quizzes,
			COALESCE(SUM(qs.score), 0)::float8 AS total_score,
			COALESCE(AVG(qs.score), 0)::float8 AS average_score,
			COALESCE(MAX(qs.score), 0)::float8 AS highest_score`).
		Group("u.id, u.name, u.avatar_url")

	if err := s.DB.WithContext(ctx.Request.Context()).
		Table("(?) AS student_scores", aggregate).
		Select(`DENSE_RANK() OVER (ORDER BY average_score DESC, total_score DESC) AS rank,
			user_id, user_name, avatar_url, total_quizzes, total_score, average_score, highest_score`).
		Order("rank ASC, user_name ASC").
		Limit(filter.limit).
		Scan(&result.Rankings).Error; err != nil {
		return nil, err
	}
	return result, nil
}

func parseRankingFilter(ctx *gin.Context, dto RankingFilterDTO) (parsedRankingFilter, error) {
	institutionID, err := uuid.Parse(ctx.GetString("institution_id"))
	if err != nil {
		return parsedRankingFilter{}, errors.New("institution_id is required in authenticated session")
	}
	filter := parsedRankingFilter{institutionID: institutionID, limit: dto.Limit}
	if filter.limit == 0 {
		filter.limit = 50
	}
	if dto.Scope != "school" && dto.LearningGroupID == "" {
		return parsedRankingFilter{}, errors.New("learning_group_id is required for class or learning_group scope")
	}
	if dto.LearningGroupID != "" {
		id, err := uuid.Parse(dto.LearningGroupID)
		if err != nil {
			return parsedRankingFilter{}, errors.New("invalid learning_group_id format")
		}
		filter.learningGroupID = &id
	}
	if dto.QuizID != "" {
		id, err := uuid.Parse(dto.QuizID)
		if err != nil {
			return parsedRankingFilter{}, errors.New("invalid quiz_id format")
		}
		filter.quizID = &id
	}
	if dto.StartDate != "" {
		value, err := time.Parse("2006-01-02", dto.StartDate)
		if err != nil {
			return parsedRankingFilter{}, errors.New("invalid start_date format; use YYYY-MM-DD")
		}
		filter.startDate = &value
	}
	if dto.EndDate != "" {
		value, err := time.Parse("2006-01-02", dto.EndDate)
		if err != nil {
			return parsedRankingFilter{}, errors.New("invalid end_date format; use YYYY-MM-DD")
		}
		filter.endDate = &value
	}
	if filter.startDate != nil && filter.endDate != nil && filter.startDate.After(*filter.endDate) {
		return parsedRankingFilter{}, errors.New("start_date cannot be later than end_date")
	}
	return filter, nil
}

func rankingAggregateQuery(db *gorm.DB, filter parsedRankingFilter) *gorm.DB {
	query := db.Table("quiz_sessions qs").
		Joins("JOIN quizzes q ON q.id = qs.quiz_id AND q.deleted_at IS NULL").
		Joins("JOIN users u ON u.id = qs.user_id AND u.deleted_at IS NULL").
		Where("qs.deleted_at IS NULL").
		Where("qs.status IN ?", []models.QuizSessionStatus{models.QuizSessionStatusSubmitted, models.QuizSessionStatusTimeout}).
		Where("q.institution_id = ?", filter.institutionID).
		Where("u.institution_id = ?", filter.institutionID).
		Where("u.type = ?", "student")
	if filter.learningGroupID != nil {
		query = query.Where("q.learning_group_id = ?", *filter.learningGroupID)
	}
	if filter.quizID != nil {
		query = query.Where("q.id = ?", *filter.quizID)
	}
	if filter.startDate != nil {
		query = query.Where("qs.start_time >= ?", *filter.startDate)
	}
	if filter.endDate != nil {
		query = query.Where("qs.start_time < ?", filter.endDate.AddDate(0, 0, 1))
	}
	return query
}
