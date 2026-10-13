package notes

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type stubStore struct {
	created   Note
	patch     Update
	createErr error
	getErr    error
	deleteErr error
}

func (s *stubStore) Create(_ context.Context, title, content string) (Note, error) {
	if s.createErr != nil {
		return Note{}, s.createErr
	}
	s.created = Note{ID: primitive.NewObjectID(), Title: title, Content: content}
	return s.created, nil
}
func (s *stubStore) List(context.Context, string, int64, int64) ([]Note, error) { return []Note{}, nil }
func (s *stubStore) Get(context.Context, primitive.ObjectID) (Note, error) {
	if s.getErr != nil {
		return Note{}, s.getErr
	}
	return s.created, nil
}
func (s *stubStore) Update(_ context.Context, _ primitive.ObjectID, patch Update) (Note, error) {
	s.patch = patch
	return s.created, nil
}
func (s *stubStore) Delete(context.Context, primitive.ObjectID) error { return s.deleteErr }

func TestInvalidRequests(t *testing.T) {
	server := NewHandler(&stubStore{}).Routes()
	cases := []struct{ method, path, body string }{
		{http.MethodPost, "/api/v1/notes", `{}`},
		{http.MethodPost, "/api/v1/notes", `{"title":42}`},
		{http.MethodPost, "/api/v1/notes", `{"title":"ok","unknown":1}`},
		{http.MethodGet, "/api/v1/notes/not-an-id", ""},
		{http.MethodGet, "/api/v1/notes?limit=-1", ""},
		{http.MethodGet, "/api/v1/notes?skip=nope", ""},
		{http.MethodPatch, "/api/v1/notes/000000000000000000000001", `{}`},
		{http.MethodPatch, "/api/v1/notes/000000000000000000000001", `{"title":null}`},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s %s %s: got %d, want 400", tc.method, tc.path, tc.body, rec.Code)
		}
	}
}

func TestStatusAndPartialPatch(t *testing.T) {
	store := &stubStore{}
	server := NewHandler(store).Routes()
	request := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rec
	}
	created := request(http.MethodPost, "/api/v1/notes", `{"title":"  Hello  ","content":"first"}`)
	if created.Code != http.StatusCreated || store.created.Title != "Hello" {
		t.Fatalf("create: %d, title %q", created.Code, store.created.Title)
	}
	idPath := "/api/v1/notes/" + store.created.ID.Hex()
	patched := request(http.MethodPatch, idPath, `{"content":"updated"}`)
	if patched.Code != http.StatusOK || store.patch.Title != nil || store.patch.Content == nil || *store.patch.Content != "updated" {
		t.Fatalf("partial patch: %d, %#v", patched.Code, store.patch)
	}
	deleted := request(http.MethodDelete, idPath, "")
	if deleted.Code != http.StatusNoContent || deleted.Body.Len() != 0 {
		t.Fatalf("delete: %d, body %q", deleted.Code, deleted.Body.String())
	}
	store.createErr = ErrDuplicateTitle
	if got := request(http.MethodPost, "/api/v1/notes", `{"title":"Hello"}`); got.Code != http.StatusConflict {
		t.Fatalf("duplicate: %d", got.Code)
	}
	store.getErr = ErrNotFound
	if got := request(http.MethodGet, idPath, ""); got.Code != http.StatusNotFound {
		t.Fatalf("missing: %d", got.Code)
	}
	store.getErr = errors.New("backend error")
	if got := request(http.MethodGet, idPath, ""); got.Code != http.StatusInternalServerError || strings.Contains(got.Body.String(), "backend error") {
		t.Fatalf("backend error response: %d %q", got.Code, got.Body.String())
	}
}
