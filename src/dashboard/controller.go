package dashboard

import (
	"clasenna-go-backend/libs/helpers"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DashboardController struct{ Service *Aggregator }

func (c *DashboardController) Overview(ctx *gin.Context) {
	var dto OverviewDTO
	if err := ctx.ShouldBindQuery(&dto); err != nil {
		helpers.RespondError(ctx, "dashboard", http.StatusBadRequest, err)
		return
	}
	if ctx.GetHeader("X-User-ID") == "" {
		helpers.RespondErrorData(ctx, "dashboard", http.StatusUnauthorized, "authenticated user is required", nil)
		return
	}
	r, err := c.Service.Get(ctx, ctx.Request.Header, ctx.Request.URL.Query())
	if err != nil {
		helpers.RespondError(ctx, "dashboard", http.StatusBadGateway, err)
		return
	}
	helpers.RespondSuccess(ctx, "dashboard", http.StatusOK, r)
}
