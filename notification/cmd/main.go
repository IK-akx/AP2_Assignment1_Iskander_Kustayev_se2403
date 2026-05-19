package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"notification/internal/consumer"
	"notification/internal/email"
	"notification/internal/idempotency"
)

func main() {
	// Load email configuration
	emailConfig, err := email.LoadConfigFromEnv()
	if err != nil {
		log.Fatal("Failed to load email configuration:", err)
	}

	// Create email sender
	emailSender, err := email.NewEmailSender(emailConfig)
	if err != nil {
		log.Fatal("Failed to create email sender:", err)
	}

	log.Printf("Initialized email provider: %s", emailSender.GetProviderName())

	// Initialize Redis client
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")

	redisAddr := redisHost + ":" + redisPort
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       0,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis not available: %v", err)
	} else {
		log.Println("Redis connected successfully")
	}

	// Create idempotency store
	var idempotencyStore *idempotency.RedisStore
	if err == nil {
		ttlSeconds := getEnvAsInt("IDEMPOTENCY_TTL_HOURS", 24)
		ttl := time.Duration(ttlSeconds) * time.Hour
		idempotencyStore = idempotency.NewRedisStore(redisClient, ttl)
	}

	// Get configuration
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	numWorkers := getEnvAsInt("WORKER_COUNT", 5)
	maxRetries := getEnvAsInt("MAX_RETRIES", 5)    // ← 5 retries
	queueSize := getEnvAsInt("JOB_QUEUE_SIZE", 10) // ← ограничение канала до 10

	log.Printf("📊 Configuration:")
	log.Printf("   Workers: %d (parallel)", numWorkers)
	log.Printf("   Max Retries: %d", maxRetries)
	log.Printf("   Queue Size: %d", queueSize)
	log.Printf("   Failure Rate: %.0f%%", emailConfig.SimulatedFailureRate*100)

	// Create consumer
	consumer, err := consumer.NewConsumer(
		natsURL,
		emailSender,
		idempotencyStore,
		numWorkers,
		maxRetries,
		queueSize,
	)
	if err != nil {
		log.Fatal("Failed to create consumer:", err)
	}

	// Start consumer
	if err := consumer.Start(maxRetries); err != nil {
		log.Fatal("Failed to start consumer:", err)
	}

	log.Println("✅ Notification service is running...")

	// Start monitoring
	go monitorQueue(consumer)

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	log.Println("Shutting down notification service...")
	consumer.Stop()
}

func monitorQueue(c *consumer.Consumer) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		queueSize := c.GetQueueSize()
		log.Printf("[Monitor] 📊 Queue status: %d pending jobs", queueSize)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
