package config

import (
	"errors"
	"os"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	DBDSN      string
	BcryptCost int
	Addr       string
}

func Load() (Config, error) {
	cfg := Config{DBDSN: os.Getenv("DB_DSN"), BcryptCost: 12, Addr: os.Getenv("APP_ADDR")}
	if cfg.DBDSN == "" {
		return Config{}, errors.New("DB_DSN is required")
	}
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	if value := os.Getenv("BCRYPT_COST"); value != "" {
		cost, err := strconv.Atoi(value)
		if err != nil || cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
			return Config{}, errors.New("BCRYPT_COST must be an integer between 4 and 31")
		}
		cfg.BcryptCost = cost
	}
	return cfg, nil
}
