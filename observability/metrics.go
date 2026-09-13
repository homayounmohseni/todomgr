package observability

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)

	requestLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_latency_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	tasksCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tasks_count",
			Help: "Current number of tasks.",
		},
	)

	cacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total cache hits for single task queries.",
		},
	)

	cacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total cache misses for single task queries.",
		},
	)
)

func TasksInc() { tasksCount.Inc() }

func TasksDec() { tasksCount.Dec() }

func TasksSet(n float64) { tasksCount.Set(n) }

func CacheHit() { cacheHits.Inc() }

func CacheMiss() { cacheMisses.Inc() }

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.FullPath() == "/metrics" {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		status := strconv.Itoa(c.Writer.Status())
		requestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		requestLatency.WithLabelValues(c.Request.Method, path).Observe(time.Since(start).Seconds())
	}
}
