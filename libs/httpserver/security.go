package httpserver

import (
	"clasenna-go-backend/libs/helpers"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const CSRFTokenCookie = "qrupi_csrf"

func SafeError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": code, "message": message})
}

func NewCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func CSRFProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		if isCSRFSafeAuthRoute(c.Request.URL.Path) {
			c.Next()
			return
		}
		if _, err := c.Cookie(helpers.ConfigString("AUTH_COOKIE_NAME", "qrupi_auth")); err != nil {
			c.Next()
			return
		}
		cookie, err := c.Cookie(CSRFTokenCookie)
		if err != nil || cookie == "" || cookie != c.GetHeader("X-CSRF-Token") {
			SafeError(c, http.StatusForbidden, "FORBIDDEN", "Permintaan tidak dapat diverifikasi.")
			return
		}
		c.Next()
	}
}

func isCSRFSafeAuthRoute(path string) bool {
	return strings.HasSuffix(path, "/auth/login") || strings.HasSuffix(path, "/auth/register")
}

// SecurityHeaders adds browser-enforced protections. Iframe sources are explicit
// and must be configured through IFRAME_ALLOWED_ORIGINS.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		frames := helpers.ConfigStrings("IFRAME_ALLOWED_ORIGINS", "'self'")
		for i := range frames {
			if frames[i] != "'self'" && !strings.Contains(frames[i], "://") {
				frames[i] = "https://" + frames[i]
			}
		}
		c.Header("Content-Security-Policy", "default-src 'self'; frame-src "+strings.Join(frames, " ")+"; frame-ancestors 'self'; object-src 'none'; base-uri 'self';")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}
