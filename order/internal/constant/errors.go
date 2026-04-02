package constant

import "errors"

var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrInvalidOrderStatus = errors.New("invalid order status for this operation")
	ErrPaymentFailed      = errors.New("payment authorization failed")
	ErrPaymentServiceDown = errors.New("payment service is unavailable")
)
