package learning_resources

import (
	"clasenna-go-backend/libs/cryptography"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterLearningResourceModule(router *gin.RouterGroup, db *gorm.DB) {
	service := NewService(db)
	controller := NewController(service)
	permissions := cryptography.AccessMiddleware{DB: db}

	group := router.Group("/learning-resources")

	group.Use(cryptography.JWTMiddleware(db))
	{
		group.GET("", permissions.Handler("learning-resources", "get-all"), controller.GetAll)
		group.GET("/:id", permissions.Handler("learning-resources", "get-by-id"), controller.GetByID)
		group.POST("", permissions.Handler("learning-resources", "create"), controller.Create)
		group.PUT("/:id", permissions.Handler("learning-resources", "update"), controller.Update)
		group.DELETE("/:id/archived", permissions.Handler("learning-resources", "delete"), controller.Archive)
		group.DELETE("/:id", permissions.Handler("learning-resources", "delete"), controller.Delete)
	}
}
