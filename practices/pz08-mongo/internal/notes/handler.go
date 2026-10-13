package notes

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const operationTimeout = 5 * time.Second

type Handler struct{ store Store }

func NewHandler(store Store) *Handler { return &Handler{store: store} }

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/api/v1/notes", h.create)
	r.Get("/api/v1/notes", h.list)
	r.Get("/api/v1/notes/{id}", h.get)
	r.Patch("/api/v1/notes/{id}", h.update)
	r.Delete("/api/v1/notes/{id}", h.delete)
	return r
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		badRequest(w, "invalid JSON body")
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		badRequest(w, "title is required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), operationTimeout)
	defer cancel()
	note, err := h.store.Create(ctx, input.Title, input.Content)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, note)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	limit, err := parseNonNegative(r, "limit", 20)
	if err != nil || limit == 0 || limit > 100 {
		badRequest(w, "limit must be between 1 and 100")
		return
	}
	skip, err := parseNonNegative(r, "skip", 0)
	if err != nil {
		badRequest(w, "skip must be a non-negative integer")
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 200 {
		badRequest(w, "q is too long")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), operationTimeout)
	defer cancel()
	items, err := h.store.List(ctx, q, limit, skip)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "skip": skip})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), operationTimeout)
	defer cancel()
	note, err := h.store.Get(ctx, id)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, note)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var raw map[string]json.RawMessage
	if err := decodeJSON(w, r, &raw); err != nil || len(raw) == 0 {
		badRequest(w, "invalid JSON body")
		return
	}
	var patch Update
	for field, value := range raw {
		if field != "title" && field != "content" {
			badRequest(w, "unknown field")
			return
		}
		if string(value) == "null" {
			badRequest(w, "null is not allowed")
			return
		}
		var text string
		if err := json.Unmarshal(value, &text); err != nil {
			badRequest(w, "fields must be strings")
			return
		}
		if field == "title" {
			text = strings.TrimSpace(text)
			if text == "" {
				badRequest(w, "title cannot be empty")
				return
			}
			patch.Title = &text
		} else {
			patch.Content = &text
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), operationTimeout)
	defer cancel()
	note, err := h.store.Update(ctx, id, patch)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, note)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), operationTimeout)
	defer cancel()
	if err := h.store.Delete(ctx, id); err != nil {
		storeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(w http.ResponseWriter, r *http.Request) (primitive.ObjectID, bool) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		badRequest(w, "invalid note id")
		return primitive.NilObjectID, false
	}
	return id, true
}

func parseNonNegative(r *http.Request, name string, defaultValue int64) (int64, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return defaultValue, nil
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n < 0 {
		return 0, errors.New("invalid non-negative integer")
	}
	return n, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("multiple JSON values")
	}
	return nil
}

func badRequest(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": message})
}

func storeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "note not found"})
	case errors.Is(err, ErrDuplicateTitle):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "title already exists"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database operation failed"})
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
