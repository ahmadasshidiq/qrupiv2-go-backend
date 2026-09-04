package models

import "time"

type Province struct {
	Code            string    `gorm:"type:varchar(20);primaryKey" json:"code"`
	Name            string    `gorm:"type:varchar(255);not null;index" json:"name"`
	IsActive        bool      `gorm:"not null;default:true;index" json:"is_active"`
	SourceUpdatedAt time.Time `gorm:"type:date;not null" json:"source_updated_at"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type Regency struct {
	Code            string    `gorm:"type:varchar(20);primaryKey" json:"code"`
	ProvinceCode    string    `gorm:"type:varchar(20);not null;index" json:"province_code"`
	Name            string    `gorm:"type:varchar(255);not null;index" json:"name"`
	IsActive        bool      `gorm:"not null;default:true;index" json:"is_active"`
	SourceUpdatedAt time.Time `gorm:"type:date;not null" json:"source_updated_at"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Province *Province `gorm:"foreignKey:ProvinceCode;references:Code;constraint:OnDelete:CASCADE" json:"province,omitempty"`
}

type District struct {
	Code            string    `gorm:"type:varchar(20);primaryKey" json:"code"`
	RegencyCode     string    `gorm:"type:varchar(20);not null;index" json:"regency_code"`
	Name            string    `gorm:"type:varchar(255);not null;index" json:"name"`
	IsActive        bool      `gorm:"not null;default:true;index" json:"is_active"`
	SourceUpdatedAt time.Time `gorm:"type:date;not null" json:"source_updated_at"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Regency *Regency `gorm:"foreignKey:RegencyCode;references:Code;constraint:OnDelete:CASCADE" json:"regency,omitempty"`
}

type Village struct {
	Code            string    `gorm:"type:varchar(20);primaryKey" json:"code"`
	DistrictCode    string    `gorm:"type:varchar(20);not null;index" json:"district_code"`
	Name            string    `gorm:"type:varchar(255);not null;index" json:"name"`
	IsActive        bool      `gorm:"not null;default:true;index" json:"is_active"`
	SourceUpdatedAt time.Time `gorm:"type:date;not null" json:"source_updated_at"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	District *District `gorm:"foreignKey:DistrictCode;references:Code;constraint:OnDelete:CASCADE" json:"district,omitempty"`
}

type RegionSyncState struct {
	Source          string     `gorm:"type:varchar(100);primaryKey" json:"source"`
	SourceUpdatedAt *time.Time `gorm:"type:date" json:"source_updated_at,omitempty"`
	LastCheckedAt   *time.Time `json:"last_checked_at,omitempty"`
	LastSyncedAt    *time.Time `json:"last_synced_at,omitempty"`
	Status          string     `gorm:"type:varchar(20);not null" json:"status"`
	ErrorMessage    string     `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt       time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}
