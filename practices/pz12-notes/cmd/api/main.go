package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/Pondaaaaaa/GO/practices/pz12-notes/docs"
	"github.com/Pondaaaaaa/GO/practices/pz12-notes/internal/core/service"
	httpx "github.com/Pondaaaaaa/GO/practices/pz12-notes/internal/http"
	"github.com/Pondaaaaaa/GO/practices/pz12-notes/internal/http/handlers"
	"github.com/Pondaaaaaa/GO/practices/pz12-notes/internal/repo"
)

// @title Notes API practice 12
// @version 1.0.0
// @description In-memory notes CRUD. Code-first Swagger 2.0 documentation.
// @BasePath /api/v1
func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		log.Fatal("PORT must be between 1 and 65535")
	}
	h := &handlers.Handler{Notes: service.NewNoteService(repo.NewNoteRepoMem())}
	srv := &http.Server{Addr: "127.0.0.1:" + port, Handler: httpx.NewRouter(h), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("Notes API listening on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
