package activities

import (
	"clasenna-go-backend/libs/cryptography"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterActivityModule(router *gin.RouterGroup, db *gorm.DB) {
	c := NewController(NewService(db))
	p := cryptography.AccessMiddleware{DB: db}
	g := router.Group("/activities")
	g.Use(cryptography.JWTMiddleware(db))
	g.GET("", p.Handler("activities", "get-all"), c.GetAll)
	g.GET("/chart", p.Handler("activities", "get-all"), c.GetChart)
	g.GET("/:id", p.Handler("activities", "get-by-id"), c.GetByID)
	g.POST("/bulk", p.Handler("activities", "create"), c.CreateBulk)
	g.POST("", p.Handler("activities", "create"), c.Create)
	g.PUT("/:id", p.Handler("activities", "update"), c.Update)
	g.DELETE("/:id/archived", p.Handler("activities", "archive"), c.Archive)
	g.DELETE("/:id", p.Handler("activities", "delete"), c.Delete)
	jobs := router.Group("/activity-bulk-jobs")
	jobs.Use(cryptography.JWTMiddleware(db))
	jobs.GET("/:id", p.Handler("activities", "get-by-id"), c.GetBulkJob)
}
