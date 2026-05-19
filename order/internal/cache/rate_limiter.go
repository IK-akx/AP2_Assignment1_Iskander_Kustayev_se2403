package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements fixed window rate limiting
type RateLimiter struct {
	client *redis.Client
	ctx    context.Context
	prefix string
	limit  int
	window time.Duration
}

func NewRateLimiter(client *redis.Client, limit int, window time.Duration) *RateLimiter { // <-- ИЗМЕНИТЬ
	return &RateLimiter{
		client: client,
		ctx:    context.Background(),
		prefix: "ratelimit:simple:",
		limit:  limit,
		window: window,
	}
}

// Allow checks if request is allowed using fixed window algorithm
func (rl *RateLimiter) Allow(identifier string) (bool, int, time.Time, error) {
	key := rl.getKey(identifier)

	// Increment counter
	count, err := rl.client.Incr(rl.ctx, key).Result()
	if err != nil {
		log.Printf("[RateLimiter] Redis INCR error: %v", err)
		return true, rl.limit, time.Now().Add(rl.window), nil
	}

	// Set expiry on first request
	if count == 1 {
		rl.client.Expire(rl.ctx, key, rl.window)
	}

	remaining := rl.limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	// Get TTL
	ttl, err := rl.client.TTL(rl.ctx, key).Result()
	if err != nil {
		ttl = rl.window
	}

	resetTime := time.Now().Add(ttl)

	// Check if under limit
	if count > int64(rl.limit) {
		return false, 0, resetTime, nil
	}

	return true, remaining, resetTime, nil
}

func (rl *RateLimiter) getKey(identifier string) string {
	windowKey := time.Now().Truncate(rl.window).Unix()
	return fmt.Sprintf("%s:%s:%d", rl.prefix, identifier, windowKey)
}

func (rl *RateLimiter) GetLimit() int {
	return rl.limit
}

func (rl *RateLimiter) GetWindow() time.Duration {
	return rl.window
}

func (rl *RateLimiter) Reset(identifier string) error {
	pattern := fmt.Sprintf("%s:%s:*", rl.prefix, identifier)
	iter := rl.client.Scan(rl.ctx, 0, pattern, 0).Iterator()

	for iter.Next(rl.ctx) {
		if err := rl.client.Del(rl.ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}

	return iter.Err()
}
