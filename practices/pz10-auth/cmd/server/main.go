package main

import (
	"log"
	"net/http"

	"github.com/Pondaaaaaa/GO/practices/pz10-auth/internal/core"
	router "github.com/Pondaaaaaa/GO/practices/pz10-auth/internal/http"
	"github.com/Pondaaaaaa/GO/practices/pz10-auth/internal/platform/config"
	jwtutil "github.com/Pondaaaaaa/GO/practices/pz10-auth/internal/platform/jwt"
	"github.com/Pondaaaaaa/GO/practices/pz10-auth/internal/repo"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	users, err := repo.NewUserMem()
	if err != nil {
		log.Fatal(err)
	}
	svc := &router.Service{Users: users, Tokens: jwtutil.New(cfg.JWTSecret, cfg.AccessTTL, cfg.RefreshTTL), Blacklist: core.NewRefreshBlacklist()}
	log.Printf("listening on %s", cfg.Addr)
	log.Fatal(http.ListenAndServe(cfg.Addr, router.Build(svc)))
}
