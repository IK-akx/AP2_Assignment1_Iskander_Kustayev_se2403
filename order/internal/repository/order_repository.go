package repository

import (
	"fmt"
	"order/internal/domain"

	"gorm.io/gorm"
)

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{
		db: db,
	}
}

func (r *OrderRepository) Create(order *domain.Order) error {
	result := r.db.Create(order)
	return result.Error
}

func (r *OrderRepository) GetByID(id string) (*domain.Order, error) {
	var order domain.Order
	result := r.db.First(&order, "id = ?", id)

	if result.Error != nil {
		return nil, result.Error
	}

	return &order, nil
}

func (r *OrderRepository) UpdateStatus(id, status string) error {
	result := r.db.Model(&domain.Order{}).Where("id = ?", id).Update("status", status)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("order not found: %s", id)
	}

	return nil
}
