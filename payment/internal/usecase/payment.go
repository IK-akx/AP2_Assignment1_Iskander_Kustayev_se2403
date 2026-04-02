package usecase

import (
	"fmt"
	"payment/internal/domain"
	"payment/internal/usecase/dto"
	"time"
)

type PaymentUsecase struct {
	PaymentRepo domain.PaymentRepository
}

func (uc *PaymentUsecase) GetPaymentByOrderID(orderID string) (*domain.Payment, error) {
	payment, err := uc.PaymentRepo.GetByOrderID(orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	if payment == nil {
		return nil, fmt.Errorf("payment not found for order: %s", orderID)
	}

	return payment, nil
}

func (uc *PaymentUsecase) AuthorizePayment(input dto.AuthorizePaymentInput) (*dto.AuthorizePaymentOutput, error) {
	existingPayment, err := uc.PaymentRepo.GetByOrderID(input.OrderID)
	if err == nil && existingPayment != nil {
		// Return existing payment if found (idempotency)
		return &dto.AuthorizePaymentOutput{
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

	transactionID := domain.GenerateTransactionID()

	payment := &domain.Payment{
		ID:            domain.GeneratePaymentID(),
		OrderID:       input.OrderID,
		TransactionID: transactionID,
		Amount:        input.Amount,
		Status:        status,
		CreatedAt:     time.Now(),
	}

	if err := uc.PaymentRepo.Create(payment); err != nil {
		return nil, fmt.Errorf("failed to save payment: %w", err)
	}

	return &dto.AuthorizePaymentOutput{
		TransactionID: transactionID,
		Status:        status,
		Message:       message,
	}, nil
}
