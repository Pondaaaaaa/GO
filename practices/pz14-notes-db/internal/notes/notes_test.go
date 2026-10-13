package notes

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOwnedSchemaValidation(t *testing.T) {
	for _, s := range []string{"public", "pz14_123_opt", "pz14_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa_opt;DROP SCHEMA public"} {
		if _, err := New(context.Background(), nil, s, false); err == nil {
			t.Fatalf("unsafe schema %q", s)
		}
	}
}
func TestDecode(t *testing.T) {
	for _, body := range []string{`{"title":"x","unknown":1}`, `{"title":"x"} {}`, `[1]`} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		var in Input
		if decode(r, &in) == nil {
			t.Errorf("accepted %s", body)
		}
	}
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"title":"x","content":"body"}`))
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	var in Input
	if decode(r, &in) != nil || in.Title != "x" {
		t.Fatal("valid input rejected")
	}
}
func TestPaginationValidation(t *testing.T) {
	h := Router(&Repo{Optimized: true})
	for _, path := range []string{"/api/v1/notes?limit=0", "/api/v1/notes?offset=-1", "/api/v1/notes?before_id=1", "/api/v1/notes/batch?ids=1,1", "/api/v1/notes/abc"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 400 {
			t.Errorf("%s: %d", path, w.Code)
		}
	}
}
