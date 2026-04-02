package rest

import (
	"errors"
	"net/http"
	"order/internal/constant"
	"order/internal/delivery/rest/dto"
	"order/internal/usecase"
	dto2 "order/internal/usecase/dto"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	OrderUsecase usecase.OrderUsecase
}

func NewOrderHandler(
	orderUsecase usecase.OrderUsecase,
) *OrderHandler {
	return &OrderHandler{
		OrderUsecase: orderUsecase,
	}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req dto.CreateOrderRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	input := dto2.CreateOrderInput{
		CustomerID: req.CustomerID,
		ItemName:   req.ItemName,
		Amount:     req.Amount,
	}

	output, err := h.OrderUsecase.CreateOrder(input)

	if err != nil {
		if err.Error() == "payment service unavailable: context deadline exceeded" ||
			err.Error() == "payment service unavailable: dial tcp" {
			c.JSON(http.StatusServiceUnavailable, dto.CreateOrderResponse{
				Order:   output.Order,
				Status:  "pending",
				Message: "Order created but payment service is unavailable. Please try again later.",
			})
			return
		}

		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.CreateOrderResponse{
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

	order, err := h.OrderUsecase.GetOrderByOrderID(orderID)

	if err != nil {
		if errors.Is(err, constant.ErrOrderNotFound) {
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

	err := h.OrderUsecase.CancelOrder(orderID)

	if err != nil {
		if errors.Is(err, constant.ErrOrderNotFound) {
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
