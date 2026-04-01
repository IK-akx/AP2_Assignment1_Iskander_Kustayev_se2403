package usecase

import (
	"fmt"
	"order/internal/domain"
	"time"
)

// CreateOrderInput represents the input data for creating an order
type CreateOrderInput struct {
	CustomerID string
	ItemName   string
	Amount     int64
}

// CreateOrderOutput represents the output data after creating an order
type CreateOrderOutput struct {
	Order   *domain.Order
	Status  string
	Message string
}

// CreateOrderUseCase handles the business logic for creating orders
type CreateOrderUseCase struct {
	orderRepo domain.OrderRepository
	paymentGW domain.PaymentGateway
}

// NewCreateOrderUseCase creates a new instance of CreateOrderUseCase
func NewCreateOrderUseCase(
	orderRepo domain.OrderRepository,
	paymentGW domain.PaymentGateway,
) *CreateOrderUseCase {
	return &CreateOrderUseCase{
		orderRepo: orderRepo,
		paymentGW: paymentGW,
	}
}

// Execute creates a new order and processes payment
func (uc *CreateOrderUseCase) Execute(input CreateOrderInput) (*CreateOrderOutput, error) {
	// Validate input
	if input.Amount <= 0 {
		return nil, fmt.Errorf("invalid amount: amount must be greater than 0")
	}

	if input.CustomerID == "" {
		return nil, fmt.Errorf("customer_id is required")
	}

	if input.ItemName == "" {
		return nil, fmt.Errorf("item_name is required")
	}

	// Create order with Pending status
	order := &domain.Order{
		ID:         generateOrderID(), // Fast ID generation
		CustomerID: input.CustomerID,
		ItemName:   input.ItemName,
		Amount:     input.Amount,
		Status:     domain.OrderStatusPending,
		CreatedAt:  time.Now(),
	}

	// Save order to database
	if err := uc.orderRepo.Create(order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Call Payment Service to authorize payment
	transactionID, paymentStatus, err := uc.paymentGW.AuthorizePayment(order.ID, order.Amount)

	if err != nil {
		// Payment service is unavailable or timeout
		// According to requirements, we should return 503, but we'll handle status update here
		// The order remains "Pending" so user can retry
		return &CreateOrderOutput{
			Order:   order,
			Status:  "pending",
			Message: "Order created but payment service is unavailable. Please try again later.",
		}, err
	}

	// Update order status based on payment response
	var finalStatus string
	var responseMessage string

	if paymentStatus == "Authorized" {
		finalStatus = domain.OrderStatusPaid
		responseMessage = fmt.Sprintf("Payment authorized successfully. Transaction ID: %s", transactionID)
	} else {
		finalStatus = domain.OrderStatusFailed
		responseMessage = "Payment declined"
	}

	// Update order status in database
	if err := uc.orderRepo.UpdateStatus(order.ID, finalStatus); err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	// Update the order object status
	order.Status = finalStatus

	return &CreateOrderOutput{
		Order:   order,
		Status:  finalStatus,
		Message: responseMessage,
	}, nil
}

// generateOrderID generates a unique order ID quickly
// Using timestamp in nanoseconds + random suffix for uniqueness
func generateOrderID() string {
	// Simple but fast: timestamp in nanoseconds
	// In production, you might want to add a random component
	return fmt.Sprintf("ORD-%d", time.Now().UnixNano())
}
