package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/Pondaaaaaa/GO/practices/pz12-notes/internal/core"
	"github.com/Pondaaaaaa/GO/practices/pz12-notes/internal/core/service"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ Notes *service.NoteService }

func jsonResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func badRequest(w http.ResponseWriter) {
	jsonResponse(w, http.StatusBadRequest, core.ErrorResponse{Error: "invalid request"})
}

func decode(r *http.Request, value any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errors.New("content type must be JSON")
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 || !strings.HasPrefix(strings.TrimSpace(string(body)), "{") {
		return errors.New("body must be a JSON object up to 1 MiB")
	}
	d := json.NewDecoder(bytes.NewReader(body))
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
		jsonResponse(w, http.StatusNotFound, core.ErrorResponse{Error: "note not found"})
		return
	}
	jsonResponse(w, http.StatusInternalServerError, core.ErrorResponse{Error: "internal server error"})
}

// CreateNote creates a note.
// @Summary Create a note
// @Description JSON object up to 1 MiB. Title is trimmed and must not be blank. Unknown fields are rejected.
// @Tags notes
// @Accept json
// @Produce json
// @Param note body core.NoteCreate true "New note"
// @Success 201 {object} core.Note
// @Header 201 {string} Location "Relative URL of the created note"
// @Failure 400 {object} core.ErrorResponse "Invalid JSON, content type or title"
// @Failure 500 {object} core.ErrorResponse "Repository error"
// @Router /notes [post]
func (h *Handler) CreateNote(w http.ResponseWriter, r *http.Request) {
	var input core.NoteCreate
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

// ListNotes lists all notes.
// @Summary List notes
// @Description All notes ordered by ascending ID; an empty repository returns []. No filtering or pagination.
// @Tags notes
// @Produce json
// @Success 200 {array} core.Note
// @Failure 500 {object} core.ErrorResponse "Repository error"
// @Router /notes [get]
func (h *Handler) ListNotes(w http.ResponseWriter, _ *http.Request) {
	notes, err := h.Notes.List()
	if err != nil {
		serverError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, notes)
}

// GetNote reads a note.
// @Summary Get a note
// @Tags notes
// @Produce json
// @Param id path int true "Positive note ID" minimum(1)
// @Success 200 {object} core.Note
// @Failure 400 {object} core.ErrorResponse "Invalid ID"
// @Failure 404 {object} core.ErrorResponse "Note not found"
// @Failure 500 {object} core.ErrorResponse "Repository error"
// @Router /notes/{id} [get]
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

// UpdateNote partially updates a note.
// @Summary Update a note
// @Description At least one non-null title or content is required. Title must not be blank; content may be empty. Unknown fields are rejected. JSON object up to 1 MiB.
// @Tags notes
// @Accept json
// @Produce json
// @Param id path int true "Positive note ID" minimum(1)
// @Param note body core.NoteUpdate true "Fields to update"
// @Success 200 {object} core.Note
// @Failure 400 {object} core.ErrorResponse "Invalid ID or input"
// @Failure 404 {object} core.ErrorResponse "Note not found"
// @Failure 500 {object} core.ErrorResponse "Repository error"
// @Router /notes/{id} [patch]
func (h *Handler) UpdateNote(w http.ResponseWriter, r *http.Request) {
	noteID, err := id(r)
	if err != nil {
		badRequest(w)
		return
	}
	var patch core.NoteUpdate
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

// DeleteNote deletes a note.
// @Summary Delete a note
// @Tags notes
// @Produce json
// @Param id path int true "Positive note ID" minimum(1)
// @Success 204 "Deleted without response body"
// @Failure 400 {object} core.ErrorResponse "Invalid ID"
// @Failure 404 {object} core.ErrorResponse "Note not found"
// @Failure 500 {object} core.ErrorResponse "Repository error"
// @Router /notes/{id} [delete]
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
