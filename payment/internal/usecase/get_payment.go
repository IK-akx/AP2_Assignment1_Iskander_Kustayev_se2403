package usecase

import (
	"fmt"
	"payment/internal/domain"
)

// GetPaymentUseCase handles retrieving payment details
type GetPaymentUseCase struct {
	paymentRepo domain.PaymentRepository
}

// NewGetPaymentUseCase creates a new instance
func NewGetPaymentUseCase(paymentRepo domain.PaymentRepository) *GetPaymentUseCase {
	return &GetPaymentUseCase{
		paymentRepo: paymentRepo,
	}
}

// Execute retrieves a payment by order ID
func (uc *GetPaymentUseCase) Execute(orderID string) (*domain.Payment, error) {
	if orderID == "" {
		return nil, fmt.Errorf("order_id is required")
	}

	payment, err := uc.paymentRepo.GetByOrderID(orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	if payment == nil {
		return nil, fmt.Errorf("payment not found for order: %s", orderID)
	}

	return payment, nil
}
