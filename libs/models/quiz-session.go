package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type QuizSessionStatus string

const (
	QuizSessionStatusInProgress QuizSessionStatus = "in_progress"
	QuizSessionStatusSubmitted  QuizSessionStatus = "submitted"
	QuizSessionStatusTimeout    QuizSessionStatus = "timeout"
)

type CheatingLog struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Penalty   int       `json:"penalty"`
	Note      string    `json:"note"`
}

type Answer struct {
	QuestionID     string `json:"question_id"`
	SelectedAnswer string `json:"selected_answer"`
	IsCorrect      bool   `json:"is_correct"`
	PointsEarned   int    `json:"points_earned"`
}

type QuizSession struct {
	ID            uuid.UUID         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	QuizID        uuid.UUID         `gorm:"type:uuid;not null;index;uniqueIndex:idx_quiz_session_user,where:deleted_at IS NULL" json:"quiz_id"`
	UserID        uuid.UUID         `gorm:"type:uuid;not null;index;uniqueIndex:idx_quiz_session_user,where:deleted_at IS NULL" json:"user_id"`
	StartTime     time.Time         `gorm:"not null" json:"start_time"`
	EndTime       time.Time         `json:"end_time"`
	CheatingCount int               `gorm:"default:0" json:"cheating_count"`
	CheatingLogs  datatypes.JSON    `gorm:"type:jsonb;not null;default:'[]'" json:"cheating_logs"`
	Status        QuizSessionStatus `gorm:"type:varchar(20);not null;default:'in_progress'" json:"status"`
	Answers       datatypes.JSON    `gorm:"type:jsonb;not null;default:'[]'" json:"answers"`
	Score         float64           `gorm:"default:0" json:"score"`
	DeviceInfo    string            `gorm:"type:text" json:"device_info"`
	CreatedAt     time.Time         `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time         `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt    `gorm:"index" json:"-"`

	Quiz *Quiz `gorm:"foreignKey:QuizID;constraint:OnDelete:CASCADE" json:"quiz,omitempty"`
	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}
