package messaging

import (
	"encoding/json"

	"github.com/nats-io/nats.go"
)

const (
	streamName = "PAYMENT_EVENTS"
	subject    = "payment.completed"
)

type NATSPublisher struct {
	conn *nats.Conn
	js   nats.JetStreamContext
}

func NewNATSPublisher(natsURL string) (*NATSPublisher, error) {
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

	return &NATSPublisher{
		conn: nc,
		js:   js,
	}, nil
}

func (p *NATSPublisher) PublishPaymentCompleted(event PaymentCompletedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = p.js.Publish(subject, data)
	return err
}

func (p *NATSPublisher) Close() {
	p.conn.Close()
}
