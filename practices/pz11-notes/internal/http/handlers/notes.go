package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Pondaaaaaa/GO/practices/pz11-notes/internal/core"
	"github.com/Pondaaaaaa/GO/practices/pz11-notes/internal/core/service"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ Notes *service.NoteService }

func jsonResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func badRequest(w http.ResponseWriter) {
	jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
}

func decode(r *http.Request, value any) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return errors.New("content type must be JSON")
	}
	d := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}

func id(r *http.Request) (int64, error) {
	v, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || v <= 0 {
		return 0, errors.New("invalid id")
	}
	return v, nil
}

func serverError(w http.ResponseWriter, err error) {
	if errors.Is(err, core.ErrNotFound) {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "note not found"})
		return
	}
	jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}

func (h *Handler) CreateNote(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if decode(r, &input) != nil {
		badRequest(w)
		return
	}
	n, err := h.Notes.Create(input.Title, input.Content)
	if errors.Is(err, service.ErrInvalid) {
		badRequest(w)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	w.Header().Set("Location", "/api/v1/notes/"+strconv.FormatInt(n.ID, 10))
	jsonResponse(w, http.StatusCreated, n)
}

func (h *Handler) ListNotes(w http.ResponseWriter, _ *http.Request) {
	notes, err := h.Notes.List()
	if err != nil {
		serverError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, notes)
}

func (h *Handler) GetNote(w http.ResponseWriter, r *http.Request) {
	noteID, err := id(r)
	if err != nil {
		badRequest(w)
		return
	}
	n, err := h.Notes.Get(noteID)
	if err != nil {
		serverError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, n)
}

func (h *Handler) UpdateNote(w http.ResponseWriter, r *http.Request) {
	noteID, err := id(r)
	if err != nil {
		badRequest(w)
		return
	}
	var patch core.Patch
	if decode(r, &patch) != nil {
		badRequest(w)
		return
	}
	n, err := h.Notes.Update(noteID, patch)
	if errors.Is(err, service.ErrInvalid) {
		badRequest(w)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, n)
}

func (h *Handler) DeleteNote(w http.ResponseWriter, r *http.Request) {
	noteID, err := id(r)
	if err != nil {
		badRequest(w)
		return
	}
	if err := h.Notes.Delete(noteID); err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
