package main

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"clasenna-go-backend/libs/cryptography"
	"clasenna-go-backend/libs/helpers"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func RegisterAllModules(router *gin.Engine, logger *slog.Logger) {
	router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.Any("/api/v1/*path", func(c *gin.Context) {
		path := c.Param("path")
		target := helpers.ConfigString("CORE_SERVICE_URL", "http://localhost:3002")
		switch {
		case strings.HasPrefix(path, "/auth/"):
			target = helpers.ConfigString("AUTH_SERVICE_URL", "http://localhost:3001")
		case isLearningRoute(path):
			target = helpers.ConfigString("LEARNING_SERVICE_URL", "http://localhost:3003")
		case isActivityRoute(path):
			target = helpers.ConfigString("ACTIVITY_SERVICE_URL", "http://localhost:3006")
		case isRegionRoute(path):
			target = helpers.ConfigString("REGION_SERVICE_URL", "http://localhost:3007")
		case strings.HasPrefix(path, "/notifications/"):
			target = helpers.ConfigString("NOTIFICATION_SERVICE_URL", "http://localhost:3004")
		case strings.HasPrefix(path, "/reports/"):
			target = helpers.ConfigString("REPORTING_SERVICE_URL", "http://localhost:3005")
		case strings.HasPrefix(path, "/export/"):
			target = helpers.ConfigString("REPORTING_SERVICE_URL", "http://localhost:3005")
		}
		if !isPublicRoute(c.Request.Method, path) && !authorize(c) {
			return
		}
		upstream, err := url.Parse(target)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid upstream"})
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(upstream)
		proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
			logger.Error("upstream unavailable", "error", err)
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	})
}

func isRegionRoute(path string) bool {
	for _, prefix := range []string{"/provinces", "/regencies", "/districts", "/villages"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func isActivityRoute(path string) bool {
	for _, prefix := range []string{"/activities", "/activity-items", "/activity-categories", "/activity-bulk-jobs"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func isLearningRoute(path string) bool {
	for _, prefix := range []string{
		"/attendance-logs",
		"/attendance-absence-reasons",
		"/learning-groups",
		"/learning-group-members",
		"/learning-resources",
		"/quizzes",
		"/quiz-sessions",
	} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func isPublicRoute(method, path string) bool {
	if strings.HasPrefix(path, "/auth/") {
		return true
	}
	return method == http.MethodGet && (path == "/roles/master-permissions" || path == "/users/template-excel")
}

func authorize(c *gin.Context) bool {
	tokenString, ok := extractToken(c.GetHeader("Authorization"))
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "bearer token required"})
		return false
	}
	_, claims, err := cryptography.ValidateToken(tokenString)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return false
	}
	c.Request.Header.Set("Authorization", "Bearer "+tokenString)
	for claim, header := range map[string]string{"user_id": "X-User-ID", "institution_id": "X-Institution-ID", "role_id": "X-Role-ID"} {
		if value, ok := claims[claim].(string); ok {
			c.Request.Header.Set(header, value)
		}
	}
	return true
}

func extractToken(header string) (string, bool) {
	parts := strings.Fields(header)
	switch {
	case len(parts) == 1 && parts[0] != "" && !strings.EqualFold(parts[0], "Bearer"):
		return parts[0], true
	case len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && parts[1] != "":
		return parts[1], true
	default:
		return "", false
	}
}
