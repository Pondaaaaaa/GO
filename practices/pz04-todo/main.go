package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Pondaaaaaa/GO/practices/pz04-todo/internal/task"
	"github.com/Pondaaaaaa/GO/practices/pz04-todo/pkg/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	repo := task.NewRepository()
	handler := task.NewHandler(repo)
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.Recoverer)
	router.Use(middleware.Logger)
	router.Use(middleware.CORS)
	router.Get("/health", handler.Health)
	router.Get("/api/tasks", handler.List)
	router.Post("/api/tasks", handler.Create)
	router.Get("/api/tasks/{id}", handler.Get)
	router.Put("/api/tasks/{id}", handler.Update)
	router.Delete("/api/tasks/{id}", handler.Delete)

	addr := ":" + port
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
