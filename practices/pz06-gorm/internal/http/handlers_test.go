package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestHealth(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewRouter(nil).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(recorder.Body.String()); got != `{"status":"ok"}` {
		t.Fatalf("body = %s, want health JSON", got)
	}
}

func TestCreateUserRejectsInvalidInputBeforeDatabase(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"","email":"not-an-email"}`))
	request.Header.Set("Content-Type", "application/json")
	NewRouter(nil).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func TestErrorResponsesDoNotExposeInternalError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{name: "validation", err: &apiError{status: http.StatusBadRequest, message: "invalid input"}, want: http.StatusBadRequest},
		{name: "not found", err: gorm.ErrRecordNotFound, want: http.StatusNotFound},
		{name: "duplicate", err: gorm.ErrDuplicatedKey, want: http.StatusConflict},
		{name: "internal", err: errors.New("secret database connection detail"), want: http.StatusInternalServerError},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			writeFailure(recorder, test.err)
			if recorder.Code != test.want {
				t.Fatalf("status = %d, want %d", recorder.Code, test.want)
			}
			if strings.Contains(recorder.Body.String(), "secret database connection detail") {
				t.Fatal("response exposed an internal database error")
			}
			if !strings.Contains(recorder.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("Content-Type = %q, want JSON", recorder.Header().Get("Content-Type"))
			}
		})
	}
}

func TestParseID(t *testing.T) {
	if _, err := parseID("0"); err == nil {
		t.Fatal("parseID(0) should fail")
	}
	if got, err := parseID("42"); err != nil || got != 42 {
		t.Fatalf("parseID(42) = %d, %v; want 42, nil", got, err)
	}
}
