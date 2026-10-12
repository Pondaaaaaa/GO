package api

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Pondaaaaaa/GO/practices/pz03-http/internal/storage"
)

func TestCreateTaskAndValidation(t *testing.T) {
	handler := NewHandler(storage.NewMemoryStore())

	request := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(`{"title":"  Buy milk  "}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body=%s", response.Code, http.StatusCreated, response.Body.String())
	}
	var created storage.Task
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created task: %v", err)
	}
	if created.ID != 1 || created.Title != "Buy milk" || created.Done {
		t.Fatalf("created task = %#v, want ID 1, trimmed title and done=false", created)
	}

	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "wrong content type", contentType: "text/plain", body: `{"title":"Task"}`},
		{name: "invalid JSON", contentType: "application/json", body: `{"title":`},
		{name: "empty title", contentType: "application/json", body: `{"title":"   "}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(test.body))
			request.Header.Set("Content-Type", test.contentType)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusBadRequest, response.Body.String())
			}
			if response.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", response.Header().Get("Content-Type"))
			}
		})
	}
}

func TestGetTaskFoundBadIDAndMissing(t *testing.T) {
	store := storage.NewMemoryStore()
	created := store.Create("Existing task")
	handler := NewHandler(store)

	tests := []struct {
		path       string
		wantStatus int
	}{
		{path: "/tasks/1", wantStatus: http.StatusOK},
		{path: "/tasks/not-a-number", wantStatus: http.StatusBadRequest},
		{path: "/tasks/9999", wantStatus: http.StatusNotFound},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.wantStatus, response.Body.String())
			}
			if test.wantStatus == http.StatusOK {
				var received storage.Task
				if err := json.Unmarshal(response.Body.Bytes(), &received); err != nil {
					t.Fatalf("decode task: %v", err)
				}
				if received.ID != created.ID || received.Title != created.Title {
					t.Fatalf("received task = %#v, want %#v", received, created)
				}
			}
		})
	}
}

func TestLoggingDefaultsImplicitStatusToOK(t *testing.T) {
	var output bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(previousOutput)

	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/implicit-status", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("response status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(output.String(), "status=200") {
		t.Fatalf("log = %q, want implicit status=200", output.String())
	}
}
