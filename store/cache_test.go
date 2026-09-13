package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/homayounmohseni/todomgr/model"
)

type fakeCache struct {
	data    map[string]string
	getErr  error
	setErr  error
	delErr  error
	deleted []string
}

func newFakeCache() *fakeCache {
	return &fakeCache{data: map[string]string{}}
}

func (c *fakeCache) Get(_ context.Context, key string) (string, error) {
	if c.getErr != nil {
		return "", c.getErr
	}
	v, ok := c.data[key]
	if !ok {
		return "", errors.New("miss")
	}
	return v, nil
}

func (c *fakeCache) Set(_ context.Context, key, value string, _ time.Duration) error {
	if c.setErr != nil {
		return c.setErr
	}
	c.data[key] = value
	return nil
}

func (c *fakeCache) Del(_ context.Context, key string) error {
	if c.delErr != nil {
		return c.delErr
	}
	c.deleted = append(c.deleted, key)
	delete(c.data, key)
	return nil
}

type countingStore struct {
	task      model.Task
	getCalls  int
	listCalls int
	getErr    error
}

func (s *countingStore) List(_ context.Context, _ ListFilter) ([]model.Task, error) {
	s.listCalls++
	return []model.Task{s.task}, nil
}

func (s *countingStore) Get(_ context.Context, id int64) (model.Task, error) {
	s.getCalls++
	if s.getErr != nil {
		return model.Task{}, s.getErr
	}
	t := s.task
	t.ID = id
	return t, nil
}

func (s *countingStore) Create(_ context.Context, title, assignee string) (model.Task, error) {
	return model.Task{ID: 1, Title: title, Assignee: assignee}, nil
}

func (s *countingStore) Update(_ context.Context, id int64, title string, status bool, assignee string) (model.Task, error) {
	return model.Task{ID: id, Title: title, Status: status, Assignee: assignee}, nil
}

func (s *countingStore) Delete(_ context.Context, _ int64) error { return nil }

func TestCachedGetMissThenHit(t *testing.T) {
	inner := &countingStore{task: model.Task{Title: "a"}}
	c := NewCachedTaskStore(inner, newFakeCache(), time.Minute)

	first, err := c.Get(context.Background(), 5)
	if err != nil || first.ID != 5 {
		t.Fatalf("miss: got %+v err %v", first, err)
	}
	second, err := c.Get(context.Background(), 5)
	if err != nil || second.Title != "a" {
		t.Fatalf("hit: got %+v err %v", second, err)
	}
	if inner.getCalls != 1 {
		t.Fatalf("store called %d times, want 1", inner.getCalls)
	}
}

func TestCachedGetKeysArePerID(t *testing.T) {
	inner := &countingStore{}
	c := NewCachedTaskStore(inner, newFakeCache(), time.Minute)
	if _, err := c.Get(context.Background(), 5); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(context.Background(), 6); err != nil {
		t.Fatal(err)
	}
	if inner.getCalls != 2 {
		t.Fatalf("store called %d times, want 2 (different keys)", inner.getCalls)
	}
}

func TestCachedUpdateEvictsOnlyThatID(t *testing.T) {
	inner := &countingStore{}
	fc := newFakeCache()
	c := NewCachedTaskStore(inner, fc, time.Minute)
	if _, err := c.Get(context.Background(), 5); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(context.Background(), 6); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Update(context.Background(), 5, "x", true, ""); err != nil {
		t.Fatal(err)
	}
	if len(fc.deleted) != 1 || fc.deleted[0] != "task:5" {
		t.Fatalf("deleted = %v", fc.deleted)
	}
	if _, err := c.Get(context.Background(), 5); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(context.Background(), 6); err != nil {
		t.Fatal(err)
	}
	if inner.getCalls != 3 {
		t.Fatalf("store called %d times, want 3 (5 re-read, 6 still cached)", inner.getCalls)
	}
}

func TestCachedDeleteEvicts(t *testing.T) {
	inner := &countingStore{}
	fc := newFakeCache()
	c := NewCachedTaskStore(inner, fc, time.Minute)
	if _, err := c.Get(context.Background(), 5); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete(context.Background(), 5); err != nil {
		t.Fatal(err)
	}
	if len(fc.deleted) != 1 || fc.deleted[0] != "task:5" {
		t.Fatalf("deleted = %v", fc.deleted)
	}
}

func TestCachedGetNotFoundIsNotCached(t *testing.T) {
	inner := &countingStore{getErr: ErrNotFound}
	fc := newFakeCache()
	c := NewCachedTaskStore(inner, fc, time.Minute)
	if _, err := c.Get(context.Background(), 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
	if len(fc.data) != 0 {
		t.Fatalf("miss should not populate cache: %v", fc.data)
	}
}

func TestCachedFailOpen(t *testing.T) {
	inner := &countingStore{task: model.Task{Title: "a"}}
	fc := newFakeCache()
	fc.getErr = errors.New("redis down")
	fc.setErr = errors.New("redis down")
	fc.delErr = errors.New("redis down")
	c := NewCachedTaskStore(inner, fc, time.Minute)
	got, err := c.Get(context.Background(), 5)
	if err != nil || got.Title != "a" {
		t.Fatalf("got %+v err %v", got, err)
	}
	if _, err := c.Update(context.Background(), 5, "x", true, ""); err != nil {
		t.Fatalf("write should succeed despite cache: %v", err)
	}
}

func TestCachedCorruptEntryIsMiss(t *testing.T) {
	inner := &countingStore{task: model.Task{Title: "a"}}
	fc := newFakeCache()
	c := NewCachedTaskStore(inner, fc, time.Minute)
	fc.data["task:5"] = "not-json{{{"
	got, err := c.Get(context.Background(), 5)
	if err != nil || got.Title != "a" {
		t.Fatalf("got %+v err %v", got, err)
	}
	if inner.getCalls != 1 {
		t.Fatalf("store called %d times, want 1", inner.getCalls)
	}
}

func TestCachedCreateTouchesNothing(t *testing.T) {
	inner := &countingStore{}
	fc := newFakeCache()
	c := NewCachedTaskStore(inner, fc, time.Minute)
	if _, err := c.Get(context.Background(), 5); err != nil {
		t.Fatal(err)
	}
	got, err := c.Create(context.Background(), "x", "Sara")
	if err != nil || got.Assignee != "Sara" {
		t.Fatalf("got %+v err %v", got, err)
	}
	if len(fc.deleted) != 0 {
		t.Fatalf("deleted = %v, want none (new id has no cache entry)", fc.deleted)
	}
}

func TestCachedListPassthrough(t *testing.T) {
	inner := &countingStore{task: model.Task{Title: "a"}}
	c := NewCachedTaskStore(inner, newFakeCache(), time.Minute)
	tasks, err := c.List(context.Background(), ListFilter{Limit: 20})
	if err != nil || len(tasks) != 1 || inner.listCalls != 1 {
		t.Fatalf("got %+v err %v calls %d", tasks, err, inner.listCalls)
	}
}

func TestCachedWriteErrorSkipsInvalidate(t *testing.T) {
	inner := &errStore{}
	fc := newFakeCache()
	c := NewCachedTaskStore(inner, fc, time.Minute)
	if _, err := c.Update(context.Background(), 5, "x", true, ""); err == nil {
		t.Fatal("want error")
	}
	if err := c.Delete(context.Background(), 5); err == nil {
		t.Fatal("want error")
	}
	if len(fc.deleted) != 0 {
		t.Fatalf("deleted = %v, want none", fc.deleted)
	}
}

type errStore struct {
	countingStore
}

func (s *errStore) Update(_ context.Context, _ int64, _ string, _ bool, _ string) (model.Task, error) {
	return model.Task{}, errors.New("db down")
}

func (s *errStore) Delete(_ context.Context, _ int64) error {
	return errors.New("db down")
}
