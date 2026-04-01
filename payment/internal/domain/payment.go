package domain

import (
	"time"
)

// Payment represents the core payment entity
type Payment struct {
	ID            string    `gorm:"primaryKey;size:50" json:"id"`
	OrderID       string    `gorm:"not null;size:50;uniqueIndex" json:"order_id"`
	TransactionID string    `gorm:"not null;size:100;uniqueIndex" json:"transaction_id"`
	Amount        int64     `gorm:"not null;check:amount > 0" json:"amount"`
	Status        string    `gorm:"not null;size:20;index" json:"status"` // "Authorized", "Declined"
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName specifies the table name for GORM
func (Payment) TableName() string {
	return "payments"
}

// Payment status constants
const (
	PaymentStatusAuthorized = "Authorized"
	PaymentStatusDeclined   = "Declined"
)

// PaymentRepository defines the interface for payment data persistence
type PaymentRepository interface {
	Create(payment *Payment) error
	GetByOrderID(orderID string) (*Payment, error)
	GetByTransactionID(transactionID string) (*Payment, error)
}

// PaymentLimit is the maximum allowed payment amount (1000 units = 100000 cents)
const PaymentLimit int64 = 100000
