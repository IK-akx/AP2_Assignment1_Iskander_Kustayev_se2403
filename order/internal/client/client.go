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

type PaymentRequest struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

type PaymentResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"` // "Authorized" or "Declined"
}

func NewPaymentClient(baseURL string, timeout time.Duration) *PaymentClient {
	return &PaymentClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout, // 2 seconds as required
		},
	}
}

func (c *PaymentClient) AuthorizePayment(orderID string, amount int64) (string, string, error) {
	reqBody := PaymentRequest{
		OrderID: orderID,
		Amount:  amount,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/payments", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if os.IsTimeout(err) || strings.Contains(err.Error(), "connection refused") {
			return "", "", fmt.Errorf("payment service unavailable: %w", err)
		}
		return "", "", fmt.Errorf("failed to call payment service: %w", err)
	}
	defer resp.Body.Close()

	var paymentResp PaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&paymentResp); err != nil {
		return "", "", fmt.Errorf("failed to parse payment response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("payment service returned error status: %d", resp.StatusCode)
	}

	return paymentResp.TransactionID, paymentResp.Status, nil
}
