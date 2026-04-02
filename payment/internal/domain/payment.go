package domain

import (
	"fmt"
	"time"
)

type Payment struct {
	ID            string    `gorm:"primaryKey;size:50" json:"id"`
	OrderID       string    `gorm:"not null;size:50;uniqueIndex" json:"order_id"`
	TransactionID string    `gorm:"not null;size:100;uniqueIndex" json:"transaction_id"`
	Amount        int64     `gorm:"not null;check:amount > 0" json:"amount"`
	Status        string    `gorm:"not null;size:20;index" json:"status"` // "Authorized", "Declined"
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func GeneratePaymentID() string {
	return fmt.Sprintf("PAY-%d", time.Now().UnixNano())
}

func GenerateTransactionID() string {
	return fmt.Sprintf("TXN-%d", time.Now().UnixNano())
}

func (Payment) TableName() string {
	return "payments"
}

const (
	PaymentStatusAuthorized = "Authorized"
	PaymentStatusDeclined   = "Declined"
)

type PaymentRepository interface {
	Create(payment *Payment) error
	GetByOrderID(orderID string) (*Payment, error)
	GetByTransactionID(transactionID string) (*Payment, error)
}

const PaymentLimit int64 = 100000
