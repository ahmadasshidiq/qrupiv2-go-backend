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

func RegisterAllModules(router *gin.RouterGroup, db *gorm.DB, sender *notif.FCMSender, logger *slog.Logger) {
	RegisterNotificationAPI(router, db)
	router.POST("/internal/events", func(c *gin.Context) {
		var event notif.Event
		if err := c.ShouldBindJSON(&event); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		recipients := event.RecipientIDs
		if len(recipients) == 0 && event.UserID != "" {
			recipients = []string{event.UserID}
		}
		if err := persistNotifications(c, db, sender, logger, string(event.Type), event.InstitutionID, recipients, event.Title, event.Message, event.EntityID, event.Deeplink, event.WebURL, event.MobileRoute, event.Data); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.Status(202)
	})
}

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
		return persistNotifications(ctx, db, sender, logger, event.EventType, event.InstitutionID, payload.RecipientIDs, payload.Title, payload.Message, payload.EntityID, payload.Deeplink, payload.WebURL, payload.MobileRoute, payload.Data)
		/*for _, recipient := range payload.RecipientIDs {
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
		}*/
		logger.InfoContext(ctx, "notification event processed", "event_type", event.EventType, "event_id", event.EventID, "recipients", len(payload.RecipientIDs))
		return nil
	}
}

func persistNotifications(ctx context.Context, db *gorm.DB, sender *notif.FCMSender, logger *slog.Logger, eventType, institution string, recipients []string, title, message, entityID, deeplink, webURL, mobileRoute string, data map[string]any) error {
	institutionID, _ := uuid.Parse(institution)
	eid, _ := uuid.Parse(entityID)
	raw, _ := json.Marshal(data)
	for _, recipient := range recipients {
		userID, err := uuid.Parse(recipient)
		if err != nil {
			continue
		}
		var existing int64
		db.WithContext(ctx).Model(&models.Notification{}).Where("user_id = ? AND event_type = ? AND entity_id = ? AND deleted_at IS NULL", userID, eventType, eid).Count(&existing)
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
		if err := db.WithContext(ctx).Create(&n).Error; err != nil {
			return err
		}
		if sender != nil {
			var devices []models.NotificationDevice
			db.WithContext(ctx).Where("user_id = ? AND is_active = ?", userID, true).Find(&devices)
			for _, d := range devices {
				_, _ = sender.Send(ctx, d.Token, title, message, deeplink, webURL, mobileRoute, entityID)
			}
		}
	}
	return nil
}
