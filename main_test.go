package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/homayounmohseni/todomgr/model"
	"github.com/homayounmohseni/todomgr/store"

	"github.com/gin-gonic/gin"
)

type stubStore struct{}

func (s *stubStore) List(_ context.Context, _ store.ListFilter) ([]model.Task, error) {
	return []model.Task{{ID: 1, Title: "a"}}, nil
}

func (s *stubStore) Get(_ context.Context, id int64) (model.Task, error) {
	if id != 1 {
		return model.Task{}, store.ErrNotFound
	}
	return model.Task{ID: 1, Title: "a"}, nil
}

func (s *stubStore) Create(_ context.Context, title, assignee string) (model.Task, error) {
	if title == "boom" {
		return model.Task{}, errors.New("db down")
	}
	return model.Task{ID: 1, Title: title, Assignee: assignee}, nil
}

func (s *stubStore) Update(_ context.Context, id int64, title string, status bool, assignee string) (model.Task, error) {
	return model.Task{ID: id, Title: title, Status: status, Assignee: assignee}, nil
}

func (s *stubStore) Delete(_ context.Context, _ int64) error { return nil }

func TestConfigFromEnvDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PORT", "")
	dbURL, port := configFromEnv()
	if dbURL != "postgres://todo:todo@localhost:5432/todo?sslmode=disable" {
		t.Fatalf("dbURL = %q", dbURL)
	}
	if port != "8080" {
		t.Fatalf("port = %q", port)
	}
}

func TestConfigFromEnvCustom(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("PORT", "9999")
	dbURL, port := configFromEnv()
	if dbURL != "postgres://x" || port != "9999" {
		t.Fatalf("got %q %q", dbURL, port)
	}
}

func TestRedisURLFromEnv(t *testing.T) {
	t.Setenv("REDIS_URL", "")
	if got := redisURLFromEnv(); got != "redis://localhost:6379/0" {
		t.Fatalf("got %q", got)
	}
	t.Setenv("REDIS_URL", "redis://redis:6379/0")
	if got := redisURLFromEnv(); got != "redis://redis:6379/0" {
		t.Fatalf("got %q", got)
	}
}

func TestHealthz(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter(&stubStore{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter(&stubStore{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/tasks", bytes.NewBufferString(`{"title":"a"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d", w.Code)
	}

	srv := httptest.NewServer(r)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	for _, want := range []string{"http_requests_total", "http_request_latency_seconds", "tasks_count"} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("metrics missing %q", want)
		}
	}
}

func TestRouterTaskRoundtrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter(&stubStore{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/tasks", bytes.NewBufferString(`{"title":"a"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/tasks", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d", w.Code)
	}
}
