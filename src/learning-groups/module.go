package learning_groups

import (
	"clasenna-go-backend/libs/cryptography"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterLearningGroupModule(router *gin.RouterGroup, db *gorm.DB) {
	service := NewService(db)
	controller := NewController(service)
	permissions := cryptography.AccessMiddleware{DB: db}

	group := router.Group("/learning-groups")

	group.Use(cryptography.JWTMiddleware(db))
	{
		group.GET("", permissions.Handler("learning-groups", "get-all"), controller.GetAll)
		group.GET("/:id", permissions.Handler("learning-groups", "get-by-id"), controller.GetByID)
		group.POST("", permissions.Handler("learning-groups", "create"), controller.Create)
		group.PUT("/:id", permissions.Handler("learning-groups", "update"), controller.Update)
		group.DELETE("/:id/archived", permissions.Handler("learning-groups", "delete"), controller.Archive)
		group.DELETE("/:id", permissions.Handler("learning-groups", "delete"), controller.Delete)
	}
}
