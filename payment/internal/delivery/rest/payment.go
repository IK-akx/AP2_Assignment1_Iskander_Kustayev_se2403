package rest

import (
	"net/http"
	"payment/internal/delivery/rest/dto"
	"payment/internal/usecase"
	dto2 "payment/internal/usecase/dto"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	PaymentUsecase *usecase.PaymentUsecase
}

func NewPaymentHandler(
	paymentUsecase *usecase.PaymentUsecase,
) *PaymentHandler {
	return &PaymentHandler{
		PaymentUsecase: paymentUsecase,
	}
}

func (h *PaymentHandler) AuthorizePayment(c *gin.Context) {
	var req dto.AuthorizePaymentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	input := dto2.AuthorizePaymentInput{
		OrderID: req.OrderID,
		Amount:  req.Amount,
	}

	output, err := h.PaymentUsecase.AuthorizePayment(input)

	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.AuthorizePaymentResponse{
		TransactionID: output.TransactionID,
		Status:        output.Status,
		Message:       output.Message,
	})
}

func (h *PaymentHandler) GetPayment(c *gin.Context) {
	orderID := c.Param("order_id")

	if orderID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "order_id is required"})
		return
	}

	payment, err := h.PaymentUsecase.GetPaymentByOrderID(orderID)

	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	response := dto.GetPaymentResponse{
		ID:            payment.ID,
		OrderID:       payment.OrderID,
		TransactionID: payment.TransactionID,
		Amount:        payment.Amount,
		Status:        payment.Status,
		CreatedAt:     payment.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	c.JSON(http.StatusOK, response)
}
