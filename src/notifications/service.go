package notifications

import (
	"clasenna-go-backend/libs/kafka"
	"clasenna-go-backend/libs/models"
	notif "clasenna-go-backend/libs/notifications"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"log/slog"
)

type Service struct {
	DB     *gorm.DB
	Sender *notif.FCMSender
	Logger *slog.Logger
}

func NewService(db *gorm.DB, sender *notif.FCMSender, logger *slog.Logger) *Service {
	return &Service{db, sender, logger}
}

func (s *Service) Persist(ctx context.Context, eventType, institution string, recipients []string, title, message, entityID, deeplink, webURL, mobileRoute string, data map[string]any) error {
	institutionID, _ := uuid.Parse(institution)
	eid, _ := uuid.Parse(entityID)
	raw, _ := json.Marshal(data)
	for _, recipient := range recipients {
		userID, err := uuid.Parse(recipient)
		if err != nil {
			continue
		}
		var existing int64
		s.DB.WithContext(ctx).Model(&models.Notification{}).Where("user_id = ? AND event_type = ? AND entity_id = ? AND deleted_at IS NULL", userID, eventType, eid).Count(&existing)
		if existing > 0 {
			continue
		}
		n := models.Notification{UserID: userID, EventType: eventType, Title: title, Message: message, Deeplink: deeplink, WebURL: webURL, MobileRoute: mobileRoute, Data: datatypes.JSON(raw)}
		if institutionID != uuid.Nil {
			n.InstitutionID = &institutionID
		}
		if eid != uuid.Nil {
			n.EntityID = &eid
		}
		if err := s.DB.WithContext(ctx).Create(&n).Error; err != nil {
			return err
		}
		if s.Sender != nil {
			var devices []models.NotificationDevice
			s.DB.WithContext(ctx).Where("user_id = ? AND is_active = ?", userID, true).Find(&devices)
			for _, d := range devices {
				_, _ = s.Sender.Send(ctx, d.Token, title, message, deeplink, webURL, mobileRoute, entityID)
			}
		}
	}
	return nil
}

func (s *Service) HandleEvent(ctx context.Context, event kafka.Envelope) error {
	var payload EventDataDTO
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return err
	}
	if err := s.Persist(ctx, event.EventType, event.InstitutionID, payload.RecipientIDs, payload.Title, payload.Message, payload.EntityID, payload.Deeplink, payload.WebURL, payload.MobileRoute, payload.Data); err != nil {
		return err
	}
	if s.Logger != nil {
		s.Logger.InfoContext(ctx, "notification event processed", "event_type", event.EventType, "event_id", event.EventID, "recipients", len(payload.RecipientIDs))
	}
	return nil
}
