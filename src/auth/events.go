package auth

import (
	kafkalib "clasenna-go-backend/libs/kafka"
	"context"
)

type EventPublisher interface {
	Publish(context.Context, string, string, string, any) error
}
type KafkaEventPublisher struct {
	Producer kafkalib.Producer
	Topic    string
}

func (p KafkaEventPublisher) Publish(ctx context.Context, eventType, institutionID, correlationID string, data any) error {
	if institutionID == "" {
		institutionID = "system"
	}
	event, err := kafkalib.NewEnvelope(eventType, "auth-service", institutionID, correlationID, data)
	if err != nil {
		return err
	}
	return p.Producer.Publish(ctx, p.Topic, event)
}
