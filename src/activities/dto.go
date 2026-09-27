package activities

import (
	"github.com/google/uuid"
	"time"

	"clasenna-go-backend/libs/models"
)

type DefaultFindDTO struct {
	SortBy    string `form:"sortBy" binding:"omitempty"`
	SortOrder string `form:"sortOrder" binding:"omitempty,oneof=asc desc"`
	Limit     int    `form:"limit" binding:"omitempty,min=1"`
	Page      int    `form:"page" binding:"omitempty,min=1"`
}

type ActivityLimitStatusDTO struct {
	Date string `form:"date" binding:"required,datetime=2006-01-02"`
}

type ActivityLimitStatus struct {
	ActivityItemID uuid.UUID `json:"activity_item_id"`
	Name           string    `json:"name"`
	DailyLimit     int       `json:"daily_limit"`
	DailyUsed      int64     `json:"daily_used"`
	PeriodLimit    int       `json:"period_limit"`
	PeriodUsed     int64     `json:"period_used"`
	PeriodType     string    `json:"period_type"`
	Locked         bool      `json:"locked"`
	Reason         string    `json:"reason,omitempty"`
}

type BulkActivityLimitStatus struct {
	UserID   string                `json:"user_id"`
	UserName string                `json:"user_name"`
	Items    []ActivityLimitStatus `json:"items"`
}

type ChartFilterDTO struct {
	CategoryID      string `form:"category_id" binding:"omitempty,uuid"`
	Type            string `form:"type" binding:"omitempty,oneof=positive violation"`
	StartDate       string `form:"start_date" binding:"omitempty,datetime=2006-01-02"`
	EndDate         string `form:"end_date" binding:"omitempty,datetime=2006-01-02"`
	LearningGroupID string `form:"learning_group_id" binding:"omitempty,uuid"`
	TopLimit        int    `form:"top_limit" binding:"omitempty,min=1,max=100"`
}
type CreateDTO struct {
	ActivityItemID  string    `json:"activity_item_id" binding:"required,uuid"`
	UserID          string    `json:"user_id" binding:"required,uuid"`
	LearningGroupID string    `json:"learning_group_id" binding:"omitempty,uuid"`
	RecordedUserID  string    `json:"recorded_user_id" binding:"required,uuid"`
	Description     string    `json:"description" binding:"omitempty"`
	PointValue      *int      `json:"point_value" binding:"omitempty,min=0"`
	Platform        string    `json:"platform" binding:"required,max=50"`
	OccurredAt      time.Time `json:"occurred_at" binding:"required"`
}
type UpdateDTO struct {
	ActivityItemID  *string    `json:"activity_item_id" binding:"omitempty,uuid"`
	UserID          *string    `json:"user_id" binding:"omitempty,uuid"`
	LearningGroupID *string    `json:"learning_group_id" binding:"omitempty,uuid"`
	RecordedUserID  *string    `json:"recorded_user_id" binding:"omitempty,uuid"`
	Description     *string    `json:"description" binding:"omitempty"`
	PointValue      *int       `json:"point_value" binding:"omitempty,min=0"`
	Platform        *string    `json:"platform" binding:"omitempty,max=50"`
	OccurredAt      *time.Time `json:"occurred_at" binding:"omitempty"`
}

const MaxBulkActivities = 10000
const SynchronousBulkThreshold = 200

type BulkCreateDTO struct {
	LearningGroupID string              `json:"learning_group_id" binding:"omitempty,uuid"`
	RecordedUserID  string              `json:"recorded_user_id" binding:"required,uuid"`
	Platform        string              `json:"platform" binding:"required,max=50"`
	Activities      []BulkActivityEntry `json:"activities" binding:"required,min=1,max=10000,dive"`
}

type BulkActivityEntry struct {
	ActivityItemID string    `json:"activity_item_id" binding:"required,uuid"`
	UserID         string    `json:"user_id" binding:"required,uuid"`
	Description    string    `json:"description" binding:"omitempty"`
	PointValue     *int      `json:"point_value" binding:"omitempty,min=0"`
	OccurredAt     time.Time `json:"occurred_at" binding:"required"`
}

type BulkCreateResult struct {
	Count   int                   `json:"count"`
	IDs     []string              `json:"ids"`
	Skipped []BulkValidationError `json:"skipped,omitempty"`
}

type BulkJobResult struct {
	JobID                      string                       `json:"job_id"`
	Status                     models.ActivityBulkJobStatus `json:"status"`
	TotalData                  int                          `json:"total_data"`
	Message                    string                       `json:"message"`
	EstimatedCompletionMinutes int                          `json:"estimated_completion_minutes"`
	RetryAfterSeconds          int                          `json:"retry_after_seconds"`
}
