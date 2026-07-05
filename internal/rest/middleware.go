package rest

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/time/rate"
)

// bodyLimit rejects oversized payloads (413) before they are read — cheap
// protection against memory abuse. maxBytes <= 0 disables the guard.
func bodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes <= 0 {
			return
		}
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge,
				gin.H{"error": "request body too large"})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
	}
}

// ipRateLimiter is a token bucket per client IP (x/time/rate). It is a
// single-node guard — a shared gateway/LB does the global job in a real
// deployment. rps <= 0 disables it.
type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

// maxTrackedIPs bounds the limiter map; when exceeded it resets — crude,
// but it caps memory and only ever forgives, never blocks.
const maxTrackedIPs = 10_000

func newIPRateLimiter(rps float64, burst int) *ipRateLimiter {
	return &ipRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.limiters) > maxTrackedIPs {
		l.limiters = make(map[string]*rate.Limiter)
	}
	limiter, ok := l.limiters[ip]
	if !ok {
		limiter = rate.NewLimiter(l.rps, l.burst)
		l.limiters[ip] = limiter
	}
	return limiter.Allow()
}

func rateLimit(rps float64, burst int) gin.HandlerFunc {
	if rps <= 0 {
		return func(*gin.Context) {}
	}
	limiter := newIPRateLimiter(rps, burst)
	return func(c *gin.Context) {
		if !limiter.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				gin.H{"error": "rate limit exceeded"})
			return
		}
	}
}

// httpMetrics records request count and duration through the global OTel
// meter — a no-op unless the app installed a real meter provider (the
// Prometheus exporter behind /metrics).
func httpMetrics() gin.HandlerFunc {
	meter := otel.Meter("cj-beer-company/rest")
	requests, _ := meter.Int64Counter("http.server.requests",
		metric.WithDescription("HTTP requests handled"))
	duration, _ := meter.Float64Histogram("http.server.duration",
		metric.WithDescription("HTTP request duration in seconds"), metric.WithUnit("s"))
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		attrs := metric.WithAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.route", c.FullPath()),
			attribute.Int("http.status_code", c.Writer.Status()),
		)
		requests.Add(c.Request.Context(), 1, attrs)
		duration.Record(c.Request.Context(), time.Since(start).Seconds(), attrs)
	}
}
