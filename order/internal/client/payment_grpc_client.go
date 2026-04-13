package client

import (
	"context"
	"fmt"
	"time"

	paymentpb "github.com/IK-akx/ap2-generated/payment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PaymentGrpcClient struct {
	conn   *grpc.ClientConn
	client paymentpb.PaymentServiceClient
}

func NewPaymentGrpcClient(address string, timeout time.Duration) (*PaymentGrpcClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to payment service: %w", err)
	}

	client := paymentpb.NewPaymentServiceClient(conn)

	return &PaymentGrpcClient{
		conn:   conn,
		client: client,
	}, nil
}

func (c *PaymentGrpcClient) AuthorizePayment(orderID string, amount int64) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := c.client.ProcessPayment(ctx, &paymentpb.PaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	})
	if err != nil {
		return "", "", fmt.Errorf("payment service unavailable: %w", err)
	}

	return resp.GetTransactionId(), resp.GetStatus(), nil
}
