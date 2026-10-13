package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Pondaaaaaa/GO/practices/pz16-integration/internal/models"
)

type Notes struct{ DB *sql.DB }

const columns = "id, title, content, created_at, updated_at"

func scan(row interface{ Scan(...any) error }) (models.Note, error) {
	var n models.Note
	err := row.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		err = models.ErrNotFound
	}
	return n, err
}

func (r Notes) Create(ctx context.Context, in models.Create) (models.Note, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return scan(r.DB.QueryRowContext(ctx, "INSERT INTO notes(title, content) VALUES($1,$2) RETURNING "+columns, in.Title, in.Content))
}
func (r Notes) Get(ctx context.Context, id int64) (models.Note, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return scan(r.DB.QueryRowContext(ctx, "SELECT "+columns+" FROM notes WHERE id=$1", id))
}
func (r Notes) List(ctx context.Context) ([]models.Note, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := r.DB.QueryContext(ctx, "SELECT "+columns+" FROM notes ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	notes := make([]models.Note, 0)
	for rows.Next() {
		n, err := scan(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}
func (r Notes) Update(ctx context.Context, id int64, patch models.Patch) (models.Note, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return scan(r.DB.QueryRowContext(ctx, "UPDATE notes SET title=COALESCE($2,title), content=COALESCE($3,content), updated_at=clock_timestamp() WHERE id=$1 RETURNING "+columns, id, patch.Title, patch.Content))
}
func (r Notes) Delete(ctx context.Context, id int64) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	result, err := r.DB.ExecContext(ctx, "DELETE FROM notes WHERE id=$1", id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return models.ErrNotFound
	}
	return nil
}
