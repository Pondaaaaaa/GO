package httpx_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Pondaaaaaa/GO/practices/pz11-notes/internal/core/service"
	httpx "github.com/Pondaaaaaa/GO/practices/pz11-notes/internal/http"
	"github.com/Pondaaaaaa/GO/practices/pz11-notes/internal/http/handlers"
	"github.com/Pondaaaaaa/GO/practices/pz11-notes/internal/repo"
)

func router() http.Handler {
	return httpx.NewRouter(&handlers.Handler{Notes: service.NewNoteService(repo.NewNoteRepoMem())})
}

func request(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestCRUD(t *testing.T) {
	h := router()
	if w := request(t, h, "GET", "/api/v1/notes", ""); w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("empty list: %d %s", w.Code, w.Body.String())
	}
	w := request(t, h, "POST", "/api/v1/notes", `{"title":"First","content":"Body"}`)
	if w.Code != 201 || w.Header().Get("Location") != "/api/v1/notes/1" {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var n struct {
		ID        int64  `json:"id"`
		Title     string `json:"title"`
		CreatedAt string `json:"createdAt"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &n); err != nil || n.ID != 1 || n.Title != "First" || n.CreatedAt == "" {
		t.Fatalf("created note: %s (%v)", w.Body.String(), err)
	}
	if w := request(t, h, "GET", "/api/v1/notes/1", ""); w.Code != 200 {
		t.Fatalf("get: %d", w.Code)
	}
	if w := request(t, h, "GET", "/api/v1/notes", ""); w.Code != 200 || !strings.Contains(w.Body.String(), `"id":1`) {
		t.Fatalf("list: %d %s", w.Code, w.Body.String())
	}
	w = request(t, h, "PATCH", "/api/v1/notes/1", `{"content":"Changed"}`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"updatedAt"`) || !strings.Contains(w.Body.String(), `"content":"Changed"`) {
		t.Fatalf("patch: %d %s", w.Code, w.Body.String())
	}
	w = request(t, h, "DELETE", "/api/v1/notes/1", "")
	if w.Code != 204 || w.Body.Len() != 0 {
		t.Fatalf("delete: %d %q", w.Code, w.Body.String())
	}
	if w := request(t, h, "GET", "/api/v1/notes/1", ""); w.Code != 404 {
		t.Fatalf("get after delete: %d", w.Code)
	}
}

func TestInvalidAndMissing(t *testing.T) {
	h := router()
	for _, tt := range []struct {
		method, path, body string
		status             int
	}{
		{"POST", "/api/v1/notes", `{"title":" "}`, 400},
		{"POST", "/api/v1/notes", `{"title":"x","other":1}`, 400},
		{"POST", "/api/v1/notes", `{"title":`, 400},
		{"GET", "/api/v1/notes/nope", "", 400},
		{"GET", "/api/v1/notes/9", "", 404},
		{"PATCH", "/api/v1/notes/9", `{"content":"x"}`, 404},
		{"DELETE", "/api/v1/notes/9", "", 404},
		{"PATCH", "/api/v1/notes/1", `{}`, 400},
	} {
		w := request(t, h, tt.method, tt.path, tt.body)
		if w.Code != tt.status || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
			t.Errorf("%s %s: %d %s", tt.method, tt.path, w.Code, w.Body.String())
		}
	}
}

func TestConcurrentCreates(t *testing.T) {
	h := router()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := request(t, h, "POST", "/api/v1/notes", `{"title":"x"}`)
			if w.Code != 201 {
				t.Errorf("create: %d", w.Code)
			}
		}()
	}
	wg.Wait()
	w := request(t, h, "GET", "/api/v1/notes", "")
	var notes []json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &notes); err != nil || len(notes) != 20 {
		t.Fatalf("list: %s (%v)", w.Body.String(), err)
	}
}
