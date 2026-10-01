package main

import (
	"context"
	"net/http"
	"os/signal"
	"strings"
	"syscall"

	"clasenna-go-backend/libs/cryptography"
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/httpserver"
	loglib "clasenna-go-backend/libs/logger"
	"clasenna-go-backend/libs/stores"
	"clasenna-go-backend/src/dashboard"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func dashboardAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""
		if header := c.GetHeader("Authorization"); header != "" {
			token = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(header, "Bearer "), "bearer "))
		}
		if token == "" {
			token, _ = c.Cookie(helpers.ConfigString("AUTH_COOKIE_NAME", "qrupi_auth"))
		}
		_, claims, err := cryptography.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "error": "UNAUTHORIZED", "message": "Sesi Anda telah berakhir. Silakan login kembali.", "status": "error"})
			return
		}
		for claim, header := range map[string]string{"user_id": "X-User-ID", "institution_id": "X-Institution-ID", "role_id": "X-Role-ID", "region_level": "X-Region-Level", "region_code": "X-Region-Code"} {
			if value, ok := claims[claim].(string); ok {
				c.Request.Header.Set(header, value)
			}
		}
		if c.GetHeader("X-User-ID") == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "error": "UNAUTHORIZED", "message": "Sesi Anda tidak valid. Silakan login kembali.", "status": "error"})
			return
		}
		c.Next()
	}
}

func main() {
	_ = godotenv.Load()
	logger := loglib.New("dashboard-service")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cache, err := stores.OpenRedis()
	if err != nil {
		logger.Warn("redis unavailable; dashboard cache disabled", "error", err)
	}
	router := gin.New()
	router.Use(gin.Recovery())
	metrics := &httpserver.Metrics{}
	router.Use(httpserver.Middleware(logger, metrics))
	httpserver.RegisterProbes(router, metrics, nil)
	router.Use(dashboardAuth())
	dashboard.RegisterDashboardModule(router.Group("/api/v1"), dashboard.NewAggregator(cache))
	if err := httpserver.Run(ctx, ":"+helpers.ConfigString("DASHBOARD_PORT", "3008"), router, logger); err != nil {
		logger.Error("server stopped", "error", err)
	}
}
