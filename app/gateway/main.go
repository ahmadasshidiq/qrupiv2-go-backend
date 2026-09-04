package main

import (
	"context"
	"os/signal"
	"syscall"

	_ "clasenna-go-backend/docs"
	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/httpserver"
	loglib "clasenna-go-backend/libs/logger"

	"github.com/joho/godotenv"
)

// @title QRUPI V2 API
// @version 2.0
// @description Public API documentation served through QRUPI API Gateway.
// @BasePath /api/v1
// @schemes http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	_ = godotenv.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := loglib.New("gateway-service")
	router := newRouter(logger)
	address := ":" + helpers.ConfigString("GATEWAY_PORT", "3000")

	if err := httpserver.Run(ctx, address, router, logger); err != nil {
		logger.Error("server stopped", "error", err)
	}
}
