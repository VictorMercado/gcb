package main

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// httpRequestsTotal counts all HTTP requests with labels for detailed analysis
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code", "hostname", "client_ip"},
	)

	// httpRequestDuration measures request latency
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// signedURLCreatedTotal counts successful signed URL generations
	signedURLCreatedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "signedurl_created_total",
			Help: "Total number of signed URLs created",
		},
		[]string{"hostname", "client_ip"},
	)
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// MetricsMiddleware records Prometheus metrics for each request
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip metrics endpoint to avoid recursion
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		// Start timer
		timer := prometheus.NewTimer(httpRequestDuration.WithLabelValues(r.Method, r.URL.Path))
		defer timer.ObserveDuration()

		// Get hostname and client IP
		hostname := r.Host
		clientIP := getClientIP(r)

		// Wrap response writer to capture status code
		wrapped := newResponseWriter(w)

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Record request metrics
		httpRequestsTotal.WithLabelValues(
			r.Method,
			r.URL.Path,
			strconv.Itoa(wrapped.statusCode),
			hostname,
			clientIP,
		).Inc()
	})
}

// IncrementSignedURLCounter increments the signed URL counter
func IncrementSignedURLCounter(hostname, clientIP string) {
	signedURLCreatedTotal.WithLabelValues(hostname, clientIP).Inc()
}
