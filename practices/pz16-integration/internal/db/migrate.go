package db

import (
	"context"
	"database/sql"
	_ "embed"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrate.sql
var migration string

// Callers must keep returned driver errors out of public diagnostics.
func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	d, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	d.SetMaxOpenConns(4)
	d.SetMaxIdleConns(2)
	d.SetConnMaxLifetime(5 * time.Minute)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err = d.PingContext(ctx); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}

func Migrate(ctx context.Context, d *sql.DB) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := d.ExecContext(ctx, migration)
	return err
}
