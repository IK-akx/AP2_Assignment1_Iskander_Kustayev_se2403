package rest

import (
	"errors"
	"net/http"
	"order/internal/cache"
	"order/internal/constant"
	"order/internal/delivery/rest/dto"
	"order/internal/domain"
	"order/internal/middleware"
	"order/internal/usecase"
	dto2 "order/internal/usecase/dto"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	OrderUsecase usecase.OrderUsecase
	OrderCache   *cache.OrderCache
}

func NewOrderHandler(
	orderUsecase usecase.OrderUsecase,
	orderCache *cache.OrderCache,
) *OrderHandler {
	return &OrderHandler{
		OrderUsecase: orderUsecase,
		OrderCache:   orderCache,
	}
}

// ApplyRateLimiter applies rate limiting to routes
func ApplyRateLimiter(router *gin.Engine, limiter *cache.RateLimiter) {
	// Create middleware with IP-based rate limiting
	rateLimiterMiddleware := middleware.RateLimiterMiddleware(
		limiter,
		middleware.GetIdentifierByIP, // or GetIdentifierByUserID
	)

	// Apply to all routes or specific ones
	router.Use(rateLimiterMiddleware)
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

// GetOrder - с поддержкой кэша (Cache-Aside Pattern)
func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("id")

	if orderID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "order ID is required"})
		return
	}

	// 1. Проверяем кэш
	var order *domain.Order
	var err error

	if h.OrderCache != nil {
		order, err = h.OrderCache.Get(orderID)
		if err != nil {
			// Логируем ошибку кэша, но продолжаем (fallback to DB)
			c.Error(err)
		}
	}

	// 2. Если в кэше нет - идем в БД
	if order == nil {
		order, err = h.OrderUsecase.GetOrderByOrderID(orderID)

		if err != nil {
			if errors.Is(err, constant.ErrOrderNotFound) {
				c.JSON(http.StatusNotFound, ErrorResponse{Error: "order not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
			return
		}

		// 3. Сохраняем в кэш (асинхронно, не блокируем ответ)
		if h.OrderCache != nil {
			go func() {
				if err := h.OrderCache.Set(order); err != nil {
					c.Error(err)
				}
			}()
		}
	}

	c.JSON(http.StatusOK, order)
}

// CancelOrder - добавить инвалидацию кэша
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

	// Инвалидируем кэш после успешной отмены
	if h.OrderCache != nil {
		go func() {
			if err := h.OrderCache.Delete(orderID); err != nil {
				c.Error(err)
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "order cancelled successfully",
		"order_id": orderID,
		"status":   "cancelled",
	})
}
