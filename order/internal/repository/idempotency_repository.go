package repository

import (
	"gorm.io/gorm"
	"order/internal/domain"
)

type IdempotencyRepository struct {
	db *gorm.DB
}

func NewIdempotencyRepository(db *gorm.DB) *IdempotencyRepository {
	return &IdempotencyRepository{db: db}
}

func (r *IdempotencyRepository) Create(key, orderID string) error {
	idempotencyKey := &domain.IdempotencyKey{
		Key:     key,
		OrderID: orderID,
	}
	return r.db.Create(idempotencyKey).Error
}

func (r *IdempotencyRepository) GetOrderID(key string) (string, error) {
	var idempotencyKey domain.IdempotencyKey
	err := r.db.Where("key = ?", key).First(&idempotencyKey).Error
	if err != nil {
		return "", err
	}
	return idempotencyKey.OrderID, nil
}
