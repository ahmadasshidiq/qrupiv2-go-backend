package attendance_logs

import (
	"clasenna-go-backend/libs/cryptography"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterAttendanceLogModule(router *gin.RouterGroup, db *gorm.DB) {
	controller := NewController(NewService(db))
	permissions := cryptography.AccessMiddleware{DB: db}
	group := router.Group("/attendance-logs")
	group.Use(cryptography.JWTMiddleware(db))
	{
		group.GET("", permissions.Handler("attendance-logs", "get-all"), controller.GetAll)
		group.GET("/:id", permissions.Handler("attendance-logs", "get-by-id"), controller.GetByID)
		group.POST("/check-in", permissions.Handler("attendance-logs", "create"), controller.CheckIn)
		group.POST("", permissions.Handler("attendance-logs", "create"), controller.Create)
		group.PUT("/:id/check-out", permissions.Handler("attendance-logs", "update"), controller.CheckOut)
		group.PUT("/:id", permissions.Handler("attendance-logs", "update"), controller.Update)
		group.DELETE("/:id/archived", permissions.Handler("attendance-logs", "delete"), controller.Archive)
		group.DELETE("/:id", permissions.Handler("attendance-logs", "delete"), controller.Delete)
	}
}
