package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LearningResourceType string

const (
	LearningResourceTypeFile  LearningResourceType = "file"
	LearningResourceTypeVideo LearningResourceType = "video"
	LearningResourceTypeLink  LearningResourceType = "link"
)

type LearningResource struct {
	ID             uuid.UUID            `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Title          string               `gorm:"type:varchar(200);not null" json:"title"`
	Description    string               `gorm:"type:text" json:"description,omitempty"`
	Type           LearningResourceType `gorm:"type:varchar(50);not null" json:"type"`
	FileURL        string               `gorm:"type:text" json:"file_url"`
	UploadedUserID uuid.UUID            `gorm:"type:uuid;not null;index" json:"uploaded_user_id"`
	CreatedAt      time.Time            `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time            `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt       `gorm:"index" json:"-"`

	LearningGroupIDs []uuid.UUID     `gorm:"-" json:"learning_group_ids"`
	LearningGroups   []LearningGroup `gorm:"many2many:learning_resource_groups;constraint:OnDelete:CASCADE" json:"learning_groups,omitempty"`
	UploadedUser     *User           `gorm:"foreignKey:UploadedUserID;constraint:OnDelete:CASCADE" json:"uploaded_user,omitempty"`
}

type LearningResourceGroup struct {
	LearningResourceID uuid.UUID `gorm:"type:uuid;primaryKey" json:"learning_resource_id"`
	LearningGroupID    uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"learning_group_id"`

	LearningResource *LearningResource `gorm:"foreignKey:LearningResourceID;constraint:OnDelete:CASCADE" json:"-"`
	LearningGroup    *LearningGroup    `gorm:"foreignKey:LearningGroupID;constraint:OnDelete:CASCADE" json:"-"`
}
