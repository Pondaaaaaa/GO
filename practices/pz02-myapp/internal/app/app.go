package app

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/Pondaaaaaa/GO/practices/pz02-myapp/internal/app/handlers"
	"github.com/Pondaaaaaa/GO/practices/pz02-myapp/utils"
)

func Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello, Go project structure!"))
	})
	mux.HandleFunc("/ping", handlers.Ping)
	mux.HandleFunc("/fail", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fail" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if err := utils.WriteErr(w, http.StatusBadRequest, "bad_request_example"); err != nil {
			utils.Error("write error response", "error", err)
		}
	})

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf(":%s", port)
	utils.Info("starting HTTP server", "address", addr)

	handler := utils.RequestID(mux)
	handler = utils.RequestLogger(handler)
	if err := http.ListenAndServe(addr, handler); err != nil && !errors.Is(err, http.ErrServerClosed) {
		utils.Error("HTTP server failed", "address", addr, "error", err)
		return err
	}
	return nil
}
