package httpx

import (
	"net/http"

	"github.com/Pondaaaaaa/GO/practices/pz12-notes/internal/http/handlers"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func NewRouter(h *handlers.Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/docs/*", httpSwagger.Handler(httpSwagger.URL("/docs/doc.json")))
	r.Get("/api/v1/notes", h.ListNotes)
	r.Post("/api/v1/notes", h.CreateNote)
	r.Get("/api/v1/notes/{id}", h.GetNote)
	r.Patch("/api/v1/notes/{id}", h.UpdateNote)
	r.Delete("/api/v1/notes/{id}", h.DeleteNote)
	return r
}
