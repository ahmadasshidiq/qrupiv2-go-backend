package kafka

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type ProcessedEvent struct {
	EventID     string    `gorm:"primaryKey;size:36"`
	ProcessedAt time.Time `gorm:"not null"`
}

type GormIdempotencyStore struct{ DB *gorm.DB }

func (s GormIdempotencyStore) Migrate() error { return s.DB.AutoMigrate(&ProcessedEvent{}) }
func (s GormIdempotencyStore) AlreadyProcessed(ctx context.Context, id string) (bool, error) {
	var count int64
	err := s.DB.WithContext(ctx).Model(&ProcessedEvent{}).Where("event_id = ?", id).Count(&count).Error
	return count > 0, err
}
func (s GormIdempotencyStore) MarkProcessed(ctx context.Context, id string) error {
	return s.DB.WithContext(ctx).Create(&ProcessedEvent{EventID: id, ProcessedAt: time.Now().UTC()}).Error
}
