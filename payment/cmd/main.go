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

	grpcHandler "payment/internal/delivery/grpc"
	"payment/internal/delivery/rest"
	"payment/internal/domain"
	"payment/internal/messaging"
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

	natsURL := getEnv("NATS_URL", "nats://localhost:4222")

	eventPublisher, err := messaging.NewNATSPublisher(natsURL)
	if err != nil {
		log.Fatal("failed to connect to NATS:", err)
	}

	paymentUsecase := usecase.PaymentUsecase{
		PaymentRepo:    paymentRepo,
		EventPublisher: eventPublisher,
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcHandler.LoggingInterceptor),
	)

	go func() {
		grpcPort := getEnv("PAYMENT_GRPC_PORT", "50051")

		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

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

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		log.Printf("Payment Service starting on port %s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server: ", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	log.Println("Shutting down Payment Service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Println("HTTP server shutdown error:", err)
	}

	grpcServer.GracefulStop()
	eventPublisher.Close()

	log.Println("Payment Service stopped gracefully")
}

func initDatabase() *gorm.DB {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "payment_db")
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
