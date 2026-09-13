package store

import (
	"context"
	"errors"

	"github.com/homayounmohseni/todomgr/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type TaskStore interface {
	List(ctx context.Context) ([]model.Task, error)
	Get(ctx context.Context, id int64) (model.Task, error)
	Create(ctx context.Context, title string) (model.Task, error)
	Update(ctx context.Context, id int64, title string, done bool) (model.Task, error)
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

func (s *PostgresStore) List(ctx context.Context) ([]model.Task, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, title, done, created_at, updated_at FROM tasks ORDER BY id ASC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tasks := []model.Task{}
	for rows.Next() {
		var t model.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (s *PostgresStore) Get(ctx context.Context, id int64) (model.Task, error) {
	var t model.Task
	err := s.Pool.QueryRow(ctx, `SELECT id, title, done, created_at, updated_at FROM tasks WHERE id = $1`, id).
		Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Task{}, ErrNotFound
	}
	return t, err
}

func (s *PostgresStore) Create(ctx context.Context, title string) (model.Task, error) {
	var t model.Task
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO tasks (title) VALUES ($1) RETURNING id, title, done, created_at, updated_at`, title).
		Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (s *PostgresStore) Update(ctx context.Context, id int64, title string, done bool) (model.Task, error) {
	var t model.Task
	err := s.Pool.QueryRow(ctx,
		`UPDATE tasks SET title = $2, done = $3, updated_at = now() WHERE id = $1
		 RETURNING id, title, done, created_at, updated_at`, id, title, done).
		Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Task{}, ErrNotFound
	}
	return t, err
}

func (s *PostgresStore) Delete(ctx context.Context, id int64) error {
	ct, err := s.Pool.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
