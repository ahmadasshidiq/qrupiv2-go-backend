package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InstitutionStatus string

const (
	InstitutionStatusActive   InstitutionStatus = "active"
	InstitutionStatusInactive InstitutionStatus = "inactive"
)

type Institution struct {
	ID                    uuid.UUID         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CurrentSubscriptionID *uuid.UUID        `gorm:"type:uuid" json:"current_subscription_id,omitempty"`
	Name                  string            `gorm:"type:varchar(255);not null" json:"name"`
	Code                  string            `gorm:"type:varchar(100);unique;not null" json:"code"`
	AvatarURL             string            `gorm:"type:varchar(255)" json:"avatar_url"`
	Address               string            `gorm:"type:varchar(255)" json:"address"`
	Latitude              float64           `gorm:"type:decimal(10,8)" json:"latitude"`
	Longitude             float64           `gorm:"type:decimal(11,8)" json:"longitude"`
	Phone                 string            `gorm:"type:varchar(20)" json:"phone"`
	Website               string            `gorm:"type:varchar(255)" json:"website"`
	ProvinceCode          string            `gorm:"type:varchar(20);index" json:"province_code"`
	ProvinceName          string            `gorm:"type:varchar(255)" json:"province_name"`
	RegencyCode           string            `gorm:"type:varchar(20);index" json:"regency_code"`
	RegencyName           string            `gorm:"type:varchar(255)" json:"regency_name"`
	DistrictCode          string            `gorm:"type:varchar(20);index" json:"district_code"`
	DistrictName          string            `gorm:"type:varchar(255)" json:"district_name"`
	VillageCode           string            `gorm:"type:varchar(20);index" json:"village_code"`
	VillageName           string            `gorm:"type:varchar(255)" json:"village_name"`
	Country               string            `gorm:"type:varchar(100)" json:"country"`
	ZipCode               string            `gorm:"type:varchar(20)" json:"zip_code"`
	Status                InstitutionStatus `gorm:"type:varchar(20);default:'active'" json:"status"`
	CreatedAt             time.Time         `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time         `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt             gorm.DeletedAt    `gorm:"index" json:"-"`

	// Relations
	Users []User `gorm:"foreignKey:InstitutionID" json:"users,omitempty"`
}
