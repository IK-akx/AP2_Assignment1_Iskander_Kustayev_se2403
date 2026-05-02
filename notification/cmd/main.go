package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"notification/internal/consumer"
)

func main() {
	natsURL := "nats://localhost:4222"

	consumer, err := consumer.NewConsumer(natsURL)
	if err != nil {
		log.Fatal("failed to connect to NATS:", err)
	}

	if err := consumer.Start(); err != nil {
		log.Fatal("failed to start consumer:", err)
	}

	log.Println("Notification service is running...")

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	<-sig

	log.Println("Shutting down notification service...")
	consumer.Close()
}
