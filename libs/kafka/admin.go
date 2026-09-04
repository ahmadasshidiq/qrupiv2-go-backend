package kafka

import (
	"context"
	"fmt"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

func EnsureTopics(ctx context.Context, brokers, topics []string, partitions, replicationFactor int) error {
	if len(brokers) == 0 {
		return fmt.Errorf("at least one Kafka broker is required")
	}
	if partitions < 1 {
		partitions = 1
	}
	if replicationFactor < 1 {
		replicationFactor = 1
	}
	setupContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	broker, err := kafkago.DialContext(setupContext, "tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("connect to Kafka broker: %w", err)
	}
	defer broker.Close()

	controller, err := broker.Controller()
	if err != nil {
		return fmt.Errorf("get Kafka controller: %w", err)
	}
	controllerAddress := fmt.Sprintf("%s:%d", controller.Host, controller.Port)
	connection, err := kafkago.DialContext(setupContext, "tcp", controllerAddress)
	if err != nil {
		return fmt.Errorf("connect to Kafka controller: %w", err)
	}
	defer connection.Close()

	configs := make([]kafkago.TopicConfig, 0, len(topics))
	for _, topic := range topics {
		if topic == "" {
			continue
		}
		configs = append(configs, kafkago.TopicConfig{
			Topic:             topic,
			NumPartitions:     partitions,
			ReplicationFactor: replicationFactor,
		})
	}
	if len(configs) == 0 {
		return nil
	}
	if err := connection.CreateTopics(configs...); err != nil {
		return fmt.Errorf("create Kafka topics: %w", err)
	}
	return nil
}
