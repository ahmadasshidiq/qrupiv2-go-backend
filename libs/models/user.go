package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
)

type User struct {
	ID            uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	InstitutionID *uuid.UUID     `gorm:"type:uuid;index" json:"institution_id"`
	RoleID        uuid.UUID      `gorm:"type:uuid;not null" json:"role_id"`
	Name          string         `gorm:"type:varchar(255);not null" json:"name"`
	Email         string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Type          string         `gorm:"type:varchar(50);not null" json:"type"`
	Phone         string         `gorm:"type:varchar(20)" json:"phone"`
	ContextType   string         `gorm:"type:varchar(50)" json:"context_type"`
	ContextCode   string         `gorm:"type:varchar(100)" json:"context_code"`
	Status        UserStatus     `gorm:"type:varchar(20);default:'active'" json:"status"`
	Password      string         `gorm:"type:text;not null" json:"-"`
	PinHash       string         `gorm:"type:text" json:"-"`
	Barcode       string         `gorm:"type:varchar(255);uniqueIndex:idx_users_barcode,where:barcode <> ''" json:"barcode,omitempty"`
	Salt          string         `gorm:"type:varchar(255);not null" json:"-"`
	LayerOne      string         `gorm:"type:text" json:"-"`
	LayerTwo      string         `gorm:"type:text" json:"-"`
	CurrentToken  string         `gorm:"type:text" json:"-"`
	AvatarURL     string         `gorm:"type:text" json:"avatar_url"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Institution *Institution `gorm:"foreignKey:InstitutionID" json:"institution,omitempty"`
	Role        *Role        `gorm:"foreignKey:RoleID" json:"role,omitempty"`
}
