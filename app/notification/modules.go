package main

import (
	"context"
	"log/slog"

	kafkalib "clasenna-go-backend/libs/kafka"
	"github.com/gin-gonic/gin"
)

// RegisterAllModules is the notification-service HTTP module registry.
// Channel endpoints will be registered here when required.
func RegisterAllModules(_ *gin.RouterGroup) {}

func HandleEvent(logger *slog.Logger) kafkalib.Handler {
	return func(ctx context.Context, event kafkalib.Envelope) error {
		if event.EventType == "student.created" {
			logger.InfoContext(ctx, "student.created consumed successfully", "event_id", event.EventID, "institution_id", event.InstitutionID, "correlation_id", event.CorrelationID)
		}
		return nil
	}
}
