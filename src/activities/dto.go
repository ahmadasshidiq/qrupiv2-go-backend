package activities

import (
	"time"

	"clasenna-go-backend/libs/models"
)

type DefaultFindDTO struct {
	SortBy    string `form:"sortBy" binding:"omitempty"`
	SortOrder string `form:"sortOrder" binding:"omitempty,oneof=asc desc"`
	Limit     int    `form:"limit" binding:"omitempty,min=1"`
	Page      int    `form:"page" binding:"omitempty,min=1"`
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
	Count int      `json:"count"`
	IDs   []string `json:"ids"`
}

type BulkJobResult struct {
	JobID                      string                       `json:"job_id"`
	Status                     models.ActivityBulkJobStatus `json:"status"`
	TotalData                  int                          `json:"total_data"`
	Message                    string                       `json:"message"`
	EstimatedCompletionMinutes int                          `json:"estimated_completion_minutes"`
	RetryAfterSeconds          int                          `json:"retry_after_seconds"`
}
