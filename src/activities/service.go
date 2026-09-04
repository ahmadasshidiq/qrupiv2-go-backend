package activities

import (
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/models"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ActivityService struct{ DB *gorm.DB }

func NewService(db *gorm.DB) *ActivityService { return &ActivityService{DB: db} }

func (s *ActivityService) getAll(ctx *gin.Context, dto DefaultFindDTO) (*helpers.PaginatedResult, error) {
	params := make(map[string]interface{})
	for key, values := range ctx.Request.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	params["a.deleted_at.isnull"] = ""
	base := `select a.id, a.item_id as activity_item_id, a.user_id, a.learning_group_id,
		a.recorded_user_id, a.description, a.point_value, a.platform, a.occurred_at,
		a.created_at, a.updated_at, ai.name as activity_item_name,
		ac.id as category_id, ac.name as category_name, u.name as user_name,
		ru.name as recorded_user_name, lg.name as learning_group_name
		from activities a
		join activity_items ai on ai.id = a.item_id and ai.deleted_at is null
		join activity_categories ac on ac.id = ai.category_id and ac.deleted_at is null
		join users u on u.id = a.user_id
		join users ru on ru.id = a.recorded_user_id
		left join learning_groups lg on lg.id = a.learning_group_id`
	return helpers.BuildPaginatedQuery(ctx, s.DB, params, "activities", base, "", "", dto.SortBy)
}

func (s *ActivityService) getByID(id string) (*models.Activity, error) {
	var data models.Activity
	err := s.DB.Preload("ActivityItem.Category").Preload("User").Preload("LearningGroup").Preload("RecordedUser").First(&data, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &data, err
}

func (s *ActivityService) create(dto CreateDTO) (*models.Activity, error) {
	itemID, err := uuid.Parse(dto.ActivityItemID)
	if err != nil {
		return nil, errors.New("invalid activity_item_id format")
	}
	userID, err := uuid.Parse(dto.UserID)
	if err != nil {
		return nil, errors.New("invalid user_id format")
	}
	recordedUserID, err := uuid.Parse(dto.RecordedUserID)
	if err != nil {
		return nil, errors.New("invalid recorded_user_id format")
	}
	_, err = s.findItem(itemID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureUser(userID); err != nil {
		return nil, err
	}
	if err := s.ensureUser(recordedUserID); err != nil {
		return nil, err
	}
	learningGroupID, err := s.parseLearningGroupID(dto.LearningGroupID)
	if err != nil {
		return nil, err
	}
	data := models.Activity{ActivityItemID: itemID, UserID: userID, LearningGroupID: learningGroupID, RecordedUserID: recordedUserID, Description: dto.Description, Platform: dto.Platform, OccurredAt: dto.OccurredAt}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		var lockedItem models.ActivityItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedItem, "id = ?", itemID).Error; err != nil {
			return err
		}
		data.PointValue = lockedItem.PointValue
		if dto.PointValue != nil {
			data.PointValue = *dto.PointValue
		}
		if err := checkActivityLimit(tx, lockedItem, userID, dto.OccurredAt, uuid.Nil); err != nil {
			return err
		}
		return tx.Create(&data).Error
	}); err != nil {
		return nil, err
	}
	return &data, nil
}

func (s *ActivityService) update(id string, dto UpdateDTO) (*models.Activity, error) {
	var data models.Activity
	if err := s.DB.First(&data, "id = ?", id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	if dto.ActivityItemID != nil {
		itemID, err := uuid.Parse(*dto.ActivityItemID)
		if err != nil {
			return nil, errors.New("invalid activity_item_id format")
		}
		_, err = s.findItem(itemID)
		if err != nil {
			return nil, err
		}
		data.ActivityItemID = itemID
	}
	if dto.UserID != nil {
		userID, err := uuid.Parse(*dto.UserID)
		if err != nil {
			return nil, errors.New("invalid user_id format")
		}
		if err := s.ensureUser(userID); err != nil {
			return nil, err
		}
		data.UserID = userID
	}
	if dto.RecordedUserID != nil {
		userID, err := uuid.Parse(*dto.RecordedUserID)
		if err != nil {
			return nil, errors.New("invalid recorded_user_id format")
		}
		if err := s.ensureUser(userID); err != nil {
			return nil, err
		}
		data.RecordedUserID = userID
	}
	if dto.LearningGroupID != nil {
		learningGroupID, err := s.parseLearningGroupID(*dto.LearningGroupID)
		if err != nil {
			return nil, err
		}
		data.LearningGroupID = learningGroupID
	}
	if dto.Description != nil {
		data.Description = *dto.Description
	}
	if dto.PointValue != nil {
		data.PointValue = *dto.PointValue
	}
	if dto.Platform != nil {
		data.Platform = *dto.Platform
	}
	if dto.OccurredAt != nil {
		data.OccurredAt = *dto.OccurredAt
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		var lockedItem models.ActivityItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedItem, "id = ?", data.ActivityItemID).Error; err != nil {
			return err
		}
		if err := checkActivityLimit(tx, lockedItem, data.UserID, data.OccurredAt, data.ID); err != nil {
			return err
		}
		return tx.Save(&data).Error
	}); err != nil {
		return nil, err
	}
	return &data, nil
}

func (s *ActivityService) findItem(id uuid.UUID) (*models.ActivityItem, error) {
	var data models.ActivityItem
	if err := s.DB.First(&data, "id = ?", id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("activity item not found")
	} else if err != nil {
		return nil, err
	}
	return &data, nil
}
func (s *ActivityService) ensureUser(id uuid.UUID) error {
	var count int64
	if err := s.DB.Model(&models.User{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("user not found")
	}
	return nil
}
func (s *ActivityService) parseLearningGroupID(value string) (*uuid.UUID, error) {
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, errors.New("invalid learning_group_id format")
	}
	var count int64
	if err := s.DB.Model(&models.LearningGroup{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errors.New("learning group not found")
	}
	return &id, nil
}

func checkActivityLimit(db *gorm.DB, item models.ActivityItem, userID uuid.UUID, occurredAt time.Time, excludeID uuid.UUID) error {
	query := db.Model(&models.Activity{}).Where("user_id = ? AND item_id = ?", userID, item.ID)
	if excludeID != uuid.Nil {
		query = query.Where("id <> ?", excludeID)
	}

	if item.DailyLimit > 0 {
		dayStart := startOfDay(occurredAt)
		var dailyCount int64
		if err := query.Where("occurred_at >= ? AND occurred_at < ?", dayStart, dayStart.AddDate(0, 0, 1)).Count(&dailyCount).Error; err != nil {
			return err
		}
		if dailyCount >= int64(item.DailyLimit) {
			return fmt.Errorf("activity item daily limit reached: maximum %d per day", item.DailyLimit)
		}
	}

	if item.PeriodLimit <= 0 || item.PeriodType == "" || item.PeriodType == models.ActivityLimitNone {
		return nil
	}
	start, end, hasPeriod, err := activityLimitPeriod(item.PeriodType, occurredAt)
	if err != nil {
		return err
	}
	periodQuery := query
	if hasPeriod {
		periodQuery = periodQuery.Where("occurred_at >= ? AND occurred_at < ?", start, end)
	}
	var count int64
	if err := periodQuery.Count(&count).Error; err != nil {
		return err
	}
	if count >= int64(item.PeriodLimit) {
		return fmt.Errorf("activity item period limit reached: maximum %d per %s", item.PeriodLimit, item.PeriodType)
	}
	return nil
}

func startOfDay(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func activityLimitPeriod(limitType models.ActivityLimitType, occurredAt time.Time) (time.Time, time.Time, bool, error) {
	location := occurredAt.Location()
	year, month, _ := occurredAt.Date()
	dayStart := startOfDay(occurredAt)
	switch limitType {
	case models.ActivityLimitDaily:
		return dayStart, dayStart.AddDate(0, 0, 1), true, nil
	case models.ActivityLimitWeekly:
		daysSinceMonday := (int(dayStart.Weekday()) + 6) % 7
		start := dayStart.AddDate(0, 0, -daysSinceMonday)
		return start, start.AddDate(0, 0, 7), true, nil
	case models.ActivityLimitMonthly:
		start := time.Date(year, month, 1, 0, 0, 0, 0, location)
		return start, start.AddDate(0, 1, 0), true, nil
	case models.ActivityLimitLifetime:
		return time.Time{}, time.Time{}, false, nil
	default:
		return time.Time{}, time.Time{}, false, fmt.Errorf("unsupported activity limit type: %s", limitType)
	}
}
func (s *ActivityService) archive(id string) (bool, error) {
	result := s.DB.Where("id = ?", id).Delete(&models.Activity{})
	return result.RowsAffected > 0, result.Error
}
func (s *ActivityService) delete(id string) (bool, error) {
	result := s.DB.Unscoped().Where("id = ?", id).Delete(&models.Activity{})
	return result.RowsAffected > 0, result.Error
}
