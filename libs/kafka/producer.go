package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

type Producer interface {
	Publish(context.Context, string, Envelope) error
	Close() error
}

type Writer struct {
	writer *kafkago.Writer
	logger *slog.Logger
}

func NewProducer(brokers []string, logger *slog.Logger) *Writer {
	return &Writer{writer: &kafkago.Writer{
		Addr: kafkago.TCP(brokers...), Balancer: &kafkago.Hash{},
		RequiredAcks: kafkago.RequireAll, Async: false, WriteTimeout: 10 * time.Second,
	}, logger: logger}
}

func (p *Writer) Publish(ctx context.Context, topic string, event Envelope) error {
	if err := event.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	err = p.writer.WriteMessages(ctx, kafkago.Message{
		Topic: topic, Key: []byte(event.InstitutionID), Value: payload,
		Headers: []kafkago.Header{{Key: "correlation_id", Value: []byte(event.CorrelationID)}},
		Time:    event.Timestamp,
	})
	if err != nil {
		return fmt.Errorf("publish %s: %w", event.EventType, err)
	}
	p.logger.InfoContext(ctx, "kafka event published", "event_id", event.EventID, "event_type", event.EventType, "topic", topic)
	return nil
}

func (p *Writer) Close() error { return p.writer.Close() }
