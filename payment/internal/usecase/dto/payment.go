package dto

type AuthorizePaymentInput struct {
	OrderID string
	Amount  int64
}

type AuthorizePaymentOutput struct {
	TransactionID string
	Status        string
	Message       string
}
