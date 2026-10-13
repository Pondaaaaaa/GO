package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Pondaaaaaa/GO/practices/pz08-mongo/internal/db"
	"github.com/Pondaaaaaa/GO/practices/pz08-mongo/internal/notes"
	"github.com/go-chi/chi/v5"
)

func main() {
	uri, database, addr := os.Getenv("MONGO_URI"), os.Getenv("MONGO_DB"), os.Getenv("HTTP_ADDR")
	if uri == "" || database == "" || addr == "" {
		log.Fatal("MONGO_URI, MONGO_DB and HTTP_ADDR are required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	client, collection, err := db.Connect(ctx, uri, database)
	if err != nil {
		log.Fatal("MongoDB startup failed")
	}
	defer func() {
		if err := db.Disconnect(client); err != nil {
			log.Print("MongoDB disconnect failed")
		}
	}()
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		if err := db.Ping(req.Context(), client); err != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("{\"status\":\"unavailable\"}\n"))
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte("{\"status\":\"ok\"}\n"))
	})
	r.Mount("/", notes.NewHandler(notes.NewRepo(collection)).Routes())
	server := &http.Server{Addr: addr, Handler: r, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	log.Printf("MongoDB notes API listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("HTTP server stopped: %v", err)
	}
}
