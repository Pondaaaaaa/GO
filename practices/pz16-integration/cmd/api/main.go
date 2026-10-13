package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Pondaaaaaa/GO/practices/pz16-integration/internal/db"
	"github.com/Pondaaaaaa/GO/practices/pz16-integration/internal/httpapi"
	"github.com/Pondaaaaaa/GO/practices/pz16-integration/internal/repo"
	"github.com/Pondaaaaaa/GO/practices/pz16-integration/internal/service"
)

func main() {
	d, err := db.Open(context.Background(), os.Getenv("DB_DSN"))
	if err != nil {
		log.Fatal("database connection failed")
	}
	defer d.Close()
	if db.Migrate(context.Background(), d) != nil {
		log.Fatal("migration failed")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "0"
	}
	l, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		log.Fatal("listener failed")
	}
	log.Printf("Notes API listening on http://%s", l.Addr())
	s := http.Server{Handler: httpapi.New(service.Service{Notes: repo.Notes{DB: d}}), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
	if err = s.Serve(l); err != nil && err != http.ErrServerClosed {
		log.Fatal("server failed")
	}
}
