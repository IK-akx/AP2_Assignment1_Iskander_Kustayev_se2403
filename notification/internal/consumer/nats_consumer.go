package consumer

import (
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"notification/internal/idempotency"
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
	conn  *nats.Conn
	js    nats.JetStreamContext
	store *idempotency.Store
	sub   *nats.Subscription
}

func NewConsumer(natsURL string) (*Consumer, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{subject},
		Storage:  nats.FileStorage,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		nc.Close()
		return nil, err
	}

	return &Consumer{
		conn:  nc,
		js:    js,
		store: idempotency.NewStore(),
	}, nil
}

func (c *Consumer) Start() error {
	sub, err := c.js.Subscribe(
		subject,
		func(msg *nats.Msg) {
			var event PaymentEvent

			if err := json.Unmarshal(msg.Data, &event); err != nil {
				log.Println("failed to parse message:", err)
				msg.Nak()
				return
			}

			if c.store.IsProcessed(event.EventID) {
				log.Println("duplicate event skipped:", event.EventID)
				msg.Ack()
				return
			}

			log.Printf(
				"[Notification] Sent email to %s for Order #%s. Amount: %d",
				event.CustomerEmail,
				event.OrderID,
				event.Amount,
			)

			c.store.MarkProcessed(event.EventID)

			if err := msg.Ack(); err != nil {
				log.Println("failed to ack message:", err)
				return
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
	return nil
}

func (c *Consumer) Close() {
	if c.sub != nil {
		_ = c.sub.Drain()
	}
	c.conn.Close()
}
