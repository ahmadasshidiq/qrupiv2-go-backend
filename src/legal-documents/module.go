package legal_documents

import (
	"clasenna-go-backend/libs/cryptography"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterLegalDocumentModule(r *gin.RouterGroup, db *gorm.DB) {
	c := NewController(NewService(db))
	p := cryptography.AccessMiddleware{DB: db}
	public := r.Group("/legal-documents")
	public.GET("", c.GetAll)
	public.GET("/:id", c.GetByID)

	protected := r.Group("/legal-documents")
	protected.Use(cryptography.JWTMiddleware(db))
	protected.POST("", p.Handler("legal-documents", "create"), c.Create)
	protected.PATCH("/:id", p.Handler("legal-documents", "update"), c.Update)
	protected.PUT("/:id", p.Handler("legal-documents", "update"), c.Update)
	protected.DELETE("/:id", p.Handler("legal-documents", "delete"), c.Delete)
}
