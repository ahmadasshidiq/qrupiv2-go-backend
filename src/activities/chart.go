package activities

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActivityChartSummary struct {
	TotalActivities     int64 `json:"total_activities"`
	PositiveActivities  int64 `json:"positive_activities"`
	ViolationActivities int64 `json:"violation_activities"`
	TotalPoints         int64 `json:"total_points"`
}

type ActivitiesByItem struct {
	ActivityItemID   uuid.UUID `json:"activity_item_id"`
	ActivityItemName string    `json:"activity_item_name"`
	TotalActivities  int64     `json:"total_activities"`
	Color            string    `json:"color"`
}

type ActivitiesByLearningGroup struct {
	LearningGroupID   uuid.UUID `json:"learning_group_id"`
	LearningGroupName string    `json:"learning_group_name"`
	TotalActivities   int64     `json:"total_activities"`
}

type TopStudent struct {
	UserID          uuid.UUID `json:"user_id"`
	UserName        string    `json:"user_name"`
	TotalActivities int64     `json:"total_activities"`
	TotalPoints     int64     `json:"total_points"`
}

type DailyActivityTrend struct {
	Date                string `json:"date"`
	PositiveActivities  int64  `json:"positive_activities"`
	ViolationActivities int64  `json:"violation_activities"`
}

type ActivityChartResponse struct {
	Summary                   ActivityChartSummary        `json:"summary"`
	ActivitiesByItem          []ActivitiesByItem          `json:"activities_by_item"`
	ActivitiesByLearningGroup []ActivitiesByLearningGroup `json:"activities_by_learning_group"`
	TopStudents               []TopStudent                `json:"top_students"`
	DailyTrend                []DailyActivityTrend        `json:"daily_trend"`
}

type parsedChartFilter struct {
	institutionID   uuid.UUID
	categoryID      *uuid.UUID
	learningGroupID *uuid.UUID
	activityType    string
	startDate       *time.Time
	endDate         *time.Time
	topLimit        int
}

func (s *ActivityService) getChart(ctx *gin.Context, dto ChartFilterDTO) (*ActivityChartResponse, error) {
	filter, err := parseChartFilter(ctx, dto)
	if err != nil {
		return nil, err
	}

	result := &ActivityChartResponse{
		ActivitiesByItem:          make([]ActivitiesByItem, 0),
		ActivitiesByLearningGroup: make([]ActivitiesByLearningGroup, 0),
		TopStudents:               make([]TopStudent, 0),
		DailyTrend:                make([]DailyActivityTrend, 0),
	}
	dbCtx := ctx.Request.Context()

	if err := activityChartQuery(s.DB.WithContext(dbCtx), filter).
		Select(`COUNT(*) AS total_activities,
			COUNT(*) FILTER (WHERE ai.type = 'positive') AS positive_activities,
			COUNT(*) FILTER (WHERE ai.type = 'violation') AS violation_activities,
			COALESCE(SUM(a.point_value), 0) AS total_points`).
		Scan(&result.Summary).Error; err != nil {
		return nil, err
	}

	if err := activityChartQuery(s.DB.WithContext(dbCtx), filter).
		Select(`ai.id AS activity_item_id, ai.name AS activity_item_name,
			COUNT(*) AS total_activities,
			COALESCE(NULLIF(ai.color, ''), NULLIF(ac.color, ''),
				CASE WHEN ai.type = 'violation' THEN '#EF4444' ELSE '#22C55E' END) AS color`).
		Group("ai.id, ai.name, ai.color, ac.color, ai.type").
		Order("total_activities DESC, ai.name ASC").
		Scan(&result.ActivitiesByItem).Error; err != nil {
		return nil, err
	}

	if err := activityChartQuery(s.DB.WithContext(dbCtx), filter).
		Select("lg.id AS learning_group_id, lg.name AS learning_group_name, COUNT(*) AS total_activities").
		Where("lg.id IS NOT NULL").
		Group("lg.id, lg.name").
		Order("total_activities DESC, lg.name ASC").
		Scan(&result.ActivitiesByLearningGroup).Error; err != nil {
		return nil, err
	}

	if err := activityChartQuery(s.DB.WithContext(dbCtx), filter).
		Select(`u.id AS user_id, u.name AS user_name, COUNT(*) AS total_activities,
			COALESCE(SUM(a.point_value), 0) AS total_points`).
		Where("u.type = ?", "student").
		Group("u.id, u.name").
		Order("total_points DESC, total_activities DESC, u.name ASC").
		Limit(filter.topLimit).
		Scan(&result.TopStudents).Error; err != nil {
		return nil, err
	}

	if err := activityChartQuery(s.DB.WithContext(dbCtx), filter).
		Select(`TO_CHAR(a.occurred_at::date, 'YYYY-MM-DD') AS date,
			COUNT(*) FILTER (WHERE ai.type = 'positive') AS positive_activities,
			COUNT(*) FILTER (WHERE ai.type = 'violation') AS violation_activities`).
		Group("a.occurred_at::date").
		Order("a.occurred_at::date ASC").
		Scan(&result.DailyTrend).Error; err != nil {
		return nil, err
	}

	return result, nil
}

func parseChartFilter(ctx *gin.Context, dto ChartFilterDTO) (parsedChartFilter, error) {
	institutionID, err := uuid.Parse(ctx.GetString("institution_id"))
	if err != nil {
		return parsedChartFilter{}, errors.New("institution_id is required in authenticated session")
	}
	filter := parsedChartFilter{institutionID: institutionID, activityType: dto.Type, topLimit: dto.TopLimit}
	if filter.topLimit == 0 {
		filter.topLimit = 10
	}
	if dto.CategoryID != "" {
		id, err := uuid.Parse(dto.CategoryID)
		if err != nil {
			return parsedChartFilter{}, errors.New("invalid category_id format")
		}
		filter.categoryID = &id
	}
	if dto.LearningGroupID != "" {
		id, err := uuid.Parse(dto.LearningGroupID)
		if err != nil {
			return parsedChartFilter{}, errors.New("invalid learning_group_id format")
		}
		filter.learningGroupID = &id
	}
	if dto.StartDate != "" {
		value, err := time.Parse("2006-01-02", dto.StartDate)
		if err != nil {
			return parsedChartFilter{}, errors.New("invalid start_date format; use YYYY-MM-DD")
		}
		filter.startDate = &value
	}
	if dto.EndDate != "" {
		value, err := time.Parse("2006-01-02", dto.EndDate)
		if err != nil {
			return parsedChartFilter{}, errors.New("invalid end_date format; use YYYY-MM-DD")
		}
		filter.endDate = &value
	}
	if filter.startDate != nil && filter.endDate != nil && filter.startDate.After(*filter.endDate) {
		return parsedChartFilter{}, errors.New("start_date cannot be later than end_date")
	}
	return filter, nil
}

func activityChartQuery(db *gorm.DB, filter parsedChartFilter) *gorm.DB {
	query := db.Table("activities a").
		Joins("JOIN activity_items ai ON ai.id = a.item_id AND ai.deleted_at IS NULL").
		Joins("LEFT JOIN activity_categories ac ON ac.id = ai.category_id AND ac.deleted_at IS NULL").
		Joins("JOIN users u ON u.id = a.user_id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN learning_groups lg ON lg.id = a.learning_group_id AND lg.deleted_at IS NULL").
		Where("a.deleted_at IS NULL").
		Where("u.institution_id = ?", filter.institutionID)
	if filter.categoryID != nil {
		query = query.Where("ai.category_id = ?", *filter.categoryID)
	}
	if filter.activityType != "" {
		query = query.Where("ai.type = ?", filter.activityType)
	}
	if filter.learningGroupID != nil {
		query = query.Where("a.learning_group_id = ?", *filter.learningGroupID)
	}
	if filter.startDate != nil {
		query = query.Where("a.occurred_at >= ?", *filter.startDate)
	}
	if filter.endDate != nil {
		query = query.Where("a.occurred_at < ?", filter.endDate.AddDate(0, 0, 1))
	}
	return query
}
