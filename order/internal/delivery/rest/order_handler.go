package rest

import (
	"net/http"
	"order/internal/usecase"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	createOrderUseCase *usecase.CreateOrderUseCase
	getOrderUseCase    *usecase.GetOrderUseCase
	cancelOrderUseCase *usecase.CancelOrderUseCase
}

// CreateOrderRequest represents the request body for creating an order
type CreateOrderRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	ItemName   string `json:"item_name" binding:"required"`
	Amount     int64  `json:"amount" binding:"required,gt=0"`
}

// CreateOrderResponse represents the response for order creation
type CreateOrderResponse struct {
	Order   interface{} `json:"order"`
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// NewOrderHandler creates a new OrderHandler instance
func NewOrderHandler(
	createUC *usecase.CreateOrderUseCase,
	getUC *usecase.GetOrderUseCase,
	cancelUC *usecase.CancelOrderUseCase,
) *OrderHandler {
	return &OrderHandler{
		createOrderUseCase: createUC,
		getOrderUseCase:    getUC,
		cancelOrderUseCase: cancelUC,
	}
}

// CreateOrder handles POST /orders
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest

	// Parse and validate request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Execute use case
	input := usecase.CreateOrderInput{
		CustomerID: req.CustomerID,
		ItemName:   req.ItemName,
		Amount:     req.Amount,
	}

	output, err := h.createOrderUseCase.Execute(input)

	if err != nil {
		// Check if it's a payment service error (503 case)
		if err.Error() == "payment service unavailable: context deadline exceeded" ||
			err.Error() == "payment service unavailable: dial tcp" {
			// Payment service is unavailable, but order is created with Pending status
			c.JSON(http.StatusServiceUnavailable, CreateOrderResponse{
				Order:   output.Order,
				Status:  "pending",
				Message: "Order created but payment service is unavailable. Please try again later.",
			})
			return
		}

		// Other errors
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Success response
	c.JSON(http.StatusOK, CreateOrderResponse{
		Order:   output.Order,
		Status:  output.Status,
		Message: output.Message,
	})
}

// GetOrder handles GET /orders/:id
func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("id")

	if orderID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "order ID is required"})
		return
	}

	order, err := h.getOrderUseCase.Execute(orderID)

	if err != nil {
		if err == usecase.ErrOrderNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

// CancelOrder handles PATCH /orders/:id/cancel
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")

	if orderID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "order ID is required"})
		return
	}

	err := h.cancelOrderUseCase.Execute(orderID)

	if err != nil {
		if err == usecase.ErrOrderNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "order not found"})
			return
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "order cancelled successfully",
		"order_id": orderID,
		"status":   "cancelled",
	})
}
