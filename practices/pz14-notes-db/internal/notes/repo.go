package notes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var ErrInvalid = errors.New("invalid note")
var schemaPattern = regexp.MustCompile(`^pz14_[a-f0-9]{32}_(base|opt)$`)

type Note struct {
	ID        int64      `json:"id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}
type Input struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}
type Patch struct {
	Title   *string `json:"title"`
	Content *string `json:"content"`
}
type Repo struct {
	DB           *sql.DB
	Table, Audit string
	Insert, Find *sql.Stmt
	Optimized    bool
}

func Open(ctx context.Context, max int) (*sql.DB, error) {
	// libpq-style PG* environment variables are parsed by pgx, not logged.
	db, err := sql.Open("pgx", "")
	if err != nil {
		return nil, errors.New("database configuration failed")
	}
	db.SetMaxOpenConns(max)
	db.SetMaxIdleConns(max)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, errors.New("database connection failed; check local PG settings")
	}
	return db, nil
}

func New(ctx context.Context, db *sql.DB, schema string, optimized bool) (*Repo, error) {
	if !schemaPattern.MatchString(schema) {
		return nil, errors.New("invalid owned schema")
	}
	r := &Repo{DB: db, Table: schema + ".notes", Audit: schema + ".note_audit", Optimized: optimized}
	var err error
	r.Insert, err = db.PrepareContext(ctx, "INSERT INTO "+r.Table+" (title,content) VALUES ($1,$2) RETURNING id,title,content,created_at,updated_at")
	if err != nil {
		return nil, errors.New("prepare insert failed")
	}
	r.Find, err = db.PrepareContext(ctx, "SELECT id,title,content,created_at,updated_at FROM "+r.Table+" WHERE id=$1")
	if err != nil {
		r.Insert.Close()
		return nil, errors.New("prepare find failed")
	}
	return r, nil
}
func (r *Repo) Close() { r.Insert.Close(); r.Find.Close() }

type scanner interface{ Scan(...any) error }

func scan(s scanner) (n Note, err error) {
	err = s.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)
	return
}
func rowsList(rows *sql.Rows) ([]Note, error) {
	defer rows.Close()
	out := make([]Note, 0)
	for rows.Next() {
		n, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
func (r *Repo) Get(ctx context.Context, id int64) (Note, error) {
	return scan(r.Find.QueryRowContext(ctx, id))
}

// CreateMany commits notes and their audit entries atomically. Validation within
// the transaction deliberately permits exercising rollback after an insert.
func (r *Repo) CreateMany(ctx context.Context, inputs []Input) ([]Note, error) {
	if len(inputs) < 1 || len(inputs) > 20 {
		return nil, ErrInvalid
	}
	tx, err := r.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	stmt := tx.StmtContext(ctx, r.Insert)
	defer stmt.Close()
	out := make([]Note, 0, len(inputs))
	for _, in := range inputs {
		in.Title = strings.TrimSpace(in.Title)
		if in.Title == "" {
			return nil, ErrInvalid
		}
		n, err := scan(stmt.QueryRowContext(ctx, in.Title, in.Content))
		if err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO "+r.Audit+"(note_id,action) VALUES ($1,'create')", n.ID); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}
func (r *Repo) Update(ctx context.Context, id int64, p Patch) (Note, error) {
	if p.Title == nil && p.Content == nil {
		return Note{}, ErrInvalid
	}
	if p.Title != nil {
		v := strings.TrimSpace(*p.Title)
		if v == "" {
			return Note{}, ErrInvalid
		}
		p.Title = &v
	}
	return scan(r.DB.QueryRowContext(ctx, "UPDATE "+r.Table+" SET title=COALESCE($1,title),content=COALESCE($2,content),updated_at=now() WHERE id=$3 RETURNING id,title,content,created_at,updated_at", p.Title, p.Content, id))
}
func (r *Repo) Delete(ctx context.Context, id int64) error {
	res, err := r.DB.ExecContext(ctx, "DELETE FROM "+r.Table+" WHERE id=$1", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return err
}
func (r *Repo) List(ctx context.Context, limit, offset int, cursorTime *time.Time, cursorID int64) ([]Note, error) {
	query := "SELECT id,title,content,created_at,updated_at FROM " + r.Table
	var args []any
	if cursorTime != nil && r.Optimized {
		query += " WHERE (created_at,id)<($1,$2) ORDER BY created_at DESC,id DESC LIMIT $3"
		args = []any{*cursorTime, cursorID, limit}
	} else {
		query += " ORDER BY created_at DESC,id DESC OFFSET $1 LIMIT $2"
		args = []any{offset, limit}
	}
	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return rowsList(rows)
}
func (r *Repo) Batch(ctx context.Context, ids []int64) ([]Note, error) {
	if !r.Optimized {
		out := make([]Note, 0, len(ids))
		for _, id := range ids {
			n, err := r.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			out = append(out, n)
		}
		return out, nil
	}
	rows, err := r.DB.QueryContext(ctx, "SELECT id,title,content,created_at,updated_at FROM "+r.Table+" WHERE id=ANY($1::bigint[]) ORDER BY array_position($1::bigint[],id)", ids)
	if err != nil {
		return nil, err
	}
	out, err := rowsList(rows)
	if err == nil && len(out) != len(ids) {
		return nil, sql.ErrNoRows
	}
	return out, err
}
func Schema(run, kind string) string { return fmt.Sprintf("pz14_%s_%s", run, kind) }
