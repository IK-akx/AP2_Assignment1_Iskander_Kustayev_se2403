package idempotency

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// Key prefix for idempotency records
	idempotencyKeyPrefix = "notif:payment:"

	// TTL for idempotency records (keep for 24 hours to prevent duplicate processing)
	defaultTTL = 24 * time.Hour
)

// RedisStore provides idempotency storage using Redis
type RedisStore struct {
	client *redis.Client
	ctx    context.Context
	ttl    time.Duration
}

// NewRedisStore creates a new Redis-based idempotency store
func NewRedisStore(redisClient *redis.Client, ttl time.Duration) *RedisStore {
	if ttl == 0 {
		ttl = defaultTTL
	}

	return &RedisStore{
		client: redisClient,
		ctx:    context.Background(),
		ttl:    ttl,
	}
}

// IsProcessed checks if an event has already been processed
// Returns (isProcessed, error)
func (s *RedisStore) IsProcessed(eventID string) (bool, error) {
	key := s.getKey(eventID)

	exists, err := s.client.Exists(s.ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check idempotency key: %w", err)
	}

	if exists > 0 {
		log.Printf("[Idempotency] Event %s already processed (cache hit)", eventID)
		return true, nil
	}

	log.Printf("[Idempotency] Event %s not processed yet (cache miss)", eventID)
	return false, nil
}

// MarkProcessed marks an event as successfully processed
func (s *RedisStore) MarkProcessed(eventID string) error {
	key := s.getKey(eventID)

	// Set with TTL to automatically clean up old records
	err := s.client.Set(s.ctx, key, "processed", s.ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}

	log.Printf("[Idempotency] Event %s marked as processed (TTL: %v)", eventID, s.ttl)
	return nil
}

// getKey returns the Redis key for an event ID
func (s *RedisStore) getKey(eventID string) string {
	return idempotencyKeyPrefix + eventID
}

// Close closes the Redis connection if needed
func (s *RedisStore) Close() error {
	return s.client.Close()
}
