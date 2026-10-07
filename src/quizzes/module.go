package quizzes

import (
	"clasenna-go-backend/libs/cryptography"
	notif "clasenna-go-backend/libs/notifications"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterQuizModule(router *gin.RouterGroup, db *gorm.DB) {
	service := NewService(db, notif.NewPublisherFromEnv())
	controller := NewController(service)
	permissions := cryptography.AccessMiddleware{DB: db}

	group := router.Group("/quizzes")

	group.Use(cryptography.JWTMiddleware(db))
	{
		group.GET("", permissions.Handler("quizzes", "get-all"), controller.GetAll)
		group.GET("/:id", permissions.Handler("quizzes", "get-by-id"), controller.GetByID)
		group.POST("", permissions.Handler("quizzes", "create"), controller.Create)
		group.PUT("/:id", permissions.Handler("quizzes", "update"), controller.Update)
		group.DELETE("/:id/archived", permissions.Handler("quizzes", "archive"), controller.Archive)
		group.DELETE("/:id", permissions.Handler("quizzes", "delete"), controller.Delete)
	}
}
