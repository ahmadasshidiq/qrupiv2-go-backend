package attendance_absence_reasons

import (
	"clasenna-go-backend/libs/cryptography"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterAttendanceAbsenceReasonModule(router *gin.RouterGroup, db *gorm.DB) {
	controller := NewController(NewService(db))
	permissions := cryptography.AccessMiddleware{DB: db}
	group := router.Group("/attendance-absence-reasons")
	group.Use(cryptography.JWTMiddleware(db))
	group.GET("", permissions.Handler("attendance-absence-reasons", "get-all"), controller.GetAll)
	group.GET("/:id", permissions.Handler("attendance-absence-reasons", "get-by-id"), controller.GetByID)
	group.POST("", permissions.Handler("attendance-absence-reasons", "create"), controller.Create)
	group.PUT("/:id", permissions.Handler("attendance-absence-reasons", "update"), controller.Update)
	group.DELETE("/:id/archived", permissions.Handler("attendance-absence-reasons", "delete"), controller.Archive)
	group.DELETE("/:id", permissions.Handler("attendance-absence-reasons", "delete"), controller.Delete)
}
