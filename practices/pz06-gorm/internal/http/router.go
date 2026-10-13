package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB) http.Handler {
	h := &handlers{db: db}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Get("/health", h.health)
	r.Post("/users", h.createUser)
	r.Get("/users/{id}", h.getUser)
	r.Post("/notes", h.createNote)
	r.Get("/notes/{id}", h.getNote)
	return r
}
