package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

type Handler func(context.Context, Envelope) error

type IdempotencyStore interface {
	AlreadyProcessed(context.Context, string) (bool, error)
	MarkProcessed(context.Context, string) error
}

type ConsumerConfig struct {
	Brokers    []string
	Topic      string
	GroupID    string
	DLQTopic   string
	MaxRetries int
	RetryDelay time.Duration
}

type Consumer struct {
	reader *kafkago.Reader
	dlq    *Writer
	cfg    ConsumerConfig
	store  IdempotencyStore
	logger *slog.Logger
}

func NewConsumer(cfg ConsumerConfig, store IdempotencyStore, logger *slog.Logger) *Consumer {
	if cfg.MaxRetries < 1 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = time.Second
	}
	return &Consumer{
		reader: kafkago.NewReader(kafkago.ReaderConfig{Brokers: cfg.Brokers, Topic: cfg.Topic, GroupID: cfg.GroupID, MinBytes: 1, MaxBytes: 10e6, CommitInterval: 0}),
		dlq:    NewProducer(cfg.Brokers, logger), cfg: cfg, store: store, logger: logger,
	}
}

func (c *Consumer) Run(ctx context.Context, handler Handler) error {
	for {
		message, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("fetch kafka message: %w", err)
		}
		if err := c.process(ctx, message, handler); err != nil {
			return err
		}
		if err := c.reader.CommitMessages(ctx, message); err != nil {
			return fmt.Errorf("commit kafka message: %w", err)
		}
	}
}

func (c *Consumer) process(ctx context.Context, message kafkago.Message, handler Handler) error {
	var event Envelope
	if err := json.Unmarshal(message.Value, &event); err != nil {
		return c.deadLetter(ctx, message, Envelope{}, fmt.Errorf("decode envelope: %w", err))
	}
	if err := event.Validate(); err != nil {
		return c.deadLetter(ctx, message, event, err)
	}
	if c.store != nil {
		done, err := c.store.AlreadyProcessed(ctx, event.EventID)
		if err != nil {
			return err
		}
		if done {
			c.logger.InfoContext(ctx, "duplicate kafka event skipped", "event_id", event.EventID)
			return nil
		}
	}
	var err error
	for attempt := 1; attempt <= c.cfg.MaxRetries; attempt++ {
		if err = handler(ctx, event); err == nil {
			break
		}
		c.logger.WarnContext(ctx, "kafka handler failed", "event_id", event.EventID, "attempt", attempt, "error", err)
		if attempt < c.cfg.MaxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.cfg.RetryDelay * time.Duration(attempt)):
			}
		}
	}
	if err != nil {
		return c.deadLetter(ctx, message, event, err)
	}
	if c.store != nil {
		return c.store.MarkProcessed(ctx, event.EventID)
	}
	return nil
}

func (c *Consumer) deadLetter(ctx context.Context, message kafkago.Message, event Envelope, cause error) error {
	if c.cfg.DLQTopic == "" {
		return fmt.Errorf("message failed and no DLQ configured: %w", cause)
	}
	event, err := buildDeadLetterEnvelope(event, c.cfg.GroupID, message.Value, cause)
	if err != nil {
		return err
	}
	if err := c.dlq.Publish(ctx, c.cfg.DLQTopic, event); err != nil {
		return fmt.Errorf("publish DLQ: %w", err)
	}
	c.logger.ErrorContext(ctx, "kafka event sent to DLQ", "event_id", event.EventID, "error", cause)
	return nil
}

func buildDeadLetterEnvelope(event Envelope, source string, payload []byte, cause error) (Envelope, error) {
	if event.Validate() == nil {
		return event, nil
	}
	correlationID := event.CorrelationID
	if correlationID == "" {
		correlationID = "unknown"
	}
	invalidEvent, err := NewEnvelope("kafka.message.invalid", source, "unknown", correlationID, map[string]any{
		"error": cause.Error(), "payload": string(payload), "original_event_id": event.EventID,
	})
	if err != nil {
		return Envelope{}, fmt.Errorf("build DLQ envelope: %w", err)
	}
	return invalidEvent, nil
}

func (c *Consumer) Close() error {
	if err := c.reader.Close(); err != nil {
		return err
	}
	return c.dlq.Close()
}
