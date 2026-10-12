package api

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/Pondaaaaaa/GO/practices/pz03-http/internal/storage"
)

type handlers struct {
	store *storage.MemoryStore
}

type createTaskRequest struct {
	Title string `json:"title"`
}

func NewHandler(store *storage.MemoryStore) http.Handler {
	h := &handlers{store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/tasks", h.tasks)
	mux.HandleFunc("/tasks/", h.task)
	return Logging(mux)
}

func (h *handlers) health(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) tasks(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		h.listTasks(w, req)
	case http.MethodPost:
		h.createTask(w, req)
	default:
		methodNotAllowed(w, http.MethodGet+", "+http.MethodPost)
	}
}

func (h *handlers) listTasks(w http.ResponseWriter, req *http.Request) {
	tasks := h.store.List()
	query := strings.ToLower(strings.TrimSpace(req.URL.Query().Get("q")))
	if query != "" {
		filtered := make([]storage.Task, 0, len(tasks))
		for _, task := range tasks {
			if strings.Contains(strings.ToLower(task.Title), query) {
				filtered = append(filtered, task)
			}
		}
		tasks = filtered
	}
	JSON(w, http.StatusOK, tasks)
}

func (h *handlers) createTask(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	mediaType, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		BadRequest(w, "Content-Type must be application/json")
		return
	}

	var input createTaskRequest
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&input); err != nil {
		BadRequest(w, "invalid JSON body")
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		BadRequest(w, "body must contain one JSON value")
		return
	}

	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		BadRequest(w, "title is required")
		return
	}
	JSON(w, http.StatusCreated, h.store.Create(input.Title))
}

func (h *handlers) task(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	idText := strings.TrimPrefix(req.URL.Path, "/tasks/")
	if idText == req.URL.Path || idText == "" || strings.Contains(idText, "/") {
		NotFound(w, "task not found")
		return
	}
	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || id <= 0 {
		BadRequest(w, "invalid task id")
		return
	}

	task, err := h.store.Get(id)
	if errors.Is(err, storage.ErrNotFound) {
		NotFound(w, "task not found")
		return
	}
	if err != nil {
		Internal(w, "could not get task")
		return
	}
	JSON(w, http.StatusOK, task)
}

func methodNotAllowed(w http.ResponseWriter, allowed string) {
	w.Header().Set("Allow", allowed)
	JSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
}
