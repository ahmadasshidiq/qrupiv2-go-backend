package dashboard

import "github.com/gin-gonic/gin"

func RegisterDashboardModule(router *gin.RouterGroup, service *Aggregator) {
	controller := &DashboardController{Service: service}
	group := router.Group("/dashboard")
	group.GET("", controller.Overview)
}
