package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoleInGroup string

const (
	RoleInGroupInstructor RoleInGroup = "instructor"
	RoleInGroupStudent    RoleInGroup = "student"
)

type LearningGroupMember struct {
	ID              uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	LearningGroupID uuid.UUID      `gorm:"type:uuid;not null;index;uniqueIndex:idx_learning_group_user,where:deleted_at IS NULL" json:"learning_group_id"`
	UserID          uuid.UUID      `gorm:"type:uuid;not null;index;uniqueIndex:idx_learning_group_user,where:deleted_at IS NULL" json:"user_id"`
	RoleInGroup     RoleInGroup    `gorm:"type:varchar(50);not null" json:"role_in_group"`
	CreatedAt       time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	LearningGroup *LearningGroup `gorm:"foreignKey:LearningGroupID;constraint:OnDelete:CASCADE" json:"learning_group,omitempty"`
	User          *User          `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}
