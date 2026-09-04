package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LearningGroupType string

const (
	LearningGroupTypeSchoolClass     LearningGroupType = "school-class"
	LearningGroupTypeUniversityClass LearningGroupType = "university-class"
	LearningGroupTypeClub            LearningGroupType = "club"
)

type LearningGroup struct {
	ID            uuid.UUID         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	InstitutionID uuid.UUID         `gorm:"type:uuid;not null;index" json:"institution_id"`
	Name          string            `gorm:"type:varchar(100);not null" json:"name"`
	Code          string            `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"`
	Type          LearningGroupType `gorm:"type:varchar(50);not null" json:"type"`
	Level         int               `gorm:"not null" json:"level"`
	Major         string            `gorm:"type:varchar(100)" json:"major,omitempty"`
	Department    string            `gorm:"type:varchar(100)" json:"department,omitempty"`
	AcademicYear  string            `gorm:"type:varchar(20)" json:"academic_year,omitempty"`
	IsActive      bool              `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time         `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time         `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt    `gorm:"index" json:"-"`

	Institution *Institution `gorm:"foreignKey:InstitutionID;constraint:OnDelete:CASCADE" json:"institution,omitempty"`
}
