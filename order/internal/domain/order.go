package domain

import (
	"time"
)

type Order struct {
	ID         string    `gorm:"primaryKey;size:50" json:"id"`
	CustomerID string    `gorm:"not null;size:100;index" json:"customer_id"`
	ItemName   string    `gorm:"not null;size:200" json:"item_name"`
	Amount     int64     `gorm:"not null;check:amount > 0" json:"amount"`              // Amount in cents
	Status     string    `gorm:"not null;size:20;default:Pending;index" json:"status"` // "Pending", "Paid", "Failed", "Cancelled"
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (Order) TableName() string {
	return "orders"
}

const (
	OrderStatusPending   = "Pending"
	OrderStatusPaid      = "Paid"
	OrderStatusFailed    = "Failed"
	OrderStatusCancelled = "Cancelled"
)

type OrderRepository interface {
	Create(order *Order) error
	GetByID(id string) (*Order, error)
	UpdateStatus(id, status string) error
}

type PaymentGateway interface {
	AuthorizePayment(orderID string, amount int64) (transactionID string, status string, err error)
}
