package metrics

import (
	"bufio"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	uptimeStart = time.Now()

	RequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "portal_requests_total",
			Help: "Total HTTP requests by method, path, and status",
		},
		[]string{"method", "path", "status"},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "portal_request_duration_seconds",
			Help:    "Request duration in seconds by method and path",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	ActiveServices = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "portal_services_active",
			Help: "Number of active (enabled) services",
		},
	)

	AuthAttempts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "portal_auth_attempts_total",
			Help: "Authentication attempts by result",
		},
		[]string{"result"},
	)

	HealthCheckResults = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "portal_health_check_results_total",
			Help: "Health check results by service and status",
		},
		[]string{"service", "status"},
	)

	SessionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "portal_sessions_active",
			Help: "Number of active sessions",
		},
	)

	Uptime = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "portal_uptime_seconds",
			Help: "Server uptime in seconds",
		},
		func() float64 { return time.Since(uptimeStart).Seconds() },
	)
)

func init() {
	prometheus.MustRegister(RequestCount, RequestDuration, ActiveServices,
		AuthAttempts, HealthCheckResults, SessionsActive, Uptime)
}

func Instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(wrapped, r)
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(wrapped.status)
		RequestCount.WithLabelValues(r.Method, r.URL.Path, status).Inc()
		RequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func Handler() http.Handler {
	slog.Info("prometheus metrics enabled at /metrics")
	return promhttp.Handler()
}
