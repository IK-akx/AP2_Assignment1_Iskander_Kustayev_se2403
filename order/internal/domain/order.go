package domain

import (
	"time"
)

// Order represents the core business entity
type Order struct {
	ID         string    `gorm:"primaryKey;size:50" json:"id"`
	CustomerID string    `gorm:"not null;size:100;index" json:"customer_id"`
	ItemName   string    `gorm:"not null;size:200" json:"item_name"`
	Amount     int64     `gorm:"not null;check:amount > 0" json:"amount"`              // Amount in cents
	Status     string    `gorm:"not null;size:20;default:Pending;index" json:"status"` // "Pending", "Paid", "Failed", "Cancelled"
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName specifies the table name for GORM
func (Order) TableName() string {
	return "orders"
}

// Order status constants
const (
	OrderStatusPending   = "Pending"
	OrderStatusPaid      = "Paid"
	OrderStatusFailed    = "Failed"
	OrderStatusCancelled = "Cancelled"
)

// OrderRepository defines the interface for order data persistence
type OrderRepository interface {
	Create(order *Order) error
	GetByID(id string) (*Order, error)
	UpdateStatus(id, status string) error
}

// PaymentGateway defines the interface for payment service communication
type PaymentGateway interface {
	AuthorizePayment(orderID string, amount int64) (transactionID string, status string, err error)
}
