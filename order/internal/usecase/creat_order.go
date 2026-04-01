package usecase

import (
	"fmt"
	"order/internal/domain"
	"time"
)

type CreateOrderInput struct {
	CustomerID string
	ItemName   string
	Amount     int64
}

type CreateOrderOutput struct {
	Order   *domain.Order
	Status  string
	Message string
}

type CreateOrderUseCase struct {
	orderRepo domain.OrderRepository
	paymentGW domain.PaymentGateway
}

func NewCreateOrderUseCase(
	orderRepo domain.OrderRepository,
	paymentGW domain.PaymentGateway,
) *CreateOrderUseCase {
	return &CreateOrderUseCase{
		orderRepo: orderRepo,
		paymentGW: paymentGW,
	}
}

func (uc *CreateOrderUseCase) Execute(input CreateOrderInput) (*CreateOrderOutput, error) {
	if input.Amount <= 0 {
		return nil, fmt.Errorf("invalid amount: amount must be greater than 0")
	}

	if input.CustomerID == "" {
		return nil, fmt.Errorf("customer_id is required")
	}

	if input.ItemName == "" {
		return nil, fmt.Errorf("item_name is required")
	}

	order := &domain.Order{
		ID:         generateOrderID(), // Fast ID generation
		CustomerID: input.CustomerID,
		ItemName:   input.ItemName,
		Amount:     input.Amount,
		Status:     domain.OrderStatusPending,
		CreatedAt:  time.Now(),
	}

	if err := uc.orderRepo.Create(order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	transactionID, paymentStatus, err := uc.paymentGW.AuthorizePayment(order.ID, order.Amount)

	if err != nil {
		return &CreateOrderOutput{
			Order:   order,
			Status:  "pending",
			Message: "Order created but payment service is unavailable. Please try again later.",
		}, err
	}

	var finalStatus string
	var responseMessage string

	if paymentStatus == "Authorized" {
		finalStatus = domain.OrderStatusPaid
		responseMessage = fmt.Sprintf("Payment authorized successfully. Transaction ID: %s", transactionID)
	} else {
		finalStatus = domain.OrderStatusFailed
		responseMessage = "Payment declined"
	}

	if err := uc.orderRepo.UpdateStatus(order.ID, finalStatus); err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	order.Status = finalStatus

	return &CreateOrderOutput{
		Order:   order,
		Status:  finalStatus,
		Message: responseMessage,
	}, nil
}

func generateOrderID() string {
	return fmt.Sprintf("ORD-%d", time.Now().UnixNano())
}
