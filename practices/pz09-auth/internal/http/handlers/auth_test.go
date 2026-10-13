package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Pondaaaaaa/GO/practices/pz09-auth/internal/core"
	"github.com/Pondaaaaaa/GO/practices/pz09-auth/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

type memoryUsers struct {
	byEmail map[string]core.User
}

func (m *memoryUsers) Create(_ context.Context, user *core.User) error {
	if _, exists := m.byEmail[user.Email]; exists {
		return repo.ErrEmailTaken
	}
	user.ID = uint(len(m.byEmail) + 1)
	m.byEmail[user.Email] = *user
	return nil
}

func (m *memoryUsers) ByEmail(_ context.Context, email string) (core.User, error) {
	user, exists := m.byEmail[email]
	if !exists {
		return core.User{}, repo.ErrUserNotFound
	}
	return user, nil
}

func TestRegisterAndLogin(t *testing.T) {
	users := &memoryUsers{byEmail: map[string]core.User{}}
	h := &AuthHandler{Users: users, BcryptCost: bcrypt.MinCost}
	call := func(fn http.HandlerFunc, body string, want int) string {
		t.Helper()
		r := httptest.NewRequest(http.MethodPost, "/auth", strings.NewReader(body))
		w := httptest.NewRecorder()
		fn(w, r)
		if w.Code != want {
			t.Fatalf("status = %d, want %d: %s", w.Code, want, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "Secret123!") || strings.Contains(w.Body.String(), "passwordHash") {
			t.Fatal("password or hash leaked into response")
		}
		return w.Body.String()
	}
	register := `{"email":" Example@TEST.local ","password":"Secret123!"}`
	if body := call(h.Register, register, 201); !strings.Contains(body, "example@test.local") {
		t.Fatalf("email was not normalized: %s", body)
	}
	user := users.byEmail["example@test.local"]
	if user.PasswordHash == "Secret123!" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("Secret123!")) != nil {
		t.Fatal("password was not stored as a working bcrypt hash")
	}
	call(h.Register, register, 409)
	call(h.Login, `{"email":"example@test.local","password":"Secret123!"}`, 200)
	wrong := call(h.Login, `{"email":"example@test.local","password":"wrong"}`, 401)
	missing := call(h.Login, `{"email":"missing@test.local","password":"wrong"}`, 401)
	if wrong != missing {
		t.Fatalf("login error reveals whether the email exists: %q vs %q", wrong, missing)
	}
	call(h.Register, `{"email":"bad","password":"Secret123!"}`, 400)
	call(h.Register, `{"email":"example@test.local","password":"short"}`, 400)
	call(h.Register, `{"email":"example@test.local","password":"Secret123!"} {}`, 400)
	call(h.Register, `{"email":"example@test.local","password":"Secret123!","role":"admin"}`, 400)
}

type failingUsers struct{}

func (failingUsers) Create(context.Context, *core.User) error { return errors.New("db unavailable") }
func (failingUsers) ByEmail(context.Context, string) (core.User, error) {
	return core.User{}, errors.New("db unavailable")
}

func TestDatabaseErrorsDoNotLeak(t *testing.T) {
	h := &AuthHandler{Users: failingUsers{}, BcryptCost: bcrypt.MinCost}
	for _, fn := range []http.HandlerFunc{h.Register, h.Login} {
		w := httptest.NewRecorder()
		fn(w, httptest.NewRequest("POST", "/auth", strings.NewReader(`{"email":"a@b.test","password":"Secret123!"}`)))
		if w.Code != 500 {
			t.Fatalf("status = %d, want 500", w.Code)
		}
		var response map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || response["error"] != "db_error" {
			t.Fatalf("unexpected body: %s", w.Body.String())
		}
	}
}
