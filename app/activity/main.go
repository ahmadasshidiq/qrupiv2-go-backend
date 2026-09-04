package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/httpserver"
	kafkalib "clasenna-go-backend/libs/kafka"
	loglib "clasenna-go-backend/libs/logger"
	"clasenna-go-backend/libs/models"
	"clasenna-go-backend/libs/stores"
	"clasenna-go-backend/src/activities"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	logger := loglib.New("activity-service")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := stores.OpenPostgres("ACTIVITY")
	if err != nil {
		logger.Error("database startup failed", "error", err)
		return
	}
	if err := db.AutoMigrate(
		&models.ActivityCategory{},
		&models.ActivityItem{},
		&models.Activity{},
		&models.ActivityBulkJob{},
		&models.ActivityOutboxEvent{},
	); err != nil {
		logger.Error("database migration failed", "error", err)
		return
	}
	brokers := helpers.ConfigStrings("KAFKA_BROKERS", "localhost:9092")
	eventsTopic := helpers.ConfigString("KAFKA_EVENTS_TOPIC", "qrupi.events")
	dlqTopic := helpers.ConfigString("KAFKA_DLQ_TOPIC", "qrupi.events.dlq")
	if err := kafkalib.EnsureTopics(ctx, brokers, []string{eventsTopic, dlqTopic}, helpers.ConfigInt("KAFKA_TOPIC_PARTITIONS", 3), helpers.ConfigInt("KAFKA_REPLICATION_FACTOR", 1)); err != nil {
		logger.Error("Kafka topic setup failed", "error", err)
		return
	}
	producer := kafkalib.NewProducer(brokers, logger)
	defer producer.Close()
	maxRetries := helpers.ConfigInt("KAFKA_MAX_RETRIES", 3)
	consumer := kafkalib.NewConsumer(kafkalib.ConsumerConfig{
		Brokers: brokers, Topic: eventsTopic, GroupID: "activity-service",
		DLQTopic: dlqTopic, MaxRetries: maxRetries,
		RetryDelay: helpers.ConfigDuration("KAFKA_RETRY_DELAY", time.Second),
	}, nil, logger)
	defer consumer.Close()
	go activities.StartActivityOutboxDispatcher(ctx, db, producer, eventsTopic, logger)
	go func() {
		if err := consumer.Run(ctx, activities.HandleActivityBulkEvent(db, maxRetries)); err != nil {
			logger.Error("activity consumer stopped", "error", err)
			stop()
		}
	}()

	router := gin.New()
	router.Use(gin.Recovery())
	metrics := &httpserver.Metrics{}
	router.Use(httpserver.Middleware(logger, metrics))
	httpserver.RegisterProbes(router, metrics, func(ctx context.Context) error {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.PingContext(ctx)
	})
	RegisterAllModules(router.Group("/api/v1"), db)
	if err := httpserver.Run(ctx, ":"+helpers.ConfigString("ACTIVITY_PORT", "3006"), router, logger); err != nil {
		logger.Error("service stopped", "error", err)
	}
}
