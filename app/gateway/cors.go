package main

import (
	"net"
	"net/url"
	"os"

	"clasenna-go-backend/libs/helpers"
	"github.com/gin-contrib/cors"
)

func newCORSConfig() cors.Config {
	config := cors.Config{
		AllowOrigins:     helpers.ConfigStrings("CORS_ALLOWED_ORIGINS", "http://localhost:4000"),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Request-ID", "X-Correlation-ID"},
		ExposeHeaders:    []string{"X-Request-ID", "X-Correlation-ID"},
		AllowCredentials: true,
	}
	if os.Getenv("NODE_ENV") != "production" {
		config.AllowOriginFunc = isPrivateDevelopmentOrigin
	}
	return config
}

func isPrivateDevelopmentOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return false
	}
	if parsed.Hostname() == "localhost" {
		return true
	}
	ip := net.ParseIP(parsed.Hostname())
	return ip != nil && (ip.IsPrivate() || ip.IsLoopback())
}
