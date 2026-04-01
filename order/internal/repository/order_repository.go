package repository

import (
	"errors"
	"fmt"
	"order/internal/domain"

	"gorm.io/gorm"
)

type OrderRepository struct {
	db *gorm.DB
}

// NewOrderRepository creates a new OrderRepository instance
func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{
		db: db,
	}
}

// Create inserts a new order into the database
func (r *OrderRepository) Create(order *domain.Order) error {
	result := r.db.Create(order)
	if result.Error != nil {
		return fmt.Errorf("failed to create order: %w", result.Error)
	}
	return nil
}

// GetByID retrieves an order by its ID
func (r *OrderRepository) GetByID(id string) (*domain.Order, error) {
	var order domain.Order
	result := r.db.First(&order, "id = ?", id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get order: %w", result.Error)
	}

	return &order, nil
}

// UpdateStatus updates the status of an order
func (r *OrderRepository) UpdateStatus(id, status string) error {
	result := r.db.Model(&domain.Order{}).Where("id = ?", id).Update("status", status)

	if result.Error != nil {
		return fmt.Errorf("failed to update order status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("order not found: %s", id)
	}

	return nil
}
