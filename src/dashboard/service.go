package dashboard

import (
	"gorm.io/gorm"
)

type DashboardService struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *DashboardService {
	return &DashboardService{DB: db}
}
