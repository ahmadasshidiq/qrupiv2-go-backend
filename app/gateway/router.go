package main

import (
	"log/slog"

	"clasenna-go-backend/libs/helpers"
	"clasenna-go-backend/libs/httpserver"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func newRouter(logger *slog.Logger) *gin.Engine {
	router := gin.New()
	metrics := &httpserver.Metrics{}

	router.Use(gin.Recovery())
	router.Use(httpserver.Middleware(logger, metrics))
	router.Use(cors.New(newCORSConfig()))
	router.Use(newRateLimiter(helpers.ConfigInt("RATE_LIMIT_PER_MINUTE", 120)).Middleware())

	httpserver.RegisterProbes(router, metrics, nil)
	RegisterAllModules(router, logger)
	return router
}
