package messaging

type EventPublisher interface {
	PublishPaymentCompleted(event PaymentCompletedEvent) error
}
