package dashboard

import (
	"clasenna-go-backend/libs/helpers"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type DashboardController struct{ Service *Aggregator }

func (c *DashboardController) Overview(ctx *gin.Context) {
	c.dashboard(ctx, "")
}

func (c *DashboardController) ByRole(ctx *gin.Context) {
	role, ok := roleAliases[ctx.Param("role")]
	if !ok {
		helpers.RespondErrorData(ctx, "dashboard", http.StatusNotFound, "unsupported dashboard role", nil)
		return
	}
	c.dashboard(ctx, role)
}

func (c *DashboardController) dashboard(ctx *gin.Context, expectedRole string) {
	var dto OverviewDTO
	if err := ctx.ShouldBindQuery(&dto); err != nil {
		helpers.RespondError(ctx, "dashboard", http.StatusBadRequest, err)
		return
	}
	if ctx.GetHeader("X-User-ID") == "" {
		helpers.RespondErrorData(ctx, "dashboard", http.StatusUnauthorized, "authenticated user is required", nil)
		return
	}
	r, err := c.Service.GetForRole(ctx, ctx.Request.Header, ctx.Request.URL.Query(), expectedRole)
	if err != nil {
		status := http.StatusBadGateway
		if strings.HasPrefix(err.Error(), "dashboard endpoint requires role") {
			status = http.StatusForbidden
			ctx.JSON(status, gin.H{"code": status, "error": "FORBIDDEN", "message": "Anda tidak memiliki izin mengakses dashboard ini.", "status": "error"})
			return
		}
		ctx.JSON(status, gin.H{"code": status, "error": "SERVER_ERROR", "message": "Terjadi kesalahan pada server. Silakan coba lagi.", "status": "error"})
		return
	}
	helpers.RespondSuccess(ctx, "dashboard", http.StatusOK, r)
}
