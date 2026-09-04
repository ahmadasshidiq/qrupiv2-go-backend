package quiz_sessions

import "time"

type DefaultFindDTO struct {
	SortBy    string `form:"sortBy" json:"sortBy" binding:"omitempty"`
	SortOrder string `form:"sortOrder" json:"sortOrder" binding:"omitempty,oneof=asc desc"`
	Limit     int    `form:"limit" json:"limit" binding:"omitempty,min=1"`
	Page      int    `form:"page" json:"page" binding:"omitempty,min=1"`
}

type CheatingLog struct {
	Timestamp time.Time `json:"timestamp" binding:"required"`
	Type      string    `json:"type" binding:"required"`    // e.g. "tab-switch", "blur", "focus-lost"
	Penalty   int       `json:"penalty" binding:"required"` // minutes deducted
	Note      string    `json:"note" binding:"omitempty"`
}

type Answer struct {
	QuestionID     string `json:"question_id" binding:"required"`
	SelectedAnswer string `json:"selected_answer" binding:"required"`
	IsCorrect      *bool  `json:"is_correct" binding:"required"`
	PointsEarned   *int   `json:"points_earned" binding:"required,gte=0"`
}

type CreateDTO struct {
	QuizID     string    `json:"quiz_id" binding:"required,uuid"`
	UserID     string    `json:"user_id" binding:"required,uuid"`
	DeviceInfo string    `json:"device_info" binding:"omitempty"`
	StartTime  time.Time `json:"start_time" binding:"required"`
}

type UpdateDTO struct {
	EndTime       *time.Time     `json:"end_time" binding:"omitempty"`
	Status        *string        `json:"status" binding:"omitempty,oneof=in_progress submitted timeout"`
	CheatingCount *int           `json:"cheating_count" binding:"omitempty,gte=0"`
	CheatingLogs  *[]CheatingLog `json:"cheating_logs" binding:"omitempty,dive"`
	Answers       *[]Answer      `json:"answers" binding:"omitempty,dive"`
	Score         *float64       `json:"score" binding:"omitempty,gte=0"`
	DeviceInfo    *string        `json:"device_info" binding:"omitempty"`
}
