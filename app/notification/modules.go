package main

import (
	"context"
	"encoding/json"
	"log/slog"

	kafkalib "clasenna-go-backend/libs/kafka"
	"clasenna-go-backend/libs/models"
	notif "clasenna-go-backend/libs/notifications"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func RegisterAllModules(_ *gin.RouterGroup) {}

type notificationEventData struct {
	RecipientIDs []string       `json:"recipient_ids"`
	Title        string         `json:"title"`
	Message      string         `json:"message"`
	EntityID     string         `json:"entity_id"`
	Deeplink     string         `json:"deeplink"`
	WebURL       string         `json:"web_url"`
	MobileRoute  string         `json:"mobile_route"`
	Data         map[string]any `json:"data"`
}

func HandleEvent(db *gorm.DB, sender *notif.FCMSender, logger *slog.Logger) kafkalib.Handler {
	return func(ctx context.Context, event kafkalib.Envelope) error {
		var payload notificationEventData
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return err
		}
		entityID, _ := uuid.Parse(payload.EntityID)
		institutionID, _ := uuid.Parse(event.InstitutionID)
		var rawData datatypes.JSON
		if payload.Data != nil {
			encoded, err := json.Marshal(payload.Data)
			if err != nil {
				return err
			}
			rawData = datatypes.JSON(encoded)
		} else {
			rawData = datatypes.JSON(`{}`)
		}
		for _, recipient := range payload.RecipientIDs {
			userID, err := uuid.Parse(recipient)
			if err != nil {
				continue
			}
			var existing int64
			db.WithContext(ctx).Model(&models.Notification{}).Where("user_id = ? AND event_type = ? AND entity_id = ? AND deleted_at IS NULL", userID, event.EventType, entityID).Count(&existing)
			if existing > 0 {
				continue
			}
			notification := models.Notification{UserID: userID, InstitutionID: &institutionID, EventType: event.EventType, Title: payload.Title, Message: payload.Message, Deeplink: payload.Deeplink, WebURL: payload.WebURL, MobileRoute: payload.MobileRoute, Data: rawData}
			if entityID != uuid.Nil {
				notification.EntityID = &entityID
			}
			if err := db.WithContext(ctx).Create(&notification).Error; err != nil {
				return err
			}
			var devices []models.NotificationDevice
			db.WithContext(ctx).Where("user_id = ? AND is_active = ?", userID, true).Find(&devices)
			for _, device := range devices {
				if _, err := sender.Send(ctx, device.Token, payload.Title, payload.Message, payload.Deeplink, payload.WebURL, payload.MobileRoute, payload.EntityID); err != nil {
					logger.WarnContext(ctx, "firebase notification failed", "error", err, "user_id", userID)
				}
			}
		}
		logger.InfoContext(ctx, "notification event processed", "event_type", event.EventType, "event_id", event.EventID, "recipients", len(payload.RecipientIDs))
		return nil
	}
}
