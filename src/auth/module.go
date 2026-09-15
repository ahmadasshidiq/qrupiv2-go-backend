package auth

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterAuthModule(router *gin.RouterGroup, db *gorm.DB, events ...EventPublisher) {
	authService := NewAuthService(db, events...)
	authController := NewAuthController(authService)

	group := router.Group("/auth")
	{
		group.POST("/register", authController.Register)
		group.POST("/login", authController.Login)
		group.POST("/student/scan", authController.StudentScanLogin)
		group.POST("/student/verify-pin", authController.StudentVerifyPin)
		group.POST("/reset-password", authController.ResetPassword)
	}
}
