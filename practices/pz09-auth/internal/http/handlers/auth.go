package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"strings"

	"github.com/Pondaaaaaa/GO/practices/pz09-auth/internal/core"
	"github.com/Pondaaaaaa/GO/practices/pz09-auth/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	Users      repo.UserStore
	BcryptCost int
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func readCredentials(r *http.Request) (credentials, bool) {
	var in credentials
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&in); err != nil {
		return credentials{}, false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return credentials{}, false
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	return in, true
}

func validEmail(email string) bool {
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email && len(email) <= 255
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	in, ok := readCredentials(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if !validEmail(in.Email) || len(in.Password) < 8 || len(in.Password) > 72 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_email_or_password_length"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), h.BcryptCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "hash_failed"})
		return
	}
	user := core.User{Email: in.Email, PasswordHash: string(hash)}
	if err := h.Users.Create(r.Context(), &user); err != nil {
		if errors.Is(err, repo.ErrEmailTaken) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "email_taken"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db_error"})
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"status": "ok", "user": map[string]any{"id": user.ID, "email": user.Email}})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	in, ok := readCredentials(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if in.Email == "" || in.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email_and_password_required"})
		return
	}
	user, err := h.Users.ByEmail(r.Context(), in.Email)
	if err != nil {
		if !errors.Is(err, repo.ErrUserNotFound) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db_error"})
			return
		}
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)) != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "user": map[string]any{"id": user.ID, "email": user.Email}})
}
