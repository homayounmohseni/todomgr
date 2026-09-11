package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestMigrateSuccess(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS todos").WillReturnResult(pgxmock.NewResult("DDL", 0))
	if err := Migrate(context.Background(), mock); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateFailure(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS todos").WillReturnError(errors.New("boom"))
	if err := Migrate(context.Background(), mock); err == nil {
		t.Fatal("want error")
	}
}

func TestPostgresListSuccess(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	now := time.Now()
	mock.ExpectQuery("SELECT id, title").WillReturnRows(
		pgxmock.NewRows([]string{"id", "title", "done", "created_at", "updated_at"}).
			AddRow(int64(1), "a", false, now, now).
			AddRow(int64(2), "b", true, now, now),
	)
	s := NewPostgresStore(mock)
	todos, err := s.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 2 || todos[0].Title != "a" || !todos[1].Done {
		t.Fatalf("unexpected todos: %+v", todos)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresListQueryError(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery("SELECT id, title").WillReturnError(errors.New("boom"))
	s := NewPostgresStore(mock)
	if _, err := s.List(context.Background()); err == nil {
		t.Fatal("want error")
	}
}

func TestPostgresListScanError(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	now := time.Now()
	mock.ExpectQuery("SELECT id, title").WillReturnRows(
		pgxmock.NewRows([]string{"id", "title", "done", "created_at", "updated_at"}).
			AddRow("not-an-int", "a", false, now, now),
	)
	s := NewPostgresStore(mock)
	if _, err := s.List(context.Background()); err == nil {
		t.Fatal("want scan error")
	}
}

func TestPostgresGetSuccess(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	now := time.Now()
	mock.ExpectQuery("SELECT id, title").WithArgs(int64(1)).WillReturnRows(
		pgxmock.NewRows([]string{"id", "title", "done", "created_at", "updated_at"}).
			AddRow(int64(1), "a", false, now, now),
	)
	s := NewPostgresStore(mock)
	got, err := s.Get(context.Background(), 1)
	if err != nil || got.Title != "a" {
		t.Fatalf("got %+v err %v", got, err)
	}
}

func TestPostgresGetNotFound(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery("SELECT id, title").WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)
	s := NewPostgresStore(mock)
	if _, err := s.Get(context.Background(), 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestPostgresGetError(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery("SELECT id, title").WithArgs(int64(1)).WillReturnError(errors.New("boom"))
	s := NewPostgresStore(mock)
	if _, err := s.Get(context.Background(), 1); err == nil {
		t.Fatal("want error")
	}
}

func TestPostgresCreateSuccess(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	now := time.Now()
	mock.ExpectQuery("INSERT INTO todos").WithArgs("buy milk").WillReturnRows(
		pgxmock.NewRows([]string{"id", "title", "done", "created_at", "updated_at"}).
			AddRow(int64(1), "buy milk", false, now, now),
	)
	s := NewPostgresStore(mock)
	got, err := s.Create(context.Background(), "buy milk")
	if err != nil || got.ID != 1 {
		t.Fatalf("got %+v err %v", got, err)
	}
}

func TestPostgresCreateError(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery("INSERT INTO todos").WithArgs("x").WillReturnError(errors.New("boom"))
	s := NewPostgresStore(mock)
	if _, err := s.Create(context.Background(), "x"); err == nil {
		t.Fatal("want error")
	}
}

func TestPostgresUpdateSuccess(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	now := time.Now()
	mock.ExpectQuery("UPDATE todos").WithArgs(int64(1), "y", true).WillReturnRows(
		pgxmock.NewRows([]string{"id", "title", "done", "created_at", "updated_at"}).
			AddRow(int64(1), "y", true, now, now),
	)
	s := NewPostgresStore(mock)
	got, err := s.Update(context.Background(), 1, "y", true)
	if err != nil || !got.Done || got.Title != "y" {
		t.Fatalf("got %+v err %v", got, err)
	}
}

func TestPostgresUpdateNotFound(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery("UPDATE todos").WithArgs(int64(999), "y", true).WillReturnError(pgx.ErrNoRows)
	s := NewPostgresStore(mock)
	if _, err := s.Update(context.Background(), 999, "y", true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestPostgresUpdateError(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery("UPDATE todos").WithArgs(int64(1), "y", true).WillReturnError(errors.New("boom"))
	s := NewPostgresStore(mock)
	if _, err := s.Update(context.Background(), 1, "y", true); err == nil {
		t.Fatal("want error")
	}
}

func TestPostgresDeleteSuccess(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectExec("DELETE FROM todos").WithArgs(int64(1)).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	s := NewPostgresStore(mock)
	if err := s.Delete(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresDeleteNotFound(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectExec("DELETE FROM todos").WithArgs(int64(999)).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))
	s := NewPostgresStore(mock)
	if err := s.Delete(context.Background(), 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestPostgresDeleteError(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectExec("DELETE FROM todos").WithArgs(int64(1)).WillReturnError(errors.New("boom"))
	s := NewPostgresStore(mock)
	if err := s.Delete(context.Background(), 1); err == nil {
		t.Fatal("want error")
	}
}
