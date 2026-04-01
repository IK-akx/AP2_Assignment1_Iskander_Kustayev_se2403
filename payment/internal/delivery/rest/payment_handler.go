package rest

import (
	"net/http"
	"payment/internal/usecase"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	authorizeUC  *usecase.AuthorizePaymentUseCase
	getPaymentUC *usecase.GetPaymentUseCase
}

type AuthorizePaymentRequest struct {
	OrderID string `json:"order_id" binding:"required"`
	Amount  int64  `json:"amount" binding:"required,gt=0"`
}

type AuthorizePaymentResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	Message       string `json:"message,omitempty"`
}

type GetPaymentResponse struct {
	ID            string `json:"id"`
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewPaymentHandler(
	authorizeUC *usecase.AuthorizePaymentUseCase,
	getPaymentUC *usecase.GetPaymentUseCase,
) *PaymentHandler {
	return &PaymentHandler{
		authorizeUC:  authorizeUC,
		getPaymentUC: getPaymentUC,
	}
}

func (h *PaymentHandler) AuthorizePayment(c *gin.Context) {
	var req AuthorizePaymentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	input := usecase.AuthorizePaymentInput{
		OrderID: req.OrderID,
		Amount:  req.Amount,
	}

	output, err := h.authorizeUC.Execute(input)

	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, AuthorizePaymentResponse{
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

	payment, err := h.getPaymentUC.Execute(orderID)

	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	response := GetPaymentResponse{
		ID:            payment.ID,
		OrderID:       payment.OrderID,
		TransactionID: payment.TransactionID,
		Amount:        payment.Amount,
		Status:        payment.Status,
		CreatedAt:     payment.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	c.JSON(http.StatusOK, response)
}
