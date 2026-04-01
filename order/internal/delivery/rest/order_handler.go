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

type CreateOrderRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	ItemName   string `json:"item_name" binding:"required"`
	Amount     int64  `json:"amount" binding:"required,gt=0"`
}

type CreateOrderResponse struct {
	Order   interface{} `json:"order"`
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

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

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	input := usecase.CreateOrderInput{
		CustomerID: req.CustomerID,
		ItemName:   req.ItemName,
		Amount:     req.Amount,
	}

	output, err := h.createOrderUseCase.Execute(input)

	if err != nil {
		if err.Error() == "payment service unavailable: context deadline exceeded" ||
			err.Error() == "payment service unavailable: dial tcp" {
			c.JSON(http.StatusServiceUnavailable, CreateOrderResponse{
				Order:   output.Order,
				Status:  "pending",
				Message: "Order created but payment service is unavailable. Please try again later.",
			})
			return
		}

		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, CreateOrderResponse{
		Order:   output.Order,
		Status:  output.Status,
		Message: output.Message,
	})
}

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
