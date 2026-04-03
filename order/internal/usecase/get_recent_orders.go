package usecase

import (
	"fmt"
	"order/internal/domain"
)

// GetRecentOrdersInput represents the input for getting recent orders
type GetRecentOrdersInput struct {
	Limit int
}

// GetRecentOrdersOutput represents the output
type GetRecentOrdersOutput struct {
	Orders []domain.Order
	Count  int
}

// GetRecentOrdersUseCase handles the business logic for getting recent orders
type GetRecentOrdersUseCase struct {
	orderRepo domain.OrderRepository
}

// NewGetRecentOrdersUseCase creates a new instance
func NewGetRecentOrdersUseCase(orderRepo domain.OrderRepository) *GetRecentOrdersUseCase {
	return &GetRecentOrdersUseCase{
		orderRepo: orderRepo,
	}
}

// Execute retrieves recent orders with limit validation
func (uc *GetRecentOrdersUseCase) Execute(input GetRecentOrdersInput) (*GetRecentOrdersOutput, error) {
	// Validate limit
	if input.Limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than 0")
	}

	if input.Limit > 100 {
		return nil, fmt.Errorf("limit cannot exceed 100")
	}

	// Get recent orders from repository
	orders, err := uc.orderRepo.GetRecentOrders(input.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent orders: %w", err)
	}

	// If no orders found, return empty list (not error)
	if orders == nil {
		orders = []domain.Order{}
	}

	return &GetRecentOrdersOutput{
		Orders: orders,
		Count:  len(orders),
	}, nil
}
