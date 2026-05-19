package consumer

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"notification/internal/email"
	"notification/internal/idempotency"
	"notification/internal/worker"

	"github.com/nats-io/nats.go"
)

const (
	streamName = "PAYMENT_EVENTS"
	subject    = "payment.completed"
	durable    = "notification-service"
)

type PaymentEvent struct {
	EventID       string `json:"event_id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

type Consumer struct {
	conn        *nats.Conn
	js          nats.JetStreamContext
	sub         *nats.Subscription
	emailSender email.EmailSender
	jobQueue    *worker.JobQueue
	workerPool  *worker.WorkerPool
	idempotency *idempotency.RedisStore
}

// NewConsumer creates a new NATS consumer with background job processing
func NewConsumer(
	natsURL string,
	emailSender email.EmailSender,
	idempotencyStore *idempotency.RedisStore,
	numWorkers int,
	maxRetries int,
	queueSize int,
) (*Consumer, error) {

	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}

	// Ensure stream exists
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{subject},
		Storage:  nats.FileStorage,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		nc.Close()
		return nil, err
	}

	// Create job queue with size limit
	jobQueue := worker.NewJobQueue(queueSize)

	// Create worker pool
	workerPool := worker.NewWorkerPool(
		numWorkers,
		jobQueue,
		emailSender,
		idempotencyStore,
	)

	return &Consumer{
		// ...
		jobQueue:   jobQueue,
		workerPool: workerPool,
		// ...
	}, nil
}

// Start starts the consumer and worker pool
func (c *Consumer) Start(maxRetries int) error {
	// Start worker pool first
	c.workerPool.Start()

	// Subscribe to NATS
	sub, err := c.js.Subscribe(
		subject,
		func(msg *nats.Msg) {
			var event PaymentEvent

			if err := json.Unmarshal(msg.Data, &event); err != nil {
				log.Println("failed to parse message:", err)
				msg.Nak()
				return
			}

			log.Printf("[Consumer] Received payment event: OrderID=%s, EventID=%s, Status=%s",
				event.OrderID, event.EventID, event.Status)

			// Check idempotency BEFORE creating job
			processed, err := c.idempotency.IsProcessed(event.EventID)
			if err != nil {
				log.Printf("[Consumer] Failed to check idempotency: %v", err)
				msg.Nak()
				return
			}

			if processed {
				log.Printf("[Consumer] Event %s already processed, skipping", event.EventID)
				msg.Ack()
				return
			}

			// Create notification job
			subject := c.buildEmailSubject(event)
			body := c.buildEmailBody(event)

			job := worker.NewJob(
				event.EventID,
				event.OrderID,
				event.CustomerEmail,
				subject,
				body,
				maxRetries,
			)

			// Add job to queue (async processing)
			c.jobQueue.Push(job)

			log.Printf("[Consumer] Job %s added to queue for Order %s", job.ID, event.OrderID)

			// Ack the message immediately (job is queued)
			// This prevents NATS from redelivering the message
			if err := msg.Ack(); err != nil {
				log.Printf("[Consumer] Failed to ack message: %v", err)
			}
		},
		nats.Durable(durable),
		nats.ManualAck(),
		nats.AckWait(10*time.Second),
	)

	if err != nil {
		return err
	}

	c.sub = sub
	log.Println("[Consumer] NATS consumer started successfully")
	log.Printf("[Consumer] Using email provider: %s", c.emailSender.GetProviderName())

	return nil
}

// buildEmailSubject creates email subject based on payment status
func (c *Consumer) buildEmailSubject(event PaymentEvent) string {
	if event.Status == "Authorized" {
		return fmt.Sprintf("✅ Payment Confirmed - Order #%s", event.OrderID)
	}
	return fmt.Sprintf("❌ Payment Failed - Order #%s", event.OrderID)
}

// buildEmailBody creates email body content
func (c *Consumer) buildEmailBody(event PaymentEvent) string {
	statusText := "successful"
	if event.Status != "Authorized" {
		statusText = "failed"
	}

	body := fmt.Sprintf(`
Dear Customer,

Your payment for Order #%s has been %s.

Order Details:
- Order ID: %s
- Amount: $%.2f
- Status: %s

Thank you for shopping with us!

Best regards,
Your E-commerce Team
`,
		event.OrderID,
		statusText,
		event.OrderID,
		float64(event.Amount)/100.0,
		event.Status,
	)

	return body
}

// Stop gracefully stops the consumer
func (c *Consumer) Stop() {
	log.Println("[Consumer] Stopping...")

	if c.workerPool != nil {
		c.workerPool.Stop()
	}

	if c.sub != nil {
		_ = c.sub.Drain()
	}

	if c.conn != nil {
		c.conn.Close()
	}

	if c.idempotency != nil {
		_ = c.idempotency.Close()
	}

	log.Println("[Consumer] Stopped")
}

// GetQueueSize returns the current number of pending jobs
func (c *Consumer) GetQueueSize() int {
	if c.jobQueue != nil {
		return c.jobQueue.Len()
	}
	return 0
}
