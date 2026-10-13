package main

import (
	"log"
	"net/http"

	"github.com/Pondaaaaaa/GO/practices/pz09-auth/internal/http/handlers"
	"github.com/Pondaaaaaa/GO/practices/pz09-auth/internal/platform/config"
	"github.com/Pondaaaaaa/GO/practices/pz09-auth/internal/repo"
	"github.com/go-chi/chi/v5"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := repo.Open(cfg.DBDSN)
	if err != nil {
		log.Fatal("connect to PostgreSQL failed")
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("get database handle: ", err)
	}
	defer sqlDB.Close()
	users := repo.NewUserRepo(db)
	if err := users.AutoMigrate(); err != nil {
		log.Fatal("migrate users table: ", err)
	}
	auth := &handlers.AuthHandler{Users: users, BcryptCost: cfg.BcryptCost}
	router := chi.NewRouter()
	router.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	router.Post("/auth/register", auth.Register)
	router.Post("/auth/login", auth.Login)
	log.Println("listening on", cfg.Addr)
	log.Fatal(http.ListenAndServe(cfg.Addr, router))
}
