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
	tasks  map[int64]model.Task
	nextID int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{tasks: map[int64]model.Task{}, nextID: 1}
}

func (f *fakeStore) List(_ context.Context, _ store.ListFilter) ([]model.Task, error) {
	out := []model.Task{}
	for _, t := range f.tasks {
		out = append(out, t)
	}
	return out, nil
}

func (f *fakeStore) Get(_ context.Context, id int64) (model.Task, error) {
	t, ok := f.tasks[id]
	if !ok {
		return model.Task{}, store.ErrNotFound
	}
	return t, nil
}

func (f *fakeStore) Create(_ context.Context, title, assignee string) (model.Task, error) {
	t := model.Task{ID: f.nextID, Title: title, Assignee: assignee}
	f.tasks[t.ID] = t
	f.nextID++
	return t, nil
}

func (f *fakeStore) Update(_ context.Context, id int64, title string, status bool, assignee string) (model.Task, error) {
	t, ok := f.tasks[id]
	if !ok {
		return model.Task{}, store.ErrNotFound
	}
	t.Title = title
	t.Status = status
	t.Assignee = assignee
	f.tasks[id] = t
	return t, nil
}

func (f *fakeStore) Delete(_ context.Context, id int64) error {
	if _, ok := f.tasks[id]; !ok {
		return store.ErrNotFound
	}
	delete(f.tasks, id)
	return nil
}

func setupRouter(s store.TaskStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewTaskHandler(s).RegisterRoutes(r)
	return r
}

func TestCreateAndGet(t *testing.T) {
	r := setupRouter(newFakeStore())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/tasks", bytes.NewBufferString(`{"title":"buy milk"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201 (%s)", w.Code, w.Body.String())
	}
	var created model.Task
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Title != "buy milk" {
		t.Fatalf("title = %q", created.Title)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/tasks/1", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get status = %d", w.Code)
	}
}

func TestCreateValidation(t *testing.T) {
	r := setupRouter(newFakeStore())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/tasks", bytes.NewBufferString(`{"title":""}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestGetNotFound(t *testing.T) {
	r := setupRouter(newFakeStore())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/tasks/999", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestUpdateAndDelete(t *testing.T) {
	fs := newFakeStore()
	r := setupRouter(fs)
	if _, err := fs.Create(context.Background(), "x", ""); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/tasks/1", bytes.NewBufferString(`{"title":"y","status":true}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update status = %d (%s)", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/tasks/1", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d", w.Code)
	}
}
