package messaging

import (
	"encoding/json"

	"github.com/nats-io/nats.go"
)

type NATSPublisher struct {
	conn *nats.Conn
}

func NewNATSPublisher(natsURL string) (*NATSPublisher, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}

	return &NATSPublisher{
		conn: nc,
	}, nil
}

func (p *NATSPublisher) PublishPaymentCompleted(event PaymentCompletedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.conn.Publish("payment.completed", data)
}

func (p *NATSPublisher) Close() {
	p.conn.Close()
}
