package auth

import (
	"clasenna-go-backend/libs/cryptography"
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
		group.POST("/student/switch", cryptography.JWTMiddleware(db), authController.SwitchStudent)
		group.POST("/logout", cryptography.JWTMiddleware(db), authController.Logout)
		group.POST("/reset-password", authController.ResetPassword)
	}
}
