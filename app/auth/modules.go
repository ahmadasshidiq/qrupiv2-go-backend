package main

import (
	kafkalib "clasenna-go-backend/libs/kafka"
	"clasenna-go-backend/src/auth"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterAllModules(router *gin.RouterGroup, db *gorm.DB, producer kafkalib.Producer, topic string) {
	auth.RegisterAuthModule(router, db, auth.KafkaEventPublisher{Producer: producer, Topic: topic})
}
