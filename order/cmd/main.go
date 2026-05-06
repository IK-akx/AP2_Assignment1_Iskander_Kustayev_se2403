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

	"order/internal/client"
	grpcDelivery "order/internal/delivery/grpc"
	"order/internal/delivery/rest"
	"order/internal/domain"
	"order/internal/repository"
	"order/internal/usecase"

	orderpb "github.com/IK-akx/ap2-generated/order"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
	}

	orderHandler := rest.NewOrderHandler(orderUC)

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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

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
