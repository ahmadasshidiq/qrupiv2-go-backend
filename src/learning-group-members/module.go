package learning_group_members

import (
	"clasenna-go-backend/libs/cryptography"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterLearningGroupMemberModule(router *gin.RouterGroup, db *gorm.DB) {
	service := NewService(db)
	controller := NewController(service)
	permissions := cryptography.AccessMiddleware{DB: db}

	group := router.Group("/learning-group-members")

	group.Use(cryptography.JWTMiddleware(db))
	{
		group.GET("", permissions.Handler("learning-group-members", "get-all"), controller.GetAll)
		group.GET("/:id", permissions.Handler("learning-group-members", "get-by-id"), controller.GetByID)
		group.POST("", permissions.Handler("learning-group-members", "create"), controller.Create)
		group.PUT("/:id", permissions.Handler("learning-group-members", "update"), controller.Update)
		group.DELETE("/:id/archived", permissions.Handler("learning-group-members", "delete"), controller.Archive)
		group.DELETE("/:id", permissions.Handler("learning-group-members", "delete"), controller.Delete)
	}
}
