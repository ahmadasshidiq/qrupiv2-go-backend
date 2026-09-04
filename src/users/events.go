package users

import (
	"context"

	kafkalib "clasenna-go-backend/libs/kafka"
	"clasenna-go-backend/libs/models"
)

type EventPublisher interface {
	UserCreated(context.Context, models.User, bool, string) error
}

type KafkaEventPublisher struct {
	Producer kafkalib.Producer
	Topic    string
}

func (p KafkaEventPublisher) UserCreated(ctx context.Context, user models.User, student bool, correlationID string) error {
	eventType := "user.created"
	if student {
		eventType = "student.created"
	}
	institutionID := "system"
	if user.InstitutionID != nil {
		institutionID = user.InstitutionID.String()
	}
	event, err := kafkalib.NewEnvelope(eventType, "core-service", institutionID, correlationID, map[string]any{
		"user_id": user.ID, "name": user.Name, "email": user.Email, "role_id": user.RoleID,
	})
	if err != nil {
		return err
	}
	return p.Producer.Publish(ctx, p.Topic, event)
}
