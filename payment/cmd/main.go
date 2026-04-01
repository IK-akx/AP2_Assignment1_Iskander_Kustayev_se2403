package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"payment/internal/delivery/rest"
	"payment/internal/domain"
	"payment/internal/repository"
	"payment/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Database connection
	db := initDatabase()

	// Auto migrate the schema
	if err := db.AutoMigrate(&domain.Payment{}); err != nil {
		log.Fatal("Failed to migrate database: ", err)
	}
	log.Println("Database migrated successfully")

	// Initialize repositories
	paymentRepo := repository.NewPaymentRepository(db)

	// Initialize use cases
	authorizeUC := usecase.NewAuthorizePaymentUseCase(paymentRepo)
	getPaymentUC := usecase.NewGetPaymentUseCase(paymentRepo)

	// Initialize handlers
	paymentHandler := rest.NewPaymentHandler(authorizeUC, getPaymentUC)

	// Setup Gin router
	router := gin.Default()

	// Routes
	router.POST("/payments", paymentHandler.AuthorizePayment)
	router.GET("/payments/:order_id", paymentHandler.GetPayment)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Start server
	port := getEnv("PAYMENT_SERVICE_PORT", "8081")
	log.Printf("Payment Service starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}

func initDatabase() *gorm.DB {
	// Database configuration
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "payment_db") // Different database from order service
	sslMode := getEnv("DB_SSL_MODE", "disable")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslMode)

	// Configure GORM
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	// Get underlying sql.DB to set connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database instance: ", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Database connected successfully")
	return db
}

// Helper functions to get environment variables
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
