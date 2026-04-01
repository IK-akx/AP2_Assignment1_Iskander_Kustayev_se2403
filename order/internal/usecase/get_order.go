package usecase

import (
	"fmt"
	"order/internal/domain"
)

type GetOrderUseCase struct {
	orderRepo domain.OrderRepository
}

func NewGetOrderUseCase(orderRepo domain.OrderRepository) *GetOrderUseCase {
	return &GetOrderUseCase{
		orderRepo: orderRepo,
	}
}

func (uc *GetOrderUseCase) Execute(orderID string) (*domain.Order, error) {
	if orderID == "" {
		return nil, fmt.Errorf("order ID is required")
	}

	order, err := uc.orderRepo.GetByID(orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		return nil, ErrOrderNotFound
	}

	return order, nil
}
