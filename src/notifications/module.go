package notifications

import (
	"clasenna-go-backend/libs/notifications"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"log/slog"
)

func RegisterModule(router *gin.RouterGroup, db *gorm.DB, sender *notifications.FCMSender, logger *slog.Logger) *Service {
	s := NewService(db, sender, logger)
	NewController(s).Register(router)
	router.POST("/internal/events", func(c *gin.Context) {
		var event notifications.Event
		if err := c.ShouldBindJSON(&event); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		recipients := event.RecipientIDs
		if len(recipients) == 0 && event.UserID != "" {
			recipients = []string{event.UserID}
		}
		if err := s.Persist(c, string(event.Type), event.InstitutionID, recipients, event.Title, event.Message, event.EntityID, event.Deeplink, event.WebURL, event.MobileRoute, event.Data); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.Status(202)
	})
	return s
}
