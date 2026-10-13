package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"github.com/Pondaaaaaa/GO/practices/pz06-gorm/internal/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type handlers struct {
	db *gorm.DB
}

type apiError struct {
	status  int
	message string
}

func (e *apiError) Error() string { return e.message }

type userInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type noteInput struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	UserID  uint     `json:"userId"`
	Tags    []string `json:"tags"`
}

type userDTO struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type tagDTO struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type noteSummaryDTO struct {
	ID      uint   `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	UserID  uint   `json:"userId"`
}

type userDetailDTO struct {
	ID    uint             `json:"id"`
	Name  string           `json:"name"`
	Email string           `json:"email"`
	Notes []noteSummaryDTO `json:"notes"`
}

type noteDTO struct {
	ID        uint     `json:"id"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	UserID    uint     `json:"userId"`
	User      userDTO  `json:"user"`
	Tags      []tagDTO `json:"tags"`
	CreatedAt string   `json:"createdAt"`
}

func (h *handlers) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) createUser(w http.ResponseWriter, r *http.Request) {
	var input userInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeFailure(w, err)
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)
	if input.Name == "" || len(input.Name) > 120 {
		writeFailure(w, &apiError{status: http.StatusBadRequest, message: "name is required and must be at most 120 characters"})
		return
	}
	parsedEmail, err := mail.ParseAddress(input.Email)
	if err != nil || parsedEmail.Address != input.Email || len(input.Email) > 254 {
		writeFailure(w, &apiError{status: http.StatusBadRequest, message: "email must be a valid address"})
		return
	}

	user := models.User{Name: input.Name, Email: input.Email}
	if err := h.db.WithContext(r.Context()).Create(&user).Error; err != nil {
		writeFailure(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toUserDTO(user))
}

func (h *handlers) getUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeFailure(w, err)
		return
	}
	var user models.User
	if err := h.db.WithContext(r.Context()).Preload("Notes").First(&user, id).Error; err != nil {
		writeFailure(w, err)
		return
	}
	response := userDetailDTO{ID: user.ID, Name: user.Name, Email: user.Email, Notes: make([]noteSummaryDTO, 0, len(user.Notes))}
	for _, note := range user.Notes {
		response.Notes = append(response.Notes, noteSummaryDTO{ID: note.ID, Title: note.Title, Content: note.Content, UserID: note.UserID})
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *handlers) createNote(w http.ResponseWriter, r *http.Request) {
	var input noteInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeFailure(w, err)
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	if input.Title == "" || input.Content == "" || input.UserID == 0 {
		writeFailure(w, &apiError{status: http.StatusBadRequest, message: "title, content and a positive userId are required"})
		return
	}
	if len(input.Title) > 200 || len(input.Content) > 10000 || len(input.Tags) > 20 {
		writeFailure(w, &apiError{status: http.StatusBadRequest, message: "title, content or tags exceed the allowed size"})
		return
	}

	tagNames := make([]string, 0, len(input.Tags))
	seen := make(map[string]struct{}, len(input.Tags))
	for _, rawName := range input.Tags {
		name := strings.TrimSpace(rawName)
		if name == "" || len(name) > 80 {
			writeFailure(w, &apiError{status: http.StatusBadRequest, message: "tag names must be non-empty and at most 80 characters"})
			return
		}
		if _, exists := seen[name]; !exists {
			seen[name] = struct{}{}
			tagNames = append(tagNames, name)
		}
	}

	var created models.Note
	err := h.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, input.UserID).Error; err != nil {
			return err
		}
		created = models.Note{Title: input.Title, Content: input.Content, UserID: user.ID}
		if err := tx.Create(&created).Error; err != nil {
			return err
		}
		for _, name := range tagNames {
			tag := models.Tag{Name: name}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "name"}}, DoNothing: true}).Create(&tag).Error; err != nil {
				return err
			}
			if err := tx.Where("name = ?", name).First(&tag).Error; err != nil {
				return err
			}
			if err := tx.Model(&created).Association("Tags").Append(&tag); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		writeFailure(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, noteSummaryDTO{ID: created.ID, Title: created.Title, Content: created.Content, UserID: created.UserID})
}

func (h *handlers) getNote(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeFailure(w, err)
		return
	}
	var note models.Note
	if err := h.db.WithContext(r.Context()).Preload("User").Preload("Tags").First(&note, id).Error; err != nil {
		writeFailure(w, err)
		return
	}
	tags := make([]tagDTO, 0, len(note.Tags))
	for _, tag := range note.Tags {
		tags = append(tags, tagDTO{ID: tag.ID, Name: tag.Name})
	}
	writeJSON(w, http.StatusOK, noteDTO{
		ID: note.ID, Title: note.Title, Content: note.Content, UserID: note.UserID,
		User: toUserDTO(note.User), Tags: tags, CreatedAt: note.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return &apiError{status: http.StatusBadRequest, message: "invalid JSON request"}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return &apiError{status: http.StatusBadRequest, message: "request must contain one JSON value"}
	}
	return nil
}

func parseID(raw string) (uint, error) {
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 || id > uint64(^uint(0)) {
		return 0, &apiError{status: http.StatusBadRequest, message: "id must be a positive integer"}
	}
	return uint(id), nil
}

func toUserDTO(user models.User) userDTO {
	return userDTO{ID: user.ID, Name: user.Name, Email: user.Email}
}

func writeFailure(w http.ResponseWriter, err error) {
	var expected *apiError
	if errors.As(err, &expected) {
		writeJSON(w, expected.status, map[string]string{"error": expected.message})
		return
	}
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "resource not found"})
	case errors.Is(err, gorm.ErrDuplicatedKey):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "email or tag already exists"})
	default:
		log.Printf("request failed with an internal database or handler error")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode JSON response: %v", err)
	}
}
