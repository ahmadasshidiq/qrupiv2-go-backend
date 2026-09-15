package main

import (
	kafkalib "clasenna-go-backend/libs/kafka"
	"clasenna-go-backend/src/institutions"
	"clasenna-go-backend/src/roles"
	"clasenna-go-backend/src/users"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterAllModules(router *gin.RouterGroup, db *gorm.DB, producer kafkalib.Producer, topic string) {
	institutions.RegisterInstitutionsModule(router, db)
	users.RegisterUserModule(router, db, users.KafkaEventPublisher{Producer: producer, Topic: topic})
	roles.RegisterRoleModule(router, db)
}
