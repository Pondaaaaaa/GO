package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Pondaaaaaa/GO/practices/pz03-http/internal/api"
	"github.com/Pondaaaaaa/GO/practices/pz03-http/internal/storage"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	handler := api.NewHandler(storage.NewMemoryStore())
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Printf("HTTP server failed: %v", err)
		os.Exit(1)
	}
}
