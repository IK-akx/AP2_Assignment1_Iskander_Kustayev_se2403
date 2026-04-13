package main

import (
	"fmt"
	"log"
	"net"
	"os"
	grpcHandler "payment/internal/delivery/grpc"
	"strconv"
	"time"

	"payment/internal/delivery/rest"
	"payment/internal/domain"
	"payment/internal/repository"
	"payment/internal/usecase"

	pb "github.com/IK-akx/ap2-generated/payment"
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

	if err := db.AutoMigrate(&domain.Payment{}); err != nil {
		log.Fatal("Failed to migrate database: ", err)
	}
	log.Println("Database migrated successfully")

	paymentRepo := repository.NewPaymentRepository(db)

	paymentUsecase := usecase.PaymentUsecase{PaymentRepo: paymentRepo}

	go func() {
		grpcPort := getEnv("PAYMENT_GRPC_PORT", "50051")

		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		grpcServer := grpc.NewServer(
			grpc.UnaryInterceptor(grpcHandler.LoggingInterceptor),
		)

		pb.RegisterPaymentServiceServer(
			grpcServer,
			grpcHandler.NewPaymentGrpcHandler(&paymentUsecase),
		)

		log.Printf("gRPC server running on port %s", grpcPort)

		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	paymentHandler := rest.NewPaymentHandler(&paymentUsecase)

	router := gin.Default()

	router.POST("/payments", paymentHandler.AuthorizePayment)
	router.GET("/payments/:order_id", paymentHandler.GetPayment)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	port := getEnv("PAYMENT_SERVICE_PORT", "8081")
	log.Printf("Payment Service starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}

func initDatabase() *gorm.DB {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "payment_db") // Different database from order service
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
