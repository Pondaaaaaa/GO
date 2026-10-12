package task

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pondaaaaaa/GO/practices/pz04-todo/pkg/middleware"
	"github.com/go-chi/chi/v5"
)

func testRouter() http.Handler {
	h := NewHandler(NewRepository())
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.CORS)
	r.Get("/health", h.Health)
	r.Get("/api/tasks", h.List)
	r.Post("/api/tasks", h.Create)
	r.Get("/api/tasks/{id}", h.Get)
	r.Put("/api/tasks/{id}", h.Update)
	r.Delete("/api/tasks/{id}", h.Delete)
	return r
}

func perform(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	r.ServeHTTP(response, req)
	return response
}

func TestCRUDAndErrors(t *testing.T) {
	r := testRouter()
	if got := perform(r, http.MethodGet, "/health", ""); got.Code != http.StatusOK || got.Body.String() != "OK" {
		t.Fatalf("health: status=%d body=%q", got.Code, got.Body.String())
	}

	created := perform(r, http.MethodPost, "/api/tasks", `{"title":"  Read Go  "}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create: status=%d body=%s", created.Code, created.Body.String())
	}
	var item Task
	if err := json.Unmarshal(created.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	if item.ID != 1 || item.Title != "Read Go" || item.Done {
		t.Fatalf("unexpected created task: %+v", item)
	}
	list := perform(r, http.MethodGet, "/api/tasks", "")
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte(`"title":"Read Go"`)) {
		t.Fatalf("list: status=%d body=%s", list.Code, list.Body.String())
	}
	get := perform(r, http.MethodGet, "/api/tasks/1", "")
	if get.Code != http.StatusOK {
		t.Fatalf("get: status=%d body=%s", get.Code, get.Body.String())
	}
	updated := perform(r, http.MethodPut, "/api/tasks/1", `{"title":"Read chi","done":true}`)
	if updated.Code != http.StatusOK || !bytes.Contains(updated.Body.Bytes(), []byte(`"done":true`)) {
		t.Fatalf("update: status=%d body=%s", updated.Code, updated.Body.String())
	}
	deleted := perform(r, http.MethodDelete, "/api/tasks/1", "")
	if deleted.Code != http.StatusNoContent || deleted.Body.Len() != 0 {
		t.Fatalf("delete: status=%d body=%q", deleted.Code, deleted.Body.String())
	}
	missing := perform(r, http.MethodGet, "/api/tasks/1", "")
	if missing.Code != http.StatusNotFound || !bytes.Contains(missing.Body.Bytes(), []byte(`"error"`)) {
		t.Fatalf("missing: status=%d body=%s", missing.Code, missing.Body.String())
	}
}

func TestBadInputAndCORSPreflight(t *testing.T) {
	r := testRouter()
	wrongTypeRequest := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"title":"x"}`))
	wrongTypeRequest.Header.Set("Content-Type", "text/plain")
	wrongTypeResponse := httptest.NewRecorder()
	r.ServeHTTP(wrongTypeResponse, wrongTypeRequest)
	if wrongTypeResponse.Code != http.StatusBadRequest {
		t.Errorf("wrong Content-Type: status=%d body=%s", wrongTypeResponse.Code, wrongTypeResponse.Body.String())
	}
	for _, tc := range []struct{ path, body string }{
		{"/api/tasks/nope", ""},
		{"/api/tasks", `{`},
		{"/api/tasks", `{"title":"  "}`},
	} {
		method := http.MethodGet
		if tc.body != "" {
			method = http.MethodPost
		}
		response := perform(r, method, tc.path, tc.body)
		if response.Code != http.StatusBadRequest || response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
			t.Errorf("%s %s: status=%d content-type=%q", method, tc.path, response.Code, response.Header().Get("Content-Type"))
		}
	}

	request := httptest.NewRequest(http.MethodOptions, "/api/tasks", nil)
	request.Header.Set("Access-Control-Request-Method", "POST")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || response.Header().Get("Access-Control-Allow-Origin") != "*" || response.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatalf("preflight: status=%d headers=%v", response.Code, response.Header())
	}
}
