package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"

	"github.com/Pondaaaaaa/GO/practices/pz16-integration/internal/models"
	"github.com/Pondaaaaaa/GO/practices/pz16-integration/internal/service"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ Service service.Service }

func New(s service.Service) http.Handler {
	h := Handler{Service: s}
	r := chi.NewRouter()
	r.Post("/api/v1/notes", h.create)
	r.Get("/api/v1/notes", h.list)
	r.Get("/api/v1/notes/{id}", h.get)
	r.Patch("/api/v1/notes/{id}", h.update)
	r.Delete("/api/v1/notes/{id}", h.delete)
	return r
}
func write(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func failure(w http.ResponseWriter, err error) {
	status, message := 500, "internal server error"
	if errors.Is(err, models.ErrNotFound) {
		status, message = 404, models.ErrNotFound.Error()
	}
	if errors.Is(err, models.ErrInvalid) {
		status, message = 400, models.ErrInvalid.Error()
	}
	// Database diagnostics can contain credentials; expose only stable messages.
	write(w, status, map[string]string{"error": message})
}
func decode(w http.ResponseWriter, r *http.Request, out any) error {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return models.ErrInvalid
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return models.ErrInvalid
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return models.ErrInvalid
	}
	return nil
}
func id(r *http.Request) (int64, error) {
	v, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || v <= 0 {
		return 0, models.ErrInvalid
	}
	return v, nil
}
func (h Handler) create(w http.ResponseWriter, r *http.Request) {
	var in models.Create
	if err := decode(w, r, &in); err != nil {
		failure(w, err)
		return
	}
	n, err := h.Service.Create(r.Context(), in)
	if err != nil {
		failure(w, err)
		return
	}
	w.Header().Set("Location", "/api/v1/notes/"+strconv.FormatInt(n.ID, 10))
	write(w, 201, n)
}
func (h Handler) list(w http.ResponseWriter, r *http.Request) {
	n, err := h.Service.List(r.Context())
	if err != nil {
		failure(w, err)
		return
	}
	write(w, 200, n)
}
func (h Handler) get(w http.ResponseWriter, r *http.Request) {
	v, err := id(r)
	if err != nil {
		failure(w, err)
		return
	}
	n, err := h.Service.Get(r.Context(), v)
	if err != nil {
		failure(w, err)
		return
	}
	write(w, 200, n)
}
func (h Handler) update(w http.ResponseWriter, r *http.Request) {
	v, err := id(r)
	if err != nil {
		failure(w, err)
		return
	}
	var patch models.Patch
	if err = decode(w, r, &patch); err != nil {
		failure(w, err)
		return
	}
	n, err := h.Service.Update(r.Context(), v, patch)
	if err != nil {
		failure(w, err)
		return
	}
	write(w, 200, n)
}
func (h Handler) delete(w http.ResponseWriter, r *http.Request) {
	v, err := id(r)
	if err != nil {
		failure(w, err)
		return
	}
	if err = h.Service.Delete(r.Context(), v); err != nil {
		failure(w, err)
		return
	}
	w.WriteHeader(204)
}
