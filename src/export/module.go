package export

import (
	"clasenna-go-backend/libs/cryptography"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterExportModule(router *gin.RouterGroup, db *gorm.DB) {
	service := NewExportService(db)
	controller := NewExportController(service)

	group := router.Group("/export")
	group.Use(cryptography.JWTMiddleware(db))
	group.POST("/excel", controller.ExportExcel)
}
