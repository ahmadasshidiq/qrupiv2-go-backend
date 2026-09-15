package activitycategories

import (
	"clasenna-go-backend/libs/cryptography"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterActivityCategoryModule(router *gin.RouterGroup, db *gorm.DB) {
	c := NewController(NewService(db))
	p := cryptography.AccessMiddleware{DB: db}
	g := router.Group("/activity-categories")
	g.Use(cryptography.JWTMiddleware(db))
	g.GET("", p.Handler("activity-categories", "get-all"), c.GetAll)
	g.GET("/:id", p.Handler("activity-categories", "get-by-id"), c.GetByID)
	g.POST("", p.Handler("activity-categories", "create"), c.Create)
	g.PUT("/:id", p.Handler("activity-categories", "update"), c.Update)
	g.DELETE("/:id/archived", p.Handler("activity-categories", "archive"), c.Archive)
	g.DELETE("/:id", p.Handler("activity-categories", "delete"), c.Delete)
}
