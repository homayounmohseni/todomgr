package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/homayounmohseni/todomgr/model"
	"github.com/homayounmohseni/todomgr/store"

	"github.com/gin-gonic/gin"
)

type stubStore struct {
	todos map[int64]model.Todo
}

func (s *stubStore) List(_ context.Context) ([]model.Todo, error) {
	return []model.Todo{{ID: 1, Title: "a"}}, nil
}

func (s *stubStore) Get(_ context.Context, id int64) (model.Todo, error) {
	if id != 1 {
		return model.Todo{}, store.ErrNotFound
	}
	return model.Todo{ID: 1, Title: "a"}, nil
}

func (s *stubStore) Create(_ context.Context, title string) (model.Todo, error) {
	if title == "boom" {
		return model.Todo{}, errors.New("db down")
	}
	return model.Todo{ID: 1, Title: title}, nil
}

func (s *stubStore) Update(_ context.Context, id int64, title string, done bool) (model.Todo, error) {
	return model.Todo{ID: id, Title: title, Done: done}, nil
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

func TestRouterTodoRoundtrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRouter(&stubStore{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/todos", bytes.NewBufferString(`{"title":"a"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/todos", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d", w.Code)
	}
}
