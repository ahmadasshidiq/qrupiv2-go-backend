package users

import (
	"clasenna-go-backend/libs/cryptography"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterUserModule(router *gin.RouterGroup, db *gorm.DB, publishers ...EventPublisher) {
	service := NewService(db, publishers...)
	controller := NewController(service)
	permissions := cryptography.AccessMiddleware{DB: db}

	group := router.Group("/users")

	// with out security
	group.GET("/template-excel", controller.DownloadTemplateExcel)

	group.Use(cryptography.JWTMiddleware(db))
	{
		group.GET("", permissions.Handler("users", "get-all"), controller.GetAll)
		group.GET("/:id", permissions.Handler("users", "get-by-id"), controller.GetByID)
		group.GET("/:id/barcode", permissions.Handler("users", "get-by-id"), controller.GetQRCode)
		group.POST("", permissions.Handler("users", "create"), controller.Create)
		group.POST("/import-excel", permissions.Handler("users", "import"), controller.ImportByExcel)
		group.PUT("/:id", permissions.Handler("users", "update"), controller.Update)
		group.DELETE("/:id/archived", permissions.Handler("users", "archive"), controller.Archive)
		group.DELETE("/:id", permissions.Handler("users", "delete"), controller.Delete)
	}
}
