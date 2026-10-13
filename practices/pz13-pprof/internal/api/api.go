package api

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"sync"

	"github.com/Pondaaaaaa/GO/practices/pz13-pprof/internal/work"
)

// HeapLab is a separate, controlled allocation experiment, not Fib storage.
type HeapLab struct {
	mu       sync.Mutex
	retained []byte
}

func (h *HeapLab) Allocate(w http.ResponseWriter, _ *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.retained = make([]byte, 4<<20)
	for i := range h.retained {
		h.retained[i] = byte(i)
	}
	fmt.Fprintf(w, "retained_bytes=%d\n", len(h.retained))
}

func NewMux(mode string, n int) *http.ServeMux {
	mux := http.NewServeMux()
	// pprof's init only registers DefaultServeMux; our mux is wired explicitly.
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("POST /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprintln(w, "ok") })
	mux.HandleFunc("GET /work", func(w http.ResponseWriter, _ *http.Request) {
		name := fmt.Sprintf("%s(%d)", mode, n)
		var result int
		// The timer measures the function call, excluding writing the HTTP body.
		func() {
			defer work.TimeIt(name)()
			if mode == "slow" {
				result = work.Fib(n)
			} else {
				result = work.FibFast(n)
			}
		}()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "%d\n", result)
	})
	heap := &HeapLab{}
	mux.HandleFunc("GET /heap-work", heap.Allocate)
	return mux
}
