package dto

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
