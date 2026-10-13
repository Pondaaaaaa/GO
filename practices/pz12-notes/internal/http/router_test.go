package httpx_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Pondaaaaaa/GO/practices/pz12-notes/docs"
	"github.com/Pondaaaaaa/GO/practices/pz12-notes/internal/core"
	"github.com/Pondaaaaaa/GO/practices/pz12-notes/internal/core/service"
	httpx "github.com/Pondaaaaaa/GO/practices/pz12-notes/internal/http"
	"github.com/Pondaaaaaa/GO/practices/pz12-notes/internal/http/handlers"
	"github.com/Pondaaaaaa/GO/practices/pz12-notes/internal/repo"
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
		{"POST", "/api/v1/notes", `{"title":"x"} {}`, 400},
		{"POST", "/api/v1/notes", `null`, 400},
	} {
		w := request(t, h, tt.method, tt.path, tt.body)
		if w.Code != tt.status || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
			t.Errorf("%s %s: %d %s", tt.method, tt.path, w.Code, w.Body.String())
		}
	}
}

func TestContentTypeAndBodyLimit(t *testing.T) {
	h := router()
	for _, contentType := range []string{"application/jsonx", "text/plain", ""} {
		r := httptest.NewRequest("POST", "/api/v1/notes", strings.NewReader(`{"title":"x"}`))
		r.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 {
			t.Errorf("content type %q: %d", contentType, w.Code)
		}
	}
	w := request(t, h, "POST", "/api/v1/notes", `{"title":"x","content":"`+strings.Repeat("a", 1<<20)+`"}`)
	if w.Code != 400 {
		t.Fatalf("oversized body: %d", w.Code)
	}
}

func TestDocumentationContract(t *testing.T) {
	h := router()
	if w := request(t, h, "GET", "/docs/index.html", ""); w.Code != 200 || !strings.Contains(w.Body.String(), "SwaggerUIBundle") {
		t.Fatalf("UI: %d", w.Code)
	}
	w := request(t, h, "GET", "/docs/doc.json", "")
	if w.Code != 200 {
		t.Fatalf("spec status: %d", w.Code)
	}
	var spec struct {
		Swagger  string `json:"swagger"`
		BasePath string `json:"basePath"`
		Paths    map[string]map[string]struct {
			Parameters []struct{ In string } `json:"parameters"`
			Responses  map[string]struct {
				Schema json.RawMessage `json:"schema"`
			} `json:"responses"`
		} `json:"paths"`
		Definitions map[string]json.RawMessage `json:"definitions"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil {
		t.Fatal(err)
	}
	if spec.Swagger != "2.0" || spec.BasePath != "/api/v1" || len(spec.Paths) != 2 {
		t.Fatalf("unexpected specification: %s", docs.SwaggerInfo.Title)
	}
	for path, methods := range map[string]map[string][]string{
		"/notes":      {"get": {"200", "500"}, "post": {"201", "400", "500"}},
		"/notes/{id}": {"get": {"200", "400", "404", "500"}, "patch": {"200", "400", "404", "500"}, "delete": {"204", "400", "404", "500"}},
	} {
		if len(spec.Paths[path]) != len(methods) {
			t.Errorf("operation count on %s", path)
		}
		for method, codes := range methods {
			op := spec.Paths[path][method]
			if len(op.Responses) != len(codes) {
				t.Errorf("response count %s %s", method, path)
			}
			for _, code := range codes {
				response, ok := op.Responses[code]
				if !ok {
					t.Errorf("missing %s %s %s", method, path, code)
				}
				if code == "204" && len(response.Schema) > 0 {
					t.Error("204 schema must be absent")
				}
			}
		}
	}
	if len(spec.Paths["/notes"]["get"].Parameters) != 0 {
		t.Error("list advertises unsupported parameters")
	}
	for _, name := range []string{"core.Note", "core.NoteCreate", "core.NoteUpdate", "core.ErrorResponse"} {
		if len(spec.Definitions[name]) == 0 {
			t.Errorf("missing %s", name)
		}
	}
}

type brokenRepo struct{}

var errStorage = errors.New("private storage failure")

func (brokenRepo) Create(core.Note) (core.Note, error) { return core.Note{}, errStorage }
func (brokenRepo) List() ([]core.Note, error)          { return nil, errStorage }
func (brokenRepo) Get(int64) (core.Note, error)        { return core.Note{}, errStorage }
func (brokenRepo) Update(int64, core.NoteUpdate, time.Time) (core.Note, error) {
	return core.Note{}, errStorage
}
func (brokenRepo) Delete(int64) error { return errStorage }

func TestRepositoryErrors(t *testing.T) {
	h := httpx.NewRouter(&handlers.Handler{Notes: service.NewNoteService(brokenRepo{})})
	for _, tt := range []struct{ method, path, body string }{
		{"GET", "/api/v1/notes", ""}, {"POST", "/api/v1/notes", `{"title":"x"}`}, {"GET", "/api/v1/notes/1", ""}, {"PATCH", "/api/v1/notes/1", `{"content":"x"}`}, {"DELETE", "/api/v1/notes/1", ""},
	} {
		w := request(t, h, tt.method, tt.path, tt.body)
		if w.Code != 500 || strings.TrimSpace(w.Body.String()) != `{"error":"internal server error"}` {
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
