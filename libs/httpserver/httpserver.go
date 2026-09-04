package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const CorrelationIDHeader = "X-Correlation-ID"
const RequestIDHeader = "X-Request-ID"

type Metrics struct{ Requests atomic.Uint64 }

func Middleware(logger *slog.Logger, metrics *Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.NewString()
		}
		correlationID := c.GetHeader(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = requestID
		}
		c.Set("request_id", requestID)
		c.Set("correlation_id", correlationID)
		c.Header(RequestIDHeader, requestID)
		c.Header(CorrelationIDHeader, correlationID)
		metrics.Requests.Add(1)
		c.Next()
		logger.InfoContext(c.Request.Context(), "http request", "method", c.Request.Method, "path", c.Request.URL.Path,
			"status", c.Writer.Status(), "duration_ms", time.Since(started).Milliseconds(), "request_id", requestID, "correlation_id", correlationID)
	}
}

func RegisterProbes(r *gin.Engine, metrics *Metrics, ready func(context.Context) error) {
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/ready", func(c *gin.Context) {
		if ready != nil && ready(c.Request.Context()) != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	r.GET("/metrics", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain; version=0.0.4")
		c.String(http.StatusOK, "# TYPE qrupi_http_requests_total counter\nqrupi_http_requests_total %d\n", metrics.Requests.Load())
	})
}

func Run(ctx context.Context, address string, handler http.Handler, logger *slog.Logger) error {
	server := &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	errCh := make(chan error, 1)
	go func() { errCh <- server.ListenAndServe() }()
	select {
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		logger.Info("shutting down HTTP server")
		return server.Shutdown(shutdownCtx)
	}
}
