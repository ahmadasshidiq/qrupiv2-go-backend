package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type QuizType string

const (
	QuizTypeHomework QuizType = "homework"
	QuizTypeQuiz     QuizType = "quiz"
	QuizTypeMidterm  QuizType = "midterm"
	QuizTypeFinal    QuizType = "final"
)

type QuizOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type QuizQuestion struct {
	QuestionText  string       `json:"question_text"`
	Options       []QuizOption `json:"options"`
	CorrectAnswer string       `json:"correct_answer"`
	Type          string       `json:"type"`
	Points        int          `json:"points"`
}

type QuizQuestions []QuizQuestion

func (q *QuizQuestions) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan quiz questions")
	}
	return json.Unmarshal(bytes, q)
}

func (q QuizQuestions) Value() (driver.Value, error) {
	return json.Marshal(q)
}

type Quiz struct {
	ID                     uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	InstitutionID          uuid.UUID      `gorm:"type:uuid;not null;index" json:"institution_id"`
	LearningGroupID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"learning_group_id"`
	CreatedUserID          uuid.UUID      `gorm:"type:uuid;not null;index" json:"created_user_id"`
	Title                  string         `gorm:"type:varchar(255);not null" json:"title"`
	Description            string         `gorm:"type:text" json:"description"`
	QuizQuestions          QuizQuestions  `gorm:"type:jsonb;not null;default:'[]'" json:"quiz_questions"`
	StartTime              time.Time      `gorm:"not null" json:"start_time"`
	EndTime                time.Time      `gorm:"not null" json:"end_time"`
	DurationMinutes        int            `gorm:"not null" json:"duration_minutes"`
	MaxCheatingWarnings    int            `gorm:"default:3" json:"max_cheating_warnings"`
	CheatingPenaltyMinutes int            `gorm:"default:0" json:"cheating_penalty_minutes"`
	Type                   QuizType       `gorm:"type:varchar(20);not null" json:"type"`
	CreatedAt              time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt              time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt              gorm.DeletedAt `gorm:"index" json:"-"`

	Institution   *Institution   `gorm:"foreignKey:InstitutionID;constraint:OnDelete:CASCADE" json:"institution,omitempty"`
	LearningGroup *LearningGroup `gorm:"foreignKey:LearningGroupID;constraint:OnDelete:CASCADE" json:"learning_group,omitempty"`
	CreatedUser   *User          `gorm:"foreignKey:CreatedUserID;constraint:OnDelete:RESTRICT" json:"created_user,omitempty"`
}
