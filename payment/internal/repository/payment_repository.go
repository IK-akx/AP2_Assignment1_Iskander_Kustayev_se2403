package repository

import (
	"errors"
	"fmt"
	"payment/internal/domain"

	"gorm.io/gorm"
)

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{
		db: db,
	}
}

func (r *PaymentRepository) Create(payment *domain.Payment) error {
	result := r.db.Create(payment)
	if result.Error != nil {
		return fmt.Errorf("failed to create payment: %w", result.Error)
	}
	return nil
}

func (r *PaymentRepository) GetByOrderID(orderID string) (*domain.Payment, error) {
	var payment domain.Payment
	result := r.db.Where("order_id = ?", orderID).First(&payment)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get payment: %w", result.Error)
	}

	return &payment, nil
}

func (r *PaymentRepository) GetByTransactionID(transactionID string) (*domain.Payment, error) {
	var payment domain.Payment
	result := r.db.Where("transaction_id = ?", transactionID).First(&payment)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get payment: %w", result.Error)
	}

	return &payment, nil
}
