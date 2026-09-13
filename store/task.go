package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/homayounmohseni/todomgr/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type TaskStore interface {
	List(ctx context.Context, f ListFilter) ([]model.Task, error)
	Get(ctx context.Context, id int64) (model.Task, error)
	Create(ctx context.Context, title, assignee string) (model.Task, error)
	Update(ctx context.Context, id int64, title string, status bool, assignee string) (model.Task, error)
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

type ListFilter struct {
	Limit    int
	Offset   int
	Status   *bool
	Assignee string
}

func (s *PostgresStore) List(ctx context.Context, f ListFilter) ([]model.Task, error) {
	query := `SELECT id, title, status, assignee, created_at, updated_at FROM tasks`
	args := []any{}
	conds := []string{}
	if f.Status != nil {
		args = append(args, *f.Status)
		conds = append(conds, fmt.Sprintf("status = $%d", len(args)))
	}
	if f.Assignee != "" {
		args = append(args, f.Assignee)
		conds = append(conds, fmt.Sprintf("assignee = $%d", len(args)))
	}
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	args = append(args, f.Limit, f.Offset)
	query += fmt.Sprintf(" ORDER BY id ASC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tasks := []model.Task{}
	for rows.Next() {
		var t model.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Status, &t.Assignee, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (s *PostgresStore) Get(ctx context.Context, id int64) (model.Task, error) {
	var t model.Task
	err := s.Pool.QueryRow(ctx, `SELECT id, title, status, assignee, created_at, updated_at FROM tasks WHERE id = $1`, id).
		Scan(&t.ID, &t.Title, &t.Status, &t.Assignee, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Task{}, ErrNotFound
	}
	return t, err
}

func (s *PostgresStore) Create(ctx context.Context, title, assignee string) (model.Task, error) {
	var t model.Task
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO tasks (title, assignee) VALUES ($1, $2) RETURNING id, title, status, assignee, created_at, updated_at`, title, assignee).
		Scan(&t.ID, &t.Title, &t.Status, &t.Assignee, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (s *PostgresStore) Update(ctx context.Context, id int64, title string, status bool, assignee string) (model.Task, error) {
	var t model.Task
	err := s.Pool.QueryRow(ctx,
		`UPDATE tasks SET title = $2, status = $3, assignee = $4, updated_at = now() WHERE id = $1
		 RETURNING id, title, status, assignee, created_at, updated_at`, id, title, status, assignee).
		Scan(&t.ID, &t.Title, &t.Status, &t.Assignee, &t.CreatedAt, &t.UpdatedAt)
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
