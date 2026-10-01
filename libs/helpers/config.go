package helpers

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// AuthCookieName returns the session cookie for the frontend that initiated
// the request. This keeps portal, student, and instructor sessions isolated
// while allowing them to use the same API deployment.
func AuthCookieName(r *http.Request) string {
	return cookieNameForOrigin(r, "AUTH_COOKIE_NAME", "qrupi_auth")
}

func CSRFCookieName(r *http.Request) string {
	return cookieNameForOrigin(r, "CSRF_COOKIE_NAME", "qrupi_csrf")
}

func cookieNameForOrigin(r *http.Request, key, fallback string) string {
	origin := strings.ToLower(r.Header.Get("Origin"))
	switch {
	case configuredOriginMatches(origin, "AUTH_COOKIE_ORIGIN_PORTAL"):
		return ConfigString(key+"_PORTAL", ConfigString(key, fallback))
	case configuredOriginMatches(origin, "AUTH_COOKIE_ORIGIN_PELAJAR"):
		return ConfigString(key+"_PELAJAR", ConfigString(key, fallback))
	case configuredOriginMatches(origin, "AUTH_COOKIE_ORIGIN_INSTRUCTOR"):
		return ConfigString(key+"_INSTRUCTOR", ConfigString(key, fallback))
	default:
		return ConfigString(key, fallback)
	}
}

func configuredOriginMatches(origin, key string) bool {
	configured := strings.TrimRight(strings.ToLower(os.Getenv(key)), "/")
	return configured != "" && strings.TrimRight(origin, "/") == configured
}

func ConfigString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func ConfigStrings(key, fallback string) []string {
	parts := strings.Split(ConfigString(key, fallback), ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func ConfigInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return value
}

func ConfigDuration(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return value
}
