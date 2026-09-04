package dashboard

import (
	"clasenna-go-backend/libs/cryptography"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterDashboardModule(router *gin.RouterGroup, db *gorm.DB) {
	group := router.Group("/dashboard")
	group.Use(cryptography.JWTMiddleware(db))
}
