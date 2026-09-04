package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AttendanceAbsenceReason struct {
	ID            uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	InstitutionID uuid.UUID      `gorm:"type:uuid;not null;index;uniqueIndex:idx_absence_reason_institution_name" json:"institution_id"`
	Name          string         `gorm:"type:varchar(255);not null;uniqueIndex:idx_absence_reason_institution_name" json:"name"`
	Description   string         `gorm:"type:text" json:"description"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	Institution *Institution `gorm:"foreignKey:InstitutionID;constraint:OnDelete:CASCADE" json:"institution,omitempty"`
}
