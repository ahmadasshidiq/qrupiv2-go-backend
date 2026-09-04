package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActivityCategory struct {
	ID            uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	InstitutionID *uuid.UUID     `gorm:"type:uuid;index" json:"institution_id,omitempty"`
	Name          string         `gorm:"type:varchar(255);not null" json:"name"`
	Description   string         `gorm:"type:text" json:"description"`
	Color         string         `gorm:"type:varchar(20)" json:"color"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	Items       []ActivityItem `gorm:"foreignKey:CategoryID" json:"items,omitempty"`
	Institution *Institution   `gorm:"foreignKey:InstitutionID;constraint:OnDelete:CASCADE" json:"institution,omitempty"`
}
