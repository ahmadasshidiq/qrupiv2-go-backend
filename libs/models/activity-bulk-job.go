package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type ActivityBulkJobStatus string

const (
	ActivityBulkJobPending    ActivityBulkJobStatus = "pending"
	ActivityBulkJobProcessing ActivityBulkJobStatus = "processing"
	ActivityBulkJobCompleted  ActivityBulkJobStatus = "completed"
	ActivityBulkJobFailed     ActivityBulkJobStatus = "failed"
)

type ActivityBulkJob struct {
	ID             uuid.UUID             `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	InstitutionID  *uuid.UUID            `gorm:"type:uuid;index" json:"institution_id,omitempty"`
	RecordedUserID uuid.UUID             `gorm:"type:uuid;not null;index" json:"recorded_user_id"`
	Status         ActivityBulkJobStatus `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	TotalData      int                   `gorm:"not null" json:"total_data"`
	ProcessedData  int                   `gorm:"not null;default:0" json:"processed_data"`
	FailedData     int                   `gorm:"not null;default:0" json:"failed_data"`
	Attempts       int                   `gorm:"not null;default:0" json:"attempts"`
	ErrorMessage   string                `gorm:"type:text" json:"error_message,omitempty"`
	Payload        datatypes.JSON        `gorm:"type:jsonb;not null" json:"-"`
	StartedAt      *time.Time            `json:"started_at,omitempty"`
	CompletedAt    *time.Time            `json:"completed_at,omitempty"`
	CreatedAt      time.Time             `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time             `gorm:"autoUpdateTime" json:"updated_at"`
}

type ActivityOutboxEvent struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	AggregateID uuid.UUID      `gorm:"type:uuid;not null;index"`
	EventType   string         `gorm:"type:varchar(100);not null"`
	Payload     datatypes.JSON `gorm:"type:jsonb;not null"`
	Attempts    int            `gorm:"not null;default:0"`
	LastError   string         `gorm:"type:text"`
	PublishedAt *time.Time     `gorm:"index"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
}
