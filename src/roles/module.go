package roles

import (
	"clasenna-go-backend/libs/cryptography"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoleModule(router *gin.RouterGroup, db *gorm.DB) {
	service := NewService(db)
	controller := NewController(service)
	permissions := cryptography.AccessMiddleware{DB: db}

	group := router.Group("/roles")

	// with out security
	group.GET("/master-permissions", controller.GetMasterPermissions)

	group.Use(cryptography.JWTMiddleware(db))
	{
		group.GET("", permissions.Handler("roles", "get-all"), controller.GetAll)
		group.GET("/:id", permissions.Handler("roles", "get-by-id"), controller.GetByID)
		group.POST("", permissions.Handler("roles", "create"), controller.Create)
		group.PUT("/:id", permissions.Handler("roles", "update"), controller.Update)
		group.DELETE("/:id/archived", permissions.Handler("roles", "archive"), controller.Archive)
		group.DELETE("/:id", permissions.Handler("roles", "delete"), controller.Delete)
	}
}
