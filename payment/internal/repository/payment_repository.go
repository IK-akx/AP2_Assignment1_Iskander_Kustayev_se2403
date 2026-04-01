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

// NewPaymentRepository creates a new PaymentRepository instance
func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{
		db: db,
	}
}

// Create inserts a new payment into the database
func (r *PaymentRepository) Create(payment *domain.Payment) error {
	result := r.db.Create(payment)
	if result.Error != nil {
		return fmt.Errorf("failed to create payment: %w", result.Error)
	}
	return nil
}

// GetByOrderID retrieves a payment by order ID
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

// GetByTransactionID retrieves a payment by transaction ID
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
