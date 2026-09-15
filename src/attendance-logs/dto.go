package attendance_logs

import "time"

const MaxBulkAttendanceLogs = 200

type DefaultFindDTO struct {
	SortBy    string `form:"sortBy" binding:"omitempty"`
	SortOrder string `form:"sortOrder" binding:"omitempty,oneof=asc desc"`
	Limit     int    `form:"limit" binding:"omitempty,min=1"`
	Page      int    `form:"page" binding:"omitempty,min=1"`
}

type CreateDTO struct {
	UserID           string     `json:"user_id" binding:"required,uuid"`
	LearningGroupID  string     `json:"learning_group_id" binding:"omitempty,uuid"`
	AbsenceReasonID  string     `json:"absence_reason_id" binding:"omitempty,uuid"`
	AbsenceNote      string     `json:"absence_note" binding:"omitempty"`
	Type             string     `json:"type" binding:"required,oneof=student teacher learning_group"`
	Status           string     `json:"status" binding:"required,oneof=on_time late absent"`
	RequiresCheckOut bool       `json:"requires_check_out"`
	CheckInAt        *time.Time `json:"check_in_at"`
	CheckInLat       *float64   `json:"check_in_lat" binding:"omitempty,latitude"`
	CheckInLong      *float64   `json:"check_in_long" binding:"omitempty,longitude"`
	CheckOutAt       *time.Time `json:"check_out_at"`
	CheckOutLat      *float64   `json:"check_out_lat" binding:"omitempty,latitude"`
	CheckOutLong     *float64   `json:"check_out_long" binding:"omitempty,longitude"`
}

type BulkCreateDTO struct {
	AttendanceLogs []CreateDTO `json:"attendance_logs" binding:"required,min=1,max=200,dive"`
}

type BulkCreateResult struct {
	Count int      `json:"count"`
	IDs   []string `json:"ids"`
}

type UpdateDTO struct {
	UserID           *string    `json:"user_id" binding:"omitempty,uuid"`
	LearningGroupID  *string    `json:"learning_group_id" binding:"omitempty"`
	AbsenceReasonID  *string    `json:"absence_reason_id" binding:"omitempty"`
	AbsenceNote      *string    `json:"absence_note" binding:"omitempty"`
	Type             *string    `json:"type" binding:"omitempty,oneof=student teacher learning_group"`
	Status           *string    `json:"status" binding:"omitempty,oneof=on_time late absent"`
	RequiresCheckOut *bool      `json:"requires_check_out"`
	CheckInAt        *time.Time `json:"check_in_at"`
	CheckInLat       *float64   `json:"check_in_lat" binding:"omitempty,latitude"`
	CheckInLong      *float64   `json:"check_in_long" binding:"omitempty,longitude"`
	CheckOutAt       *time.Time `json:"check_out_at"`
	CheckOutLat      *float64   `json:"check_out_lat" binding:"omitempty,latitude"`
	CheckOutLong     *float64   `json:"check_out_long" binding:"omitempty,longitude"`
}

type CheckInDTO struct {
	UserID           string     `json:"user_id" binding:"required,uuid"`
	LearningGroupID  string     `json:"learning_group_id" binding:"omitempty,uuid"`
	Type             string     `json:"type" binding:"required,oneof=student teacher learning_group"`
	Status           string     `json:"status" binding:"required,oneof=on_time late"`
	RequiresCheckOut bool       `json:"requires_check_out"`
	CheckInAt        *time.Time `json:"check_in_at"`
	LocationLat      *float64   `json:"location_lat" binding:"omitempty,latitude"`
	LocationLong     *float64   `json:"location_long" binding:"omitempty,longitude"`
}

type CheckOutDTO struct {
	CheckOutAt   *time.Time `json:"check_out_at"`
	LocationLat  *float64   `json:"location_lat" binding:"omitempty,latitude"`
	LocationLong *float64   `json:"location_long" binding:"omitempty,longitude"`
}
