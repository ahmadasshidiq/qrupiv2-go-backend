package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type visitor struct {
	window time.Time
	count  int
}

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]visitor
	limit    int
}

func newRateLimiter(limit int) *rateLimiter {
	return &rateLimiter{visitors: make(map[string]visitor), limit: limit}
}

func (l *rateLimiter) Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		blocked := l.record(ctx.ClientIP(), time.Now())
		if blocked {
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		ctx.Next()
	}
}

func (l *rateLimiter) record(clientIP string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	current := l.visitors[clientIP]
	if current.window.IsZero() || now.Sub(current.window) >= time.Minute {
		current = visitor{window: now}
	}
	current.count++
	l.visitors[clientIP] = current
	return current.count > l.limit
}
