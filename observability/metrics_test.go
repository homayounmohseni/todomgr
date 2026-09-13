package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMiddlewareRecords(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Middleware())
	r.GET("/tasks/:id", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/metrics", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/tasks/1", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if got := testutil.ToFloat64(requestsTotal.WithLabelValues("GET", "/tasks/:id", "200")); got < 1 {
		t.Fatalf("requestsTotal = %v, want >= 1", got)
	}
	if got := testutil.CollectAndCount(requestLatency); got < 1 {
		t.Fatalf("latency observations = %d, want >= 1", got)
	}

	before := testutil.ToFloat64(requestsTotal.WithLabelValues("GET", "/metrics", "200"))
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/metrics", nil)
	r.ServeHTTP(w, req)
	if got := testutil.ToFloat64(requestsTotal.WithLabelValues("GET", "/metrics", "200")); got != before {
		t.Fatalf("/metrics should be skipped, before %v after %v", before, got)
	}
}

func TestCounterFuncs(t *testing.T) {
	TasksInc()
	TasksDec()
	TasksSet(3)
	if got := testutil.ToFloat64(tasksCount); got != 3 {
		t.Fatalf("tasksCount = %v, want 3", got)
	}
	CacheHit()
	CacheMiss()
	if got := testutil.ToFloat64(cacheHits); got < 1 {
		t.Fatalf("cacheHits = %v, want >= 1", got)
	}
	if got := testutil.ToFloat64(cacheMisses); got < 1 {
		t.Fatalf("cacheMisses = %v, want >= 1", got)
	}
}
