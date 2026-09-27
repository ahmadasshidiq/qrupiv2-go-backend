package models

import "time"

import "github.com/google/uuid"

type LegalDocument struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Slug          string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"slug"`
	Title         string    `gorm:"type:varchar(255);not null" json:"title"`
	Content       string    `gorm:"type:text;not null" json:"content"`
	Version       string    `gorm:"type:varchar(50);not null" json:"version"`
	Status        string    `gorm:"type:varchar(20);not null;index" json:"status"`
	EffectiveDate time.Time `gorm:"not null" json:"effective_date"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
