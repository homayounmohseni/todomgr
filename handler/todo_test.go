package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/homayounmohseni/todomgr/model"
	"github.com/homayounmohseni/todomgr/store"

	"github.com/gin-gonic/gin"
)

type fakeStore struct {
	todos  map[int64]model.Todo
	nextID int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{todos: map[int64]model.Todo{}, nextID: 1}
}

func (f *fakeStore) List(_ context.Context) ([]model.Todo, error) {
	out := []model.Todo{}
	for _, t := range f.todos {
		out = append(out, t)
	}
	return out, nil
}

func (f *fakeStore) Get(_ context.Context, id int64) (model.Todo, error) {
	t, ok := f.todos[id]
	if !ok {
		return model.Todo{}, store.ErrNotFound
	}
	return t, nil
}

func (f *fakeStore) Create(_ context.Context, title string) (model.Todo, error) {
	t := model.Todo{ID: f.nextID, Title: title}
	f.todos[t.ID] = t
	f.nextID++
	return t, nil
}

func (f *fakeStore) Update(_ context.Context, id int64, title string, done bool) (model.Todo, error) {
	t, ok := f.todos[id]
	if !ok {
		return model.Todo{}, store.ErrNotFound
	}
	t.Title = title
	t.Done = done
	f.todos[id] = t
	return t, nil
}

func (f *fakeStore) Delete(_ context.Context, id int64) error {
	if _, ok := f.todos[id]; !ok {
		return store.ErrNotFound
	}
	delete(f.todos, id)
	return nil
}

func setupRouter(s store.TodoStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewTodoHandler(s).RegisterRoutes(r)
	return r
}

func TestCreateAndGet(t *testing.T) {
	r := setupRouter(newFakeStore())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/todos", bytes.NewBufferString(`{"title":"buy milk"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201 (%s)", w.Code, w.Body.String())
	}
	var created model.Todo
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Title != "buy milk" {
		t.Fatalf("title = %q", created.Title)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/todos/1", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get status = %d", w.Code)
	}
}

func TestCreateValidation(t *testing.T) {
	r := setupRouter(newFakeStore())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/todos", bytes.NewBufferString(`{"title":""}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestGetNotFound(t *testing.T) {
	r := setupRouter(newFakeStore())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/todos/999", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestUpdateAndDelete(t *testing.T) {
	fs := newFakeStore()
	r := setupRouter(fs)
	if _, err := fs.Create(context.Background(), "x"); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/todos/1", bytes.NewBufferString(`{"title":"y","done":true}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update status = %d (%s)", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/todos/1", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d", w.Code)
	}
}
