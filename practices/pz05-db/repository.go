package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Task struct {
	ID        int64
	Title     string
	Done      bool
	CreatedAt time.Time
}

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) CreateTask(ctx context.Context, title string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO tasks (title) VALUES ($1) RETURNING id`, title,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert task: %w", err)
	}
	return id, nil
}

func (r *Repository) ListTasks(ctx context.Context) ([]Task, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, title, done, created_at FROM tasks ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("select tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		var item Task
		if err := rows.Scan(&item.ID, &item.Title, &item.Done, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	return tasks, nil
}

func (r *Repository) ListDone(ctx context.Context, done bool) ([]Task, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, title, done, created_at FROM tasks WHERE done = $1 ORDER BY id`, done,
	)
	if err != nil {
		return nil, fmt.Errorf("select tasks by done: %w", err)
	}
	defer rows.Close()
	var tasks []Task
	for rows.Next() {
		var item Task
		if err := rows.Scan(&item.ID, &item.Title, &item.Done, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan filtered task: %w", err)
		}
		tasks = append(tasks, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate filtered tasks: %w", err)
	}
	return tasks, nil
}

func (r *Repository) FindByID(ctx context.Context, id int) (Task, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var item Task
	err := r.db.QueryRowContext(ctx,
		`SELECT id, title, done, created_at FROM tasks WHERE id = $1`, id,
	).Scan(&item.ID, &item.Title, &item.Done, &item.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, sql.ErrNoRows
		}
		return Task{}, fmt.Errorf("find task by ID: %w", err)
	}
	return item, nil
}

func (r *Repository) FindByTitle(ctx context.Context, title string) (Task, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var item Task
	err := r.db.QueryRowContext(ctx,
		`SELECT id, title, done, created_at FROM tasks WHERE title = $1 ORDER BY id LIMIT 1`, title,
	).Scan(&item.ID, &item.Title, &item.Done, &item.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, sql.ErrNoRows
		}
		return Task{}, fmt.Errorf("find task by title: %w", err)
	}
	return item, nil
}

func (r *Repository) CreateMany(ctx context.Context, titles []string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for i, title := range titles {
		if strings.TrimSpace(title) == "" {
			return fmt.Errorf("title at index %d is empty", i)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO tasks (title) VALUES ($1)`, title); err != nil {
			return fmt.Errorf("insert task at index %d: %w", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
