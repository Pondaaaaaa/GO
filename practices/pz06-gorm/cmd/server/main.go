package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	database "github.com/Pondaaaaaa/GO/practices/pz06-gorm/internal/db"
	httpapi "github.com/Pondaaaaaa/GO/practices/pz06-gorm/internal/http"
)

func main() {
	dsn := os.Getenv("DB_DSN")
	if strings.TrimSpace(dsn) == "" {
		log.Fatal("DB_DSN is required; run run-local.ps1 or configure it in the current process environment")
	}

	rootCtx := context.Background()
	db, err := database.Open(rootCtx, dsn)
	if err != nil {
		log.Fatal("could not connect to PostgreSQL; check local connection settings")
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("could not initialize SQL connection pool")
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Printf("close PostgreSQL pool: %v", err)
		}
	}()

	migrateCtx, cancelMigrate := context.WithTimeout(rootCtx, 15*time.Second)
	if err := database.Migrate(migrateCtx, db); err != nil {
		cancelMigrate()
		log.Fatal("could not migrate the practice schema")
	}
	cancelMigrate()

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr:              "127.0.0.1:" + port,
		Handler:           httpapi.NewRouter(db),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(rootCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("server listening at http://%s/", server.Addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server failed: %v", err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown: %v", err)
		}
	}
}
