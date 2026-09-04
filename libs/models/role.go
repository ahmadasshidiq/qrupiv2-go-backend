package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PermissionItem struct {
	Model  string `json:"model"`
	Action string `json:"action"`
}

type GroupedPermission struct {
	Title       string           `json:"title"`
	Permissions []PermissionItem `json:"permissions"`
}

type Role struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name        string         `gorm:"type:varchar(100);unique;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Permissions datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"permissions"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
