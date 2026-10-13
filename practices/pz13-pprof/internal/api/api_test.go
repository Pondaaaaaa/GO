package api

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoutes(t *testing.T) {
	for _, mode := range []string{"slow", "fast"} {
		mux := NewMux(mode, 10)
		for _, tt := range []struct{ path, body string }{{"/work", "55\n"}, {"/health", "ok\n"}, {"/debug/pprof/", "Types of profiles available"}, {"/debug/pprof/goroutine?debug=1", "goroutine profile"}, {"/heap-work", "retained_bytes=4194304\n"}} {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("GET", tt.path, nil))
			if w.Code != 200 || !strings.Contains(w.Body.String(), tt.body) {
				t.Errorf("%s %s: %d %s", mode, tt.path, w.Code, w.Body.String())
			}
		}
	}
}
