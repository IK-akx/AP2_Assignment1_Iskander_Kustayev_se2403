package email

type EmailSender interface {
	Send(to, subject, body string) error

	// GetProviderName returns the name of the provider (for logging)
	GetProviderName() string
}
