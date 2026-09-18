package quizzes

import "time"

type DefaultFindDTO struct {
	SortBy    string `form:"sortBy" json:"sortBy" binding:"omitempty"`
	SortOrder string `form:"sortOrder" json:"sortOrder" binding:"omitempty,oneof=asc desc"`
	Limit     int    `form:"limit" json:"limit" binding:"omitempty,min=1"`
	Page      int    `form:"page" json:"page" binding:"omitempty,min=1"`
}

type QuizOption struct {
	Label string `json:"label" binding:"required"`
	Value string `json:"value" binding:"required"`
}

type QuizQuestion struct {
	QuestionText  string       `json:"question_text" binding:"required"`
	Options       []QuizOption `json:"options" binding:"required,dive"`
	CorrectAnswer string       `json:"correct_answer" binding:"required"`
	Type          string       `json:"type" binding:"required,oneof=multiple_choice essay true_false fill_blank"`
	Points        int          `json:"points" binding:"required,gte=1"`
}

type CreateDTO struct {
	InstitutionID          string         `json:"institution_id" binding:"required,uuid"`
	LearningGroupID        string         `json:"learning_group_id" binding:"required,uuid"`
	CreatedUserID          string         `json:"created_user_id" binding:"required,uuid"`
	Title                  string         `json:"title" binding:"required"`
	Description            string         `json:"description" binding:"required"`
	QuizQuestions          []QuizQuestion `json:"quiz_questions" binding:"required,dive"`
	StartTime              time.Time      `json:"start_time" binding:"required"`
	EndTime                time.Time      `json:"end_time" binding:"required,gtfield=StartTime"`
	DurationMinutes        int            `json:"duration_minutes" binding:"required,gte=1"`
	MaxCheatingWarnings    int            `json:"max_cheating_warnings" binding:"omitempty,gte=0"`
	CheatingPenaltyMinutes int            `json:"cheating_penalty_minutes" binding:"omitempty,gte=0"`
	Type                   string         `json:"type" binding:"required,oneof=quiz midterm final"`
}

type UpdateDTO struct {
	Title                  *string         `json:"title" binding:"omitempty"`
	Description            *string         `json:"description" binding:"omitempty"`
	QuizQuestions          *[]QuizQuestion `json:"quiz_questions" binding:"omitempty,dive"`
	StartTime              *time.Time      `json:"start_time" binding:"omitempty"`
	EndTime                *time.Time      `json:"end_time" binding:"omitempty"`
	DurationMinutes        *int            `json:"duration_minutes" binding:"omitempty,gte=1"`
	MaxCheatingWarnings    *int            `json:"max_cheating_warnings" binding:"omitempty,gte=0"`
	CheatingPenaltyMinutes *int            `json:"cheating_penalty_minutes" binding:"omitempty,gte=0"`
	Type                   *string         `json:"type" binding:"omitempty,oneof=quiz midterm final"`
}
