package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActivityLimitType string

type ActivityItemType string

const (
	ActivityLimitNone     ActivityLimitType = "none"
	ActivityLimitDaily    ActivityLimitType = "daily"
	ActivityLimitWeekly   ActivityLimitType = "weekly"
	ActivityLimitMonthly  ActivityLimitType = "monthly"
	ActivityLimitLifetime ActivityLimitType = "lifetime"
)

const (
	ActivityItemTypePositive  ActivityItemType = "positive"
	ActivityItemTypeViolation ActivityItemType = "violation"
)

type ActivityItem struct {
	ID            uuid.UUID         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	InstitutionID *uuid.UUID        `gorm:"type:uuid;index" json:"institution_id,omitempty"`
	CategoryID    *uuid.UUID        `gorm:"type:uuid;index" json:"category_id"`
	Name          string            `gorm:"type:varchar(255);not null" json:"name"`
	Description   string            `gorm:"type:text" json:"description"`
	Type          ActivityItemType  `gorm:"type:varchar(50);not null;default:'positive'" json:"type"`
	PointValue    int               `gorm:"type:int;not null;default:0" json:"point_value"`
	DailyLimit    int               `gorm:"type:int;not null;default:0" json:"daily_limit"`
	PeriodLimit   int               `gorm:"type:int;not null;default:0" json:"period_limit"`
	PeriodType    ActivityLimitType `gorm:"type:varchar(50);not null;default:'none'" json:"period_type"`
	IsSendNotif   bool              `gorm:"type:boolean;default:false" json:"is_send_notif"`
	Color         string            `gorm:"type:varchar(20)" json:"color"`
	CreatedAt     time.Time         `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time         `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt    `gorm:"index" json:"-"`

	Category    *ActivityCategory `gorm:"foreignKey:CategoryID;constraint:OnDelete:SET NULL" json:"category,omitempty"`
	Institution *Institution      `gorm:"foreignKey:InstitutionID;constraint:OnDelete:CASCADE" json:"institution,omitempty"`
	Activities  []Activity        `gorm:"foreignKey:ActivityItemID" json:"activities,omitempty"`
}
