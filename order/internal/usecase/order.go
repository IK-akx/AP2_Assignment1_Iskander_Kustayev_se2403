package usecase

import (
	"fmt"
	"order/internal/constant"
	"order/internal/domain"
	"order/internal/usecase/dto"
	"time"
)

type OrderUsecase struct {
	OrderRepo   domain.OrderRepository
	OrderClient domain.PaymentGateway
}

func (uc *OrderUsecase) GetOrderByOrderID(orderID string) (*domain.Order, error) {
	order, err := uc.OrderRepo.GetByID(orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		return nil, constant.ErrOrderNotFound
	}

	return order, nil
}

func (uc *OrderUsecase) CreateOrder(input dto.CreateOrderInput) (*dto.CreateOrderOutput, error) {
	order := &domain.Order{
		ID:         domain.GenerateOrderID(), // Fast ID generation
		CustomerID: input.CustomerID,
		ItemName:   input.ItemName,
		Amount:     input.Amount,
		Status:     domain.OrderStatusPending,
		CreatedAt:  time.Now(),
	}

	if err := uc.OrderRepo.Create(order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	transactionID, paymentStatus, err := uc.OrderClient.AuthorizePayment(order.ID, order.Amount)

	if err != nil {
		return &dto.CreateOrderOutput{
			Order:   order,
			Status:  "pending",
			Message: "Order created but payment service is unavailable. Please try again later.",
		}, err
	}

	var finalStatus string
	var responseMessage string

	if paymentStatus == "Authorized" {
		finalStatus = domain.OrderStatusPaid
		responseMessage = fmt.Sprintf("Payment authorized successfully. Transaction ID: %s", transactionID)
	} else {
		finalStatus = domain.OrderStatusFailed
		responseMessage = "Payment declined"
	}

	if err := uc.OrderRepo.UpdateStatus(order.ID, finalStatus); err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	order.Status = finalStatus

	return &dto.CreateOrderOutput{
		Order:   order,
		Status:  finalStatus,
		Message: responseMessage,
	}, nil
}

func (uc *OrderUsecase) CancelOrder(orderID string) error {
	order, err := uc.OrderRepo.GetByID(orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		return constant.ErrOrderNotFound
	}

	// Business rule: Only Pending orders can be cancelled
	if order.Status != domain.OrderStatusPending {
		return fmt.Errorf("%w: order status is %s, cannot cancel", constant.ErrInvalidOrderStatus, order.Status)
	}

	// Update order status to Cancelled
	if err := uc.OrderRepo.UpdateStatus(orderID, domain.OrderStatusCancelled); err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	return nil
}
