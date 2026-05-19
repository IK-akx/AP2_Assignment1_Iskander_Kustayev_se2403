package email

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

// ProviderType defines available email providers
type ProviderType string

const (
	ProviderSimulated ProviderType = "SIMULATED"
	ProviderReal      ProviderType = "REAL"
)

// Config holds all email provider configuration
type Config struct {
	Provider ProviderType

	// Simulated provider settings
	SimulatedDelayMs     int
	SimulatedFailureRate float64

	// Real SMTP settings
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	SMTPUseTLS   bool
}

// LoadConfigFromEnv loads email configuration from environment variables
func LoadConfigFromEnv() (*Config, error) {
	config := &Config{
		Provider: ProviderSimulated, // default
	}

	// Read provider mode
	providerMode := os.Getenv("PROVIDER_MODE")
	if providerMode == "" {
		providerMode = "SIMULATED"
	}

	switch providerMode {
	case "SIMULATED":
		config.Provider = ProviderSimulated
	case "REAL":
		config.Provider = ProviderReal
	default:
		return nil, fmt.Errorf("invalid PROVIDER_MODE: %s (must be SIMULATED or REAL)", providerMode)
	}

	// Simulated settings
	delayMs := os.Getenv("SIMULATED_DELAY_MS")
	if delayMs == "" {
		delayMs = "100" // default 100ms
	}
	delayMsInt, err := strconv.Atoi(delayMs)
	if err != nil {
		return nil, fmt.Errorf("invalid SIMULATED_DELAY_MS: %w", err)
	}
	config.SimulatedDelayMs = delayMsInt

	failureRate := os.Getenv("SIMULATED_FAILURE_RATE")
	if failureRate == "" {
		failureRate = "0.3" // default 30% failure rate
	}
	failureRateFloat, err := strconv.ParseFloat(failureRate, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid SIMULATED_FAILURE_RATE: %w", err)
	}
	config.SimulatedFailureRate = failureRateFloat

	// Real SMTP settings
	config.SMTPHost = os.Getenv("SMTP_HOST")
	config.SMTPPort = os.Getenv("SMTP_PORT")
	config.SMTPUser = os.Getenv("SMTP_USER")
	config.SMTPPassword = os.Getenv("SMTP_PASSWORD")
	config.SMTPFrom = os.Getenv("SMTP_FROM")

	useTLS := os.Getenv("SMTP_USE_TLS")
	config.SMTPUseTLS = useTLS == "true" || useTLS == "1"

	return config, nil
}

// NewEmailSender creates an email sender based on configuration
func NewEmailSender(config *Config) (EmailSender, error) {
	switch config.Provider {
	case ProviderSimulated:
		log.Printf("Creating Simulated Email Provider with delay=%dms, failureRate=%.2f%%",
			config.SimulatedDelayMs, config.SimulatedFailureRate*100)
		return NewSimulatedEmailSender(config.SimulatedDelayMs, config.SimulatedFailureRate), nil

	case ProviderReal:
		log.Printf("Creating Real Email Provider (SMTP) with host=%s:%s",
			config.SMTPHost, config.SMTPPort)

		if config.SMTPHost == "" || config.SMTPPort == "" {
			return nil, fmt.Errorf("SMTP configuration incomplete: SMTP_HOST and SMTP_PORT are required for REAL provider")
		}

		if config.SMTPFrom == "" {
			config.SMTPFrom = config.SMTPUser // use username as from if not specified
		}

		smtpConfig := SMTPConfig{
			Host:     config.SMTPHost,
			Port:     config.SMTPPort,
			Username: config.SMTPUser,
			Password: config.SMTPPassword,
			From:     config.SMTPFrom,
			UseTLS:   config.SMTPUseTLS,
		}

		return NewRealEmailSender(smtpConfig), nil

	default:
		return nil, fmt.Errorf("unknown provider type: %s", config.Provider)
	}
}
