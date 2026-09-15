package activityitems

import (
	"clasenna-go-backend/libs/cryptography"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterActivityItemModule(router *gin.RouterGroup, db *gorm.DB) {
	c := NewController(NewService(db))
	p := cryptography.AccessMiddleware{DB: db}
	g := router.Group("/activity-items")
	g.Use(cryptography.JWTMiddleware(db))
	g.GET("", p.Handler("activity-items", "get-all"), c.GetAll)
	g.GET("/:id", p.Handler("activity-items", "get-by-id"), c.GetByID)
	g.POST("", p.Handler("activity-items", "create"), c.Create)
	g.PUT("/:id", p.Handler("activity-items", "update"), c.Update)
	g.DELETE("/:id/archived", p.Handler("activity-items", "archive"), c.Archive)
	g.DELETE("/:id", p.Handler("activity-items", "delete"), c.Delete)
}
