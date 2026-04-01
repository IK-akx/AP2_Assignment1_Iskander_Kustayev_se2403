package usecase

import (
	"fmt"
	"order/internal/domain"
)

type CancelOrderUseCase struct {
	orderRepo domain.OrderRepository
}

func NewCancelOrderUseCase(orderRepo domain.OrderRepository) *CancelOrderUseCase {
	return &CancelOrderUseCase{
		orderRepo: orderRepo,
	}
}

func (uc *CancelOrderUseCase) Execute(orderID string) error {
	if orderID == "" {
		return fmt.Errorf("order ID is required")
	}

	order, err := uc.orderRepo.GetByID(orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		return ErrOrderNotFound
	}

	// Business rule: Only Pending orders can be cancelled
	if order.Status != domain.OrderStatusPending {
		return fmt.Errorf("%w: order status is %s, cannot cancel", ErrInvalidOrderStatus, order.Status)
	}

	// Update order status to Cancelled
	if err := uc.orderRepo.UpdateStatus(orderID, domain.OrderStatusCancelled); err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	return nil
}
