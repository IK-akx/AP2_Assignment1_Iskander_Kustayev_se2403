package usecase

import (
	"fmt"
	"payment/internal/domain"
	"time"
)

type AuthorizePaymentInput struct {
	OrderID string
	Amount  int64
}

type AuthorizePaymentOutput struct {
	TransactionID string
	Status        string
	Message       string
}

type AuthorizePaymentUseCase struct {
	paymentRepo domain.PaymentRepository
}

func NewAuthorizePaymentUseCase(paymentRepo domain.PaymentRepository) *AuthorizePaymentUseCase {
	return &AuthorizePaymentUseCase{
		paymentRepo: paymentRepo,
	}
}

func (uc *AuthorizePaymentUseCase) Execute(input AuthorizePaymentInput) (*AuthorizePaymentOutput, error) {
	if input.OrderID == "" {
		return nil, fmt.Errorf("order_id is required")
	}

	if input.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}

	existingPayment, err := uc.paymentRepo.GetByOrderID(input.OrderID)
	if err == nil && existingPayment != nil {
		// Return existing payment if found (idempotency)
		return &AuthorizePaymentOutput{
			TransactionID: existingPayment.TransactionID,
			Status:        existingPayment.Status,
			Message:       "Payment already processed for this order",
		}, nil
	}

	var status string
	var message string

	if input.Amount > domain.PaymentLimit {
		status = domain.PaymentStatusDeclined
		message = fmt.Sprintf("Payment declined: amount %d exceeds limit of %d", input.Amount, domain.PaymentLimit)
	} else {
		status = domain.PaymentStatusAuthorized
		message = "Payment authorized successfully"
	}

	transactionID := generateTransactionID()

	payment := &domain.Payment{
		ID:            generatePaymentID(),
		OrderID:       input.OrderID,
		TransactionID: transactionID,
		Amount:        input.Amount,
		Status:        status,
		CreatedAt:     time.Now(),
	}

	if err := uc.paymentRepo.Create(payment); err != nil {
		return nil, fmt.Errorf("failed to save payment: %w", err)
	}

	return &AuthorizePaymentOutput{
		TransactionID: transactionID,
		Status:        status,
		Message:       message,
	}, nil
}

func generatePaymentID() string {
	return fmt.Sprintf("PAY-%d", time.Now().UnixNano())
}

func generateTransactionID() string {
	return fmt.Sprintf("TXN-%d", time.Now().UnixNano())
}
