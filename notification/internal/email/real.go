package email

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

// SMTPConfig holds the configuration for SMTP server
type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	UseTLS   bool
}

type RealEmailSender struct {
	config SMTPConfig
}

// NewRealEmailSender creates an SMTP-based email sender
func NewRealEmailSender(config SMTPConfig) *RealEmailSender {
	return &RealEmailSender{
		config: config,
	}
}

// Send sends a real email via SMTP
func (r *RealEmailSender) Send(to, subject, body string) error {
	log.Printf("[SMTP] Sending real email to %s", to)

	// Prepare email headers
	headers := make(map[string]string)
	headers["From"] = r.config.From
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=\"utf-8\""

	// Build email message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// SMTP server address
	addr := fmt.Sprintf("%s:%s", r.config.Host, r.config.Port)

	// Auth (if credentials provided)
	var auth smtp.Auth
	if r.config.Username != "" && r.config.Password != "" {
		auth = smtp.PlainAuth("", r.config.Username, r.config.Password, r.config.Host)
	}

	// TLS configuration
	var err error
	if r.config.UseTLS {
		// For ports like 465 (SMTPS)
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         r.config.Host,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect via TLS: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, r.config.Host)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Quit()

		if auth != nil {
			if err = client.Auth(auth); err != nil {
				return fmt.Errorf("SMTP auth failed: %w", err)
			}
		}

		if err = client.Mail(r.config.From); err != nil {
			return fmt.Errorf("failed to set sender: %w", err)
		}

		recipients := strings.Split(to, ",")
		for _, rcpt := range recipients {
			if err = client.Rcpt(strings.TrimSpace(rcpt)); err != nil {
				return fmt.Errorf("failed to set recipient: %w", err)
			}
		}

		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to get data writer: %w", err)
		}

		_, err = w.Write([]byte(message))
		if err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}

		err = w.Close()
		if err != nil {
			return fmt.Errorf("failed to close data writer: %w", err)
		}

	} else {
		// For ports like 25, 587 (STARTTLS)
		err = smtp.SendMail(addr, auth, r.config.From, []string{to}, []byte(message))
	}

	if err != nil {
		log.Printf("[SMTP] ❌ Failed to send email: %v", err)
		return fmt.Errorf("SMTP error: %w", err)
	}

	log.Printf("[SMTP] ✅ Email successfully sent to %s", to)
	return nil
}

func (r *RealEmailSender) GetProviderName() string {
	return fmt.Sprintf("SMTP(%s:%s)", r.config.Host, r.config.Port)
}
