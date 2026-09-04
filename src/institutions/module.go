package institutions

import (
	"clasenna-go-backend/libs/cryptography"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterInstitutionsModule(router *gin.RouterGroup, db *gorm.DB) {
	service := NewService(db)
	controller := NewController(service)
	permissions := cryptography.AccessMiddleware{DB: db}

	group := router.Group("/institutions")
	group.Use(cryptography.JWTMiddleware(db))
	{
		group.GET("", permissions.Handler("institutions", "get-all"), controller.GetAll)
		group.GET("/:id", permissions.Handler("institutions", "get-by-id"), controller.GetByID)
		group.POST("", permissions.Handler("institutions", "create"), controller.Create)
		group.PUT("/:id", permissions.Handler("institutions", "update"), controller.Update)
		group.DELETE("/:id/archived", permissions.Handler("institutions", "delete"), controller.Archive)
		group.DELETE("/:id", permissions.Handler("institutions", "delete"), controller.Delete)
	}
}
