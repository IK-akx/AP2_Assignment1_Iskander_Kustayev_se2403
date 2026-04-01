package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type PaymentClient struct {
	baseURL    string
	httpClient *http.Client
}

// PaymentRequest represents the request to payment service
type PaymentRequest struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

// PaymentResponse represents the response from payment service
type PaymentResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"` // "Authorized" or "Declined"
}

// NewPaymentClient creates a new PaymentClient instance
func NewPaymentClient(baseURL string, timeout time.Duration) *PaymentClient {
	return &PaymentClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout, // 2 seconds as required
		},
	}
}

// AuthorizePayment calls the payment service to authorize a payment
func (c *PaymentClient) AuthorizePayment(orderID string, amount int64) (string, string, error) {
	// Prepare request body
	reqBody := PaymentRequest{
		OrderID: orderID,
		Amount:  amount,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/payments", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request with timeout
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Check for timeout or connection errors
		if os.IsTimeout(err) || strings.Contains(err.Error(), "connection refused") {
			return "", "", fmt.Errorf("payment service unavailable: %w", err)
		}
		return "", "", fmt.Errorf("failed to call payment service: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var paymentResp PaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&paymentResp); err != nil {
		return "", "", fmt.Errorf("failed to parse payment response: %w", err)
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("payment service returned error status: %d", resp.StatusCode)
	}

	return paymentResp.TransactionID, paymentResp.Status, nil
}
