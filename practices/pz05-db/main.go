package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

const operationTimeout = 5 * time.Second

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		log.Fatal("DATABASE_URL is required; set it in this PowerShell session using your local PostgreSQL settings")
	}

	ctx := context.Background()
	db, err := openDB(ctx, dsn)
	if err != nil {
		log.Fatal("open PostgreSQL failed; check the local PostgreSQL settings and connection")
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("close PostgreSQL: %v", err)
		}
	}()

	repo := NewRepository(db)
	stamp := time.Now().UTC().Format(time.RFC3339Nano)
	titles := []string{"PZ05 single " + stamp, "PZ05 transaction A " + stamp, "PZ05 transaction B " + stamp}
	var firstID int64

	for _, title := range titles[:1] {
		opCtx, cancel := context.WithTimeout(ctx, operationTimeout)
		id, err := repo.CreateTask(opCtx, title)
		cancel()
		if err != nil {
			log.Fatalf("insert task: %v", err)
		}
		firstID = id
		fmt.Printf("inserted task id=%d title=%q\n", id, title)
	}

	opCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	if err := repo.CreateMany(opCtx, titles[1:]); err != nil {
		cancel()
		log.Fatalf("insert tasks transaction: %v", err)
	}
	cancel()
	fmt.Printf("inserted %d tasks in transaction\n", len(titles[1:]))

	opCtx, cancel = context.WithTimeout(ctx, operationTimeout)
	tasks, err := repo.ListTasks(opCtx)
	cancel()
	if err != nil {
		log.Fatalf("list tasks: %v", err)
	}
	for _, item := range tasks {
		fmt.Printf("task id=%d title=%q done=%t created_at=%s\n", item.ID, item.Title, item.Done, item.CreatedAt.Format(time.RFC3339Nano))
	}

	for _, done := range []bool{false, true} {
		opCtx, cancel = context.WithTimeout(ctx, operationTimeout)
		matches, err := repo.ListDone(opCtx, done)
		cancel()
		if err != nil {
			log.Fatalf("filter tasks done=%t: %v", done, err)
		}
		fmt.Printf("ListDone(%t): %d task(s)\n", done, len(matches))
	}

	opCtx, cancel = context.WithTimeout(ctx, operationTimeout)
	found, err := repo.FindByID(opCtx, int(firstID))
	cancel()
	if err != nil {
		log.Fatalf("find task: %v", err)
	}
	fmt.Printf("FindByID: id=%d title=%q done=%t\n", found.ID, found.Title, found.Done)
	opCtx, cancel = context.WithTimeout(ctx, operationTimeout)
	_, missingErr := repo.FindByID(opCtx, -1)
	cancel()
	if !errors.Is(missingErr, sql.ErrNoRows) {
		log.Fatalf("FindByID missing-row check returned %v, want sql.ErrNoRows", missingErr)
	}
	fmt.Println("FindByID(-1): sql.ErrNoRows (expected)")

	pool := db.Stats()
	fmt.Printf("db.Stats: OpenConnections=%d InUse=%d Idle=%d WaitCount=%d WaitDuration=%s MaxOpenConnections=%d\n",
		pool.OpenConnections, pool.InUse, pool.Idle, pool.WaitCount, pool.WaitDuration, pool.MaxOpenConnections)

	// An empty title after a valid title forces CreateMany to roll back its
	// first insert; check that the probe title is absent after the error.
	rollbackTitle := "PZ05 rollback probe " + stamp
	opCtx, cancel = context.WithTimeout(ctx, operationTimeout)
	rollbackErr := repo.CreateMany(opCtx, []string{rollbackTitle, "   "})
	cancel()
	if rollbackErr == nil {
		log.Fatal("expected CreateMany to reject a blank title")
	}
	opCtx, cancel = context.WithTimeout(ctx, operationTimeout)
	_, findErr := repo.FindByTitle(opCtx, rollbackTitle)
	cancel()
	if !errors.Is(findErr, sql.ErrNoRows) {
		log.Fatalf("transaction rollback check failed: title lookup error=%v", findErr)
	}
	fmt.Printf("CreateMany rollback check: %v; probe row absent\n", rollbackErr)
}
