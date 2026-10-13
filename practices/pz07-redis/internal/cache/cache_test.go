package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestSetGetTTLAndExpiration(t *testing.T) {
	server := miniredis.RunT(t)
	c := &Client{redis: redis.NewClient(&redis.Options{Addr: server.Addr()}), timeout: time.Second}
	t.Cleanup(func() { _ = c.Close() })
	if err := c.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("k", "v", 10*time.Second); err != nil {
		t.Fatal(err)
	}
	got, err := c.Get("k")
	if err != nil || got != "v" {
		t.Fatalf("Get() = %q, %v", got, err)
	}
	ttl, err := c.TTL("k")
	if err != nil || ttl <= 0 || ttl > 10*time.Second {
		t.Fatalf("TTL() = %v, %v", ttl, err)
	}
	server.FastForward(11 * time.Second)
	if _, err := c.Get("k"); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("expired Get() error = %v", err)
	}
	if _, err := c.TTL("k"); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("expired TTL() error = %v", err)
	}
}
