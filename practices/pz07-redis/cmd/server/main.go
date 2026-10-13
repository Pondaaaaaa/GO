package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Pondaaaaaa/GO/practices/pz07-redis/internal/cache"
)

type api struct {
	cache *cache.Client
	ttl   time.Duration
}

func main() {
	addr := env("REDIS_ADDR", "127.0.0.1:6379")
	port := env("PORT", "8080")
	client := cache.New(addr, 2*time.Second)
	defer client.Close()
	ctx, cancel := contextWithTimeout()
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		log.Fatalf("Redis unavailable at configured address: %v", err)
	}
	app := &api{cache: client, ttl: 10 * time.Second}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /set", app.set)
	mux.HandleFunc("GET /get", app.get)
	mux.HandleFunc("GET /ttl", app.getTTL)
	log.Printf("HTTP server listening on 127.0.0.1:%s", port)
	log.Fatal(http.ListenAndServe("127.0.0.1:"+port, mux))
}

func contextWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Second)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func (a *api) set(w http.ResponseWriter, r *http.Request) {
	key, value := r.URL.Query().Get("key"), r.URL.Query().Get("value")
	if strings.TrimSpace(key) == "" || value == "" {
		writeError(w, http.StatusBadRequest, "key and value are required")
		return
	}
	if err := a.cache.Set(key, value, a.ttl); err != nil {
		writeError(w, http.StatusServiceUnavailable, "cache unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"key": key, "value": value, "ttl_seconds": int(a.ttl.Seconds())})
}

func (a *api) get(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if strings.TrimSpace(key) == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}
	value, err := a.cache.Get(key)
	if errors.Is(err, cache.ErrKeyNotFound) {
		writeError(w, http.StatusNotFound, "key not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "cache unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"key": key, "value": value})
}

func (a *api) getTTL(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if strings.TrimSpace(key) == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}
	ttl, err := a.cache.TTL(key)
	if errors.Is(err, cache.ErrKeyNotFound) {
		writeError(w, http.StatusNotFound, "key not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "cache unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"key": key, "ttl_seconds": ttl.Seconds()})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
