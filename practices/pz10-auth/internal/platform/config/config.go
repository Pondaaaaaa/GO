package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr                  string
	JWTSecret             []byte
	AccessTTL, RefreshTTL time.Duration
}

func Load() (Config, error) {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return Config{}, errors.New("APP_PORT must be 1..65535")
	}
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		return Config{}, errors.New("JWT_SECRET must contain at least 32 bytes")
	}
	access := 15 * time.Minute
	refresh := 7 * 24 * time.Hour
	if raw := os.Getenv("JWT_TTL"); raw != "" {
		access, err = time.ParseDuration(raw)
		if err != nil || access <= 0 {
			return Config{}, errors.New("JWT_TTL must be a positive duration")
		}
	}
	if raw := os.Getenv("JWT_REFRESH_TTL"); raw != "" {
		refresh, err = time.ParseDuration(raw)
		if err != nil || refresh <= 0 {
			return Config{}, errors.New("JWT_REFRESH_TTL must be a positive duration")
		}
	}
	if access >= refresh {
		return Config{}, errors.New("access TTL must be shorter than refresh TTL")
	}
	host := "127.0.0.1"
	if raw := os.Getenv("APP_HOST"); raw != "" {
		host = strings.TrimSpace(raw)
	}
	return Config{Addr: host + ":" + port, JWTSecret: []byte(secret), AccessTTL: access, RefreshTTL: refresh}, nil
}
