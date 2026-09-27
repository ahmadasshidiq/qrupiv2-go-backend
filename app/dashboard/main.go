package main

import (
	"context"
	"os/signal"
	"syscall"

	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/httpserver"
	loglib "clasenna-go-backend/libs/logger"
	"clasenna-go-backend/libs/stores"
	"clasenna-go-backend/src/dashboard"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

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
	dashboard.RegisterDashboardModule(router.Group("/api/v1"), dashboard.NewAggregator(cache))
	if err := httpserver.Run(ctx, ":"+helpers.ConfigString("DASHBOARD_PORT", "3008"), router, logger); err != nil {
		logger.Error("server stopped", "error", err)
	}
}
