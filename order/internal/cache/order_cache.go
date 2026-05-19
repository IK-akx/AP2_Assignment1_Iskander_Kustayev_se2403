package cache

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"order/internal/domain"
)

const (
	OrderCachePrefix = "order:"
	DefaultTTL       = 5 * time.Minute // 5 minutes as specified
)

type OrderCache struct {
	redis *RedisClient
	ttl   time.Duration
}

func NewOrderCache(redis *RedisClient, ttl time.Duration) *OrderCache {
	if ttl == 0 {
		ttl = DefaultTTL
	}
	return &OrderCache{
		redis: redis,
		ttl:   ttl,
	}
}

// Get retrieves an order from cache by ID
func (c *OrderCache) Get(orderID string) (*domain.Order, error) {
	key := c.getKey(orderID)

	data, err := c.redis.Client.Get(c.redis.Ctx, key).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			// Cache miss
			return nil, nil
		}
		log.Printf("Redis get error for key %s: %v", key, err)
		return nil, err
	}

	var order domain.Order
	if err := json.Unmarshal([]byte(data), &order); err != nil {
		log.Printf("Failed to unmarshal order from cache: %v", err)
		return nil, err
	}

	log.Printf("Cache HIT for order %s", orderID)
	return &order, nil
}

// Set stores an order in cache with TTL
func (c *OrderCache) Set(order *domain.Order) error {
	key := c.getKey(order.ID)

	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	err = c.redis.Client.Set(c.redis.Ctx, key, data, c.ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	log.Printf("Cached order %s with TTL %v", order.ID, c.ttl)
	return nil
}

// Delete removes an order from cache (invalidation)
func (c *OrderCache) Delete(orderID string) error {
	key := c.getKey(orderID)

	err := c.redis.Client.Del(c.redis.Ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete cache key %s: %w", key, err)
	}

	log.Printf("Cache INVALIDATED for order %s", orderID)
	return nil
}

// getKey returns the Redis key for an order ID
func (c *OrderCache) getKey(orderID string) string {
	return OrderCachePrefix + orderID
}
