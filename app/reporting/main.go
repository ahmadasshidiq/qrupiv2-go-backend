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
	"clasenna-go-backend/libs/stores"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	logger := loglib.New("reporting-service")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	db, err := stores.OpenPostgres("REPORTING")
	if err != nil {
		logger.Error("database startup failed", "error", err)
		return
	}
	store := kafkalib.GormIdempotencyStore{DB: db}
	if db.Migrator().HasColumn(&StudentReadModel{}, "tenant_id") && !db.Migrator().HasColumn(&StudentReadModel{}, "institution_id") {
		if err := db.Migrator().RenameColumn(&StudentReadModel{}, "tenant_id", "institution_id"); err != nil {
			logger.Error("read model migration failed", "error", err)
			return
		}
	}
	if err := db.AutoMigrate(&StudentReadModel{}, &kafkalib.ProcessedEvent{}); err != nil {
		logger.Error("migration failed", "error", err)
		return
	}
	brokers := helpers.ConfigStrings("KAFKA_BROKERS", "localhost:9092")
	eventsTopic := helpers.ConfigString("KAFKA_EVENTS_TOPIC", "qrupi.events")
	dlqTopic := helpers.ConfigString("KAFKA_DLQ_TOPIC", "qrupi.events.dlq")
	if err := kafkalib.EnsureTopics(ctx, brokers, []string{eventsTopic, dlqTopic}, helpers.ConfigInt("KAFKA_TOPIC_PARTITIONS", 3), helpers.ConfigInt("KAFKA_REPLICATION_FACTOR", 1)); err != nil {
		logger.Error("Kafka topic setup failed", "error", err)
		return
	}
	consumer := kafkalib.NewConsumer(kafkalib.ConsumerConfig{Brokers: brokers, Topic: eventsTopic, GroupID: "reporting-service", DLQTopic: dlqTopic, MaxRetries: helpers.ConfigInt("KAFKA_MAX_RETRIES", 3), RetryDelay: helpers.ConfigDuration("KAFKA_RETRY_DELAY", time.Second)}, store, logger)
	defer consumer.Close()
	go func() {
		if err := consumer.Run(ctx, HandleEvent(db, logger)); err != nil {
			logger.Error("consumer stopped", "error", err)
			stop()
		}
	}()
	r := gin.New()
	r.Use(gin.Recovery())
	metrics := &httpserver.Metrics{}
	r.Use(httpserver.Middleware(logger, metrics))
	httpserver.RegisterProbes(r, metrics, nil)
	RegisterAllModules(r.Group("/api/v1"), db)
	if err := httpserver.Run(ctx, ":"+helpers.ConfigString("REPORTING_PORT", "3005"), r, logger); err != nil {
		logger.Error("server stopped", "error", err)
	}
}
