package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AnnouncementAudience string

const (
	AnnouncementAudienceSystem        AnnouncementAudience = "system"
	AnnouncementAudienceInstitution   AnnouncementAudience = "institution"
	AnnouncementAudienceLearningGroup AnnouncementAudience = "learning_group"
	AnnouncementAudienceUser          AnnouncementAudience = "user"
)

type Notification struct {
	ID             uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	InstitutionID  *uuid.UUID     `gorm:"type:uuid;index" json:"institution_id,omitempty"`
	UserID         uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	AnnouncementID *uuid.UUID     `gorm:"type:uuid;index" json:"announcement_id,omitempty"`
	EventType      string         `gorm:"type:varchar(100);not null;index" json:"event_type"`
	Title          string         `gorm:"type:varchar(255);not null" json:"title"`
	Message        string         `gorm:"type:text;not null" json:"message"`
	EntityID       *uuid.UUID     `gorm:"type:uuid;index" json:"entity_id,omitempty"`
	Deeplink       string         `gorm:"type:text" json:"deeplink,omitempty"`
	WebURL         string         `gorm:"type:text" json:"web_url,omitempty"`
	MobileRoute    string         `gorm:"type:varchar(255)" json:"mobile_route,omitempty"`
	Data           datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"data,omitempty"`
	ReadAt         *time.Time     `json:"read_at,omitempty"`
	CreatedAt      time.Time      `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

type Announcement struct {
	ID            uuid.UUID            `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	InstitutionID *uuid.UUID           `gorm:"type:uuid;index" json:"institution_id,omitempty"`
	CreatedBy     uuid.UUID            `gorm:"type:uuid;not null;index" json:"created_by"`
	Audience      AnnouncementAudience `gorm:"type:varchar(30);not null;index" json:"audience"`
	TargetID      *uuid.UUID           `gorm:"type:uuid;index" json:"target_id,omitempty"`
	Title         string               `gorm:"type:varchar(255);not null" json:"title"`
	Message       string               `gorm:"type:text;not null" json:"message"`
	Deeplink      string               `gorm:"type:text" json:"deeplink,omitempty"`
	WebURL        string               `gorm:"type:text" json:"web_url,omitempty"`
	MobileRoute   string               `gorm:"type:varchar(255)" json:"mobile_route,omitempty"`
	PublishedAt   *time.Time           `gorm:"index" json:"published_at,omitempty"`
	CreatedAt     time.Time            `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time            `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt       `gorm:"index" json:"-"`
}

type NotificationDevice struct {
	ID            uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID        uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:idx_notification_device_user_token" json:"user_id"`
	InstitutionID *uuid.UUID     `gorm:"type:uuid;index" json:"institution_id,omitempty"`
	Token         string         `gorm:"type:text;not null;uniqueIndex:idx_notification_device_user_token" json:"-"`
	Platform      string         `gorm:"type:varchar(20);not null;default:'web'" json:"platform"`
	IsActive      bool           `gorm:"not null;default:true;index" json:"is_active"`
	LastSeenAt    time.Time      `gorm:"autoUpdateTime" json:"last_seen_at"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}
