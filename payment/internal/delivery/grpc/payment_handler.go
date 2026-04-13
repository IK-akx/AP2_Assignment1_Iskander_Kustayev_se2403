package grpc

import (
	"context"

	paymentpb "github.com/IK-akx/ap2-generated/payment"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"payment/internal/usecase"
	ucdto "payment/internal/usecase/dto"
)

type PaymentGrpcHandler struct {
	paymentpb.UnimplementedPaymentServiceServer
	paymentUsecase *usecase.PaymentUsecase
}

func NewPaymentGrpcHandler(paymentUsecase *usecase.PaymentUsecase) *PaymentGrpcHandler {
	return &PaymentGrpcHandler{
		paymentUsecase: paymentUsecase,
	}
}

func (h *PaymentGrpcHandler) ProcessPayment(
	ctx context.Context,
	req *paymentpb.PaymentRequest,
) (*paymentpb.PaymentResponse, error) {

	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	if req.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be greater than 0")
	}

	output, err := h.paymentUsecase.AuthorizePayment(ucdto.AuthorizePaymentInput{
		OrderID: req.GetOrderId(),
		Amount:  req.GetAmount(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &paymentpb.PaymentResponse{
		TransactionId: output.TransactionID,
		Status:        output.Status,
		Message:       output.Message,
		CreatedAt:     timestamppb.Now(),
	}, nil
}
