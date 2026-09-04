package regions

import (
	"clasenna-go-backend/libs/cryptography"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRegionModule(router *gin.RouterGroup, db *gorm.DB) {
	controller := NewController(NewService(db))
	permissions := cryptography.AccessMiddleware{DB: db}

	for _, route := range []struct {
		path      string
		model     string
		getAll    gin.HandlerFunc
		getByCode gin.HandlerFunc
	}{
		{"/provinces", "provinces", controller.GetProvinces, controller.GetProvince},
		{"/regencies", "regencies", controller.GetRegencies, controller.GetRegency},
		{"/districts", "districts", controller.GetDistricts, controller.GetDistrict},
		{"/villages", "villages", controller.GetVillages, controller.GetVillage},
	} {
		group := router.Group(route.path)
		group.Use(cryptography.JWTMiddleware(db))
		group.GET("", permissions.Handler(route.model, "get-all"), route.getAll)
		group.GET("/:code", permissions.Handler(route.model, "get-by-id"), route.getByCode)
	}
}
