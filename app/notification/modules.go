package main

import (
	notif "clasenna-go-backend/libs/notifications"
	feature "clasenna-go-backend/src/notifications"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"log/slog"
)

func RegisterAllModules(router *gin.RouterGroup, db *gorm.DB, sender *notif.FCMSender, logger *slog.Logger) *feature.Service {
	return feature.RegisterModule(router, db, sender, logger)
}
