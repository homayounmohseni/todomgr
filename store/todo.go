package store

import (
	"context"
	"errors"

	"github.com/homayounmohseni/todomgr/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type TodoStore interface {
	List(ctx context.Context) ([]model.Todo, error)
	Get(ctx context.Context, id int64) (model.Todo, error)
	Create(ctx context.Context, title string) (model.Todo, error)
	Update(ctx context.Context, id int64, title string, done bool) (model.Todo, error)
	Delete(ctx context.Context, id int64) error
}

type DBPool interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PostgresStore struct {
	Pool DBPool
}

func NewPostgresStore(pool DBPool) *PostgresStore {
	return &PostgresStore{Pool: pool}
}

func Migrate(ctx context.Context, pool DBPool) error {
	_, err := pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS todos (
  id BIGSERIAL PRIMARY KEY,
  title TEXT NOT NULL CHECK (char_length(title) BETWEEN 1 AND 500),
  done BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`)
	return err
}

func (s *PostgresStore) List(ctx context.Context) ([]model.Todo, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, title, done, created_at, updated_at FROM todos ORDER BY id ASC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	todos := []model.Todo{}
	for rows.Next() {
		var t model.Todo
		if err := rows.Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}
	return todos, rows.Err()
}

func (s *PostgresStore) Get(ctx context.Context, id int64) (model.Todo, error) {
	var t model.Todo
	err := s.Pool.QueryRow(ctx, `SELECT id, title, done, created_at, updated_at FROM todos WHERE id = $1`, id).
		Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Todo{}, ErrNotFound
	}
	return t, err
}

func (s *PostgresStore) Create(ctx context.Context, title string) (model.Todo, error) {
	var t model.Todo
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO todos (title) VALUES ($1) RETURNING id, title, done, created_at, updated_at`, title).
		Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (s *PostgresStore) Update(ctx context.Context, id int64, title string, done bool) (model.Todo, error) {
	var t model.Todo
	err := s.Pool.QueryRow(ctx,
		`UPDATE todos SET title = $2, done = $3, updated_at = now() WHERE id = $1
		 RETURNING id, title, done, created_at, updated_at`, id, title, done).
		Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Todo{}, ErrNotFound
	}
	return t, err
}

func (s *PostgresStore) Delete(ctx context.Context, id int64) error {
	ct, err := s.Pool.Exec(ctx, `DELETE FROM todos WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
