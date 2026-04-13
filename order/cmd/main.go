package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"order/internal/client"
	"order/internal/delivery/rest"
	"order/internal/domain"
	"order/internal/repository"
	"order/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	grpcDelivery "order/internal/delivery/grpc"

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

	orderRepo := repository.NewOrderRepository(db)

	// 🔹 gRPC client (Order -> Payment)
	grpcHost := getEnv("PAYMENT_GRPC_HOST", "localhost")
	grpcPort := getEnv("PAYMENT_GRPC_PORT", "50051")
	grpcTimeout := getEnvAsInt("PAYMENT_GRPC_TIMEOUT", 3)

	address := fmt.Sprintf("%s:%s", grpcHost, grpcPort)

	paymentClient, err := client.NewPaymentGrpcClient(address, time.Duration(grpcTimeout)*time.Second)
	if err != nil {
		log.Fatal("failed to connect to payment service:", err)
	}

	// 🔹 notifier для streaming
	notifier := grpcDelivery.NewNotifier()

	// 🔹 usecase (ОДИН раз!)
	orderUC := usecase.OrderUsecase{
		OrderRepo:   orderRepo,
		OrderClient: paymentClient,
		Notifier:    notifier,
	}

	orderHandler := rest.NewOrderHandler(orderUC)

	// 🔹 gRPC server (streaming)
	go func() {
		grpcPort := getEnv("ORDER_GRPC_PORT", "50052")

		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		grpcServer := grpc.NewServer()

		orderpb.RegisterOrderTrackingServiceServer(
			grpcServer,
			grpcDelivery.NewOrderGrpcHandler(notifier),
		)

		log.Printf("Order gRPC server running on port %s", grpcPort)

		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	// 🔹 REST API
	router := gin.Default()

	router.POST("/orders", orderHandler.CreateOrder)
	router.GET("/orders/:id", orderHandler.GetOrder)
	router.PATCH("/orders/:id/cancel", orderHandler.CancelOrder)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	port := getEnv("ORDER_SERVICE_PORT", "8080")
	log.Printf("Order Service starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
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
