package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrKeyNotFound = errors.New("cache key not found")

type Client struct {
	redis   *redis.Client
	timeout time.Duration
}

func New(addr string, timeout time.Duration) *Client {
	return &Client{redis: redis.NewClient(&redis.Options{Addr: addr}), timeout: timeout}
}

func (c *Client) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.redis.Ping(ctx).Err()
}

func (c *Client) Set(key, value string, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	return c.redis.Set(ctx, key, value, ttl).Err()
}

func (c *Client) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	value, err := c.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrKeyNotFound
	}
	return value, err
}

func (c *Client) TTL(key string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	ttl, err := c.redis.TTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if ttl < 0 {
		exists, existsErr := c.redis.Exists(ctx, key).Result()
		if existsErr != nil {
			return 0, existsErr
		}
		if exists == 0 {
			return 0, ErrKeyNotFound
		}
	}
	return ttl, nil
}

func (c *Client) Close() error { return c.redis.Close() }
