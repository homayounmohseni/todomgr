package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/homayounmohseni/todomgr/model"
	"github.com/homayounmohseni/todomgr/store"
)

type stubStore struct {
	listFn   func(ctx context.Context) ([]model.Todo, error)
	getFn    func(ctx context.Context, id int64) (model.Todo, error)
	createFn func(ctx context.Context, title string) (model.Todo, error)
	updateFn func(ctx context.Context, id int64, title string, done bool) (model.Todo, error)
	deleteFn func(ctx context.Context, id int64) error
}

func (s *stubStore) List(ctx context.Context) ([]model.Todo, error) {
	return s.listFn(ctx)
}

func (s *stubStore) Get(ctx context.Context, id int64) (model.Todo, error) {
	return s.getFn(ctx, id)
}

func (s *stubStore) Create(ctx context.Context, title string) (model.Todo, error) {
	return s.createFn(ctx, title)
}

func (s *stubStore) Update(ctx context.Context, id int64, title string, done bool) (model.Todo, error) {
	return s.updateFn(ctx, id, title, done)
}

func (s *stubStore) Delete(ctx context.Context, id int64) error {
	return s.deleteFn(ctx, id)
}

func doReq(r interface {
	ServeHTTP(w http.ResponseWriter, req *http.Request)
}, method, path, body string,
) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var req *http.Request
	if body == "" {
		req, _ = http.NewRequest(method, path, nil)
	} else {
		req, _ = http.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	}
	r.ServeHTTP(w, req)
	return w
}

func TestListSuccess(t *testing.T) {
	s := &stubStore{
		listFn: func(_ context.Context) ([]model.Todo, error) {
			return []model.Todo{{ID: 1, Title: "a"}, {ID: 2, Title: "b", Done: true}}, nil
		},
	}
	r := setupRouter(s)
	w := doReq(r, "GET", "/todos", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"title":"a"`) || !strings.Contains(w.Body.String(), `"title":"b"`) {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestListEmptyReturnsArray(t *testing.T) {
	s := &stubStore{
		listFn: func(_ context.Context) ([]model.Todo, error) { return nil, nil },
	}
	r := setupRouter(s)
	w := doReq(r, "GET", "/todos", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("body = %q, want []", w.Body.String())
	}
}

func TestListFailure(t *testing.T) {
	s := &stubStore{
		listFn: func(_ context.Context) ([]model.Todo, error) {
			return nil, errors.New("db down")
		},
	}
	r := setupRouter(s)
	if w := doReq(r, "GET", "/todos", ""); w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestCreateStoreFailure(t *testing.T) {
	s := &stubStore{
		createFn: func(_ context.Context, _ string) (model.Todo, error) {
			return model.Todo{}, errors.New("db down")
		},
	}
	r := setupRouter(s)
	if w := doReq(r, "POST", "/todos", `{"title":"x"}`); w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestCreateInvalidJSON(t *testing.T) {
	r := setupRouter(newFakeStore())
	if w := doReq(r, "POST", "/todos", `not-json`); w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestGetInvalidID(t *testing.T) {
	r := setupRouter(newFakeStore())
	if w := doReq(r, "GET", "/todos/abc", ""); w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestGetStoreFailure(t *testing.T) {
	s := &stubStore{
		getFn: func(_ context.Context, _ int64) (model.Todo, error) {
			return model.Todo{}, errors.New("db down")
		},
	}
	r := setupRouter(s)
	if w := doReq(r, "GET", "/todos/1", ""); w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestUpdateInvalidID(t *testing.T) {
	r := setupRouter(newFakeStore())
	if w := doReq(r, "PUT", "/todos/abc", `{"title":"x"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestUpdateValidation(t *testing.T) {
	r := setupRouter(newFakeStore())
	cases := []string{`{"title":""}`, `not-json`, `{}`}
	for _, b := range cases {
		if w := doReq(r, "PUT", "/todos/1", b); w.Code != http.StatusBadRequest {
			t.Fatalf("body %q: status = %d, want 400", b, w.Code)
		}
	}
}

func TestUpdateNotFound(t *testing.T) {
	s := &stubStore{
		updateFn: func(_ context.Context, _ int64, _ string, _ bool) (model.Todo, error) {
			return model.Todo{}, store.ErrNotFound
		},
	}
	r := setupRouter(s)
	if w := doReq(r, "PUT", "/todos/999", `{"title":"x"}`); w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestUpdateStoreFailure(t *testing.T) {
	s := &stubStore{
		updateFn: func(_ context.Context, _ int64, _ string, _ bool) (model.Todo, error) {
			return model.Todo{}, errors.New("db down")
		},
	}
	r := setupRouter(s)
	if w := doReq(r, "PUT", "/todos/1", `{"title":"x"}`); w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestDeleteInvalidID(t *testing.T) {
	r := setupRouter(newFakeStore())
	if w := doReq(r, "DELETE", "/todos/abc", ""); w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestDeleteNotFound(t *testing.T) {
	s := &stubStore{
		deleteFn: func(_ context.Context, _ int64) error { return store.ErrNotFound },
	}
	r := setupRouter(s)
	if w := doReq(r, "DELETE", "/todos/999", ""); w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestDeleteStoreFailure(t *testing.T) {
	s := &stubStore{
		deleteFn: func(_ context.Context, _ int64) error { return errors.New("db down") },
	}
	r := setupRouter(s)
	if w := doReq(r, "DELETE", "/todos/1", ""); w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}
