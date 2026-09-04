package quiz_sessions

import (
	"clasenna-go-backend/libs/cryptography"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterQuizSessionModule(router *gin.RouterGroup, db *gorm.DB) {
	service := NewService(db)
	controller := NewController(service)
	permissions := cryptography.AccessMiddleware{DB: db}

	group := router.Group("/quiz-sessions")

	group.Use(cryptography.JWTMiddleware(db))
	{
		group.GET("", permissions.Handler("quiz-sessions", "get-all"), controller.GetAll)
		group.GET("/:id", permissions.Handler("quiz-sessions", "get-by-id"), controller.GetByID)
		group.POST("", permissions.Handler("quiz-sessions", "create"), controller.Create)
		group.PUT("/:id", permissions.Handler("quiz-sessions", "update"), controller.Update)
		group.DELETE("/:id/archived", permissions.Handler("quiz-sessions", "delete"), controller.Archive)
		group.DELETE("/:id", permissions.Handler("quiz-sessions", "delete"), controller.Delete)
	}
}
