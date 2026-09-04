package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AttendanceStatus string
type AttendanceType string

const (
	AttendanceStatusOnTime AttendanceStatus = "on_time"
	AttendanceStatusLate   AttendanceStatus = "late"
	AttendanceStatusAbsent AttendanceStatus = "absent"
)

const (
	AttendanceTypeStudent       AttendanceType = "student"
	AttendanceTypeTeacher       AttendanceType = "teacher"
	AttendanceTypeLearningGroup AttendanceType = "learning_group"
)

type AttendanceLog struct {
	ID               uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID           uuid.UUID        `gorm:"type:uuid;not null;index;index:idx_attendance_context_checkin,priority:1" json:"user_id"`
	LearningGroupID  *uuid.UUID       `gorm:"type:uuid;index;index:idx_attendance_context_checkin,priority:3" json:"learning_group_id,omitempty"`
	AbsenceReasonID  *uuid.UUID       `gorm:"type:uuid;index" json:"absence_reason_id,omitempty"`
	AbsenceNote      string           `gorm:"type:text" json:"absence_note"`
	Type             AttendanceType   `gorm:"type:varchar(30);not null;default:'student';index;index:idx_attendance_context_checkin,priority:2" json:"type"`
	Status           AttendanceStatus `gorm:"type:varchar(20);not null;default:'absent'" json:"status"`
	RequiresCheckOut bool             `gorm:"type:boolean;not null;default:false" json:"requires_check_out"`
	CheckInAt        *time.Time       `gorm:"index;index:idx_attendance_context_checkin,priority:4" json:"check_in_at,omitempty"`
	CheckInLat       *float64         `gorm:"type:decimal(10,8)" json:"check_in_lat,omitempty"`
	CheckInLong      *float64         `gorm:"type:decimal(11,8)" json:"check_in_long,omitempty"`
	CheckOutAt       *time.Time       `gorm:"index" json:"check_out_at,omitempty"`
	CheckOutLat      *float64         `gorm:"type:decimal(10,8)" json:"check_out_lat,omitempty"`
	CheckOutLong     *float64         `gorm:"type:decimal(11,8)" json:"check_out_long,omitempty"`
	CreatedAt        time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time        `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt        gorm.DeletedAt   `gorm:"index" json:"-"`

	User          *User                    `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	LearningGroup *LearningGroup           `gorm:"foreignKey:LearningGroupID;constraint:OnDelete:SET NULL" json:"learning_group,omitempty"`
	AbsenceReason *AttendanceAbsenceReason `gorm:"foreignKey:AbsenceReasonID;constraint:OnDelete:SET NULL" json:"absence_reason,omitempty"`
}
