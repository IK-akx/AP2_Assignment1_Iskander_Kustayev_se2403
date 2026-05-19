package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"order/internal/cache"
	"order/internal/client"
	grpcDelivery "order/internal/delivery/grpc"
	"order/internal/delivery/rest"
	"order/internal/domain"
	"order/internal/middleware"
	"order/internal/repository"
	"order/internal/usecase"

	orderpb "github.com/IK-akx/ap2-generated/order"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	db := initDatabase()

	if err := db.AutoMigrate(&domain.Order{}); err != nil {
		log.Fatal("Failed to migrate database: ", err)
	}
	log.Println("Database migrated successfully")

	// Инициализация Redis клиента (общий для кэша и rate limiter)
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort)

	// Redis для кэша (DB 0)
	cacheRedis := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       0,
	})

	ctx := context.Background()
	var orderCache *cache.OrderCache

	if err := cacheRedis.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis not available for cache - %v. Cache will be disabled.", err)
	} else {
		log.Println("Redis connected successfully for cache")
		redisCache := &cache.RedisClient{
			Client: cacheRedis,
			Ctx:    ctx,
		}
		ttlSeconds := getEnvAsInt("CACHE_TTL", 300)
		ttl := time.Duration(ttlSeconds) * time.Second
		orderCache = cache.NewOrderCache(redisCache, ttl)
		defer cacheRedis.Close()
	}

	// Redis для rate limiter (DB 1 - отдельная база)
	rateLimitRedis := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       1,
	})

	var rateLimiter *cache.RateLimiter
	if err := rateLimitRedis.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis not available for rate limiter - %v. Rate limiting will be disabled.", err)
	} else {
		rateLimit := getEnvAsInt("RATE_LIMIT_REQUESTS", 10)
		rateLimitWindow := getEnvAsInt("RATE_LIMIT_WINDOW_SECONDS", 60)
		rateLimiter = cache.NewRateLimiter(
			rateLimitRedis,
			rateLimit,
			time.Duration(rateLimitWindow)*time.Second,
		)
		log.Printf("Rate limiter enabled: %d requests per %d seconds", rateLimit, rateLimitWindow)
		defer rateLimitRedis.Close()
	}

	orderRepo := repository.NewOrderRepository(db)

	// gRPC client: Order -> Payment
	grpcHost := getEnv("PAYMENT_GRPC_HOST", "localhost")
	grpcPort := getEnv("PAYMENT_GRPC_PORT", "50051")
	grpcTimeout := getEnvAsInt("PAYMENT_GRPC_TIMEOUT", 3)

	address := fmt.Sprintf("%s:%s", grpcHost, grpcPort)

	paymentClient, err := client.NewPaymentGrpcClient(address, time.Duration(grpcTimeout)*time.Second)
	if err != nil {
		log.Fatal("failed to connect to payment service:", err)
	}

	// Notifier for streaming
	notifier := grpcDelivery.NewNotifier()

	orderUC := usecase.OrderUsecase{
		OrderRepo:   orderRepo,
		OrderClient: paymentClient,
		Notifier:    notifier,
		OrderCache:  orderCache,
	}

	orderHandler := rest.NewOrderHandler(orderUC, orderCache)

	// gRPC server: Order streaming
	grpcServer := grpc.NewServer()

	go func() {
		orderGrpcPort := getEnv("ORDER_GRPC_PORT", "50052")

		lis, err := net.Listen("tcp", ":"+orderGrpcPort)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		orderpb.RegisterOrderTrackingServiceServer(
			grpcServer,
			grpcDelivery.NewOrderGrpcHandler(notifier),
		)

		log.Printf("Order gRPC server running on port %s", orderGrpcPort)

		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	// REST API
	router := gin.Default()

	// Apply rate limiter middleware (if enabled)
	if rateLimiter != nil {
		rateLimiterMiddleware := middleware.RateLimiterMiddleware(
			rateLimiter,
			middleware.GetIdentifierByIP,
		)
		router.Use(rateLimiterMiddleware)
		log.Println("Rate limiter middleware applied to all routes")
	}

	router.POST("/orders", orderHandler.CreateOrder)
	router.GET("/orders/:id", orderHandler.GetOrder)
	router.PATCH("/orders/:id/cancel", orderHandler.CancelOrder)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	port := getEnv("ORDER_SERVICE_PORT", "8080")

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		log.Printf("Order Service starting on port %s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server: ", err)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Order Service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Println("HTTP server shutdown error:", err)
	}

	grpcServer.GracefulStop()

	log.Println("Order Service stopped gracefully")
}

func initDatabase() *gorm.DB {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "order_db")
	sslMode := getEnv("DB_SSL_MODE", "disable")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslMode)

	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database instance: ", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Database connected successfully")
	return db
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
