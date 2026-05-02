package consumer

import (
	"encoding/json"
	"log"

	"notification/internal/idempotency"

	"github.com/nats-io/nats.go"
)

type PaymentEvent struct {
	EventID       string `json:"event_id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

type Consumer struct {
	conn  *nats.Conn
	store *idempotency.Store
}

func NewConsumer(natsURL string) (*Consumer, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		conn:  nc,
		store: idempotency.NewStore(),
	}, nil
}

func (c *Consumer) Start() error {
	_, err := c.conn.Subscribe("payment.completed", func(msg *nats.Msg) {

		var event PaymentEvent

		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Println("failed to parse message:", err)
			return
		}

		// Idempotency check
		if c.store.IsProcessed(event.EventID) {
			log.Println("duplicate event skipped:", event.EventID)
			return
		}

		// Simulate email
		log.Printf(
			"[Notification] Sent email to %s for Order #%s. Amount: %d",
			event.CustomerEmail,
			event.OrderID,
			event.Amount,
		)

		// mark processed
		c.store.MarkProcessed(event.EventID)

	})

	return err
}

func (c *Consumer) Close() {
	c.conn.Close()
}
