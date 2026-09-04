package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Activity struct {
	ID              uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ActivityItemID  uuid.UUID      `gorm:"column:item_id;type:uuid;not null;index" json:"activity_item_id"`
	UserID          uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	LearningGroupID *uuid.UUID     `gorm:"type:uuid;index" json:"learning_group_id,omitempty"`
	RecordedUserID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"recorded_user_id"`
	Description     string         `gorm:"type:text" json:"description"`
	PointValue      int            `gorm:"type:int;not null;default:0" json:"point_value"`
	Platform        string         `gorm:"type:varchar(50);not null" json:"platform"`
	OccurredAt      time.Time      `gorm:"not null;index" json:"occurred_at"`
	CreatedAt       time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	ActivityItem  *ActivityItem  `gorm:"foreignKey:ActivityItemID" json:"activity_item,omitempty"`
	User          *User          `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	LearningGroup *LearningGroup `gorm:"foreignKey:LearningGroupID;constraint:OnDelete:SET NULL" json:"learning_group,omitempty"`
	RecordedUser  *User          `gorm:"foreignKey:RecordedUserID;constraint:OnDelete:RESTRICT" json:"recorded_user,omitempty"`
}
