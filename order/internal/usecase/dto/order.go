package dto

import "order/internal/domain"

type CreateOrderInput struct {
	CustomerID string
	ItemName   string
	Amount     int64
}

type CreateOrderOutput struct {
	Order   *domain.Order
	Status  string
	Message string
}
