package main

import (
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/httpserver"
	kafkalib "clasenna-go-backend/libs/kafka"
	loglib "clasenna-go-backend/libs/logger"
	"clasenna-go-backend/libs/models"
	"clasenna-go-backend/libs/stores"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"os/signal"
	"syscall"
)

func main() {
	_ = godotenv.Load()
	logger := loglib.New("auth-service")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	db, err := stores.OpenPostgres("AUTH")
	if err != nil {
		logger.Error("database startup failed", "error", err)
		return
	}
	if err := db.AutoMigrate(&models.Institution{}, &models.Role{}, &models.User{}); err != nil {
		logger.Error("database migration failed", "error", err)
		return
	}
	if err := models.MigrateLegacyStatusColumns(db); err != nil {
		logger.Error("status migration failed", "error", err)
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
	r := gin.New()
	r.Use(gin.Recovery())
	metrics := &httpserver.Metrics{}
	r.Use(httpserver.Middleware(logger, metrics))
	httpserver.RegisterProbes(r, metrics, func(ctx context.Context) error {
		sqlDB, e := db.DB()
		if e != nil {
			return e
		}
		return sqlDB.PingContext(ctx)
	})
	RegisterAllModules(r.Group("/api/v1"), db, producer, eventsTopic)
	if err := httpserver.Run(ctx, ":"+helpers.ConfigString("AUTH_PORT", "3001"), r, logger); err != nil {
		logger.Error("server stopped", "error", err)
	}
}
