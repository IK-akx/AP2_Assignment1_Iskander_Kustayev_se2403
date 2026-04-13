package grpc

import (
	orderpb "github.com/IK-akx/ap2-generated/order"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderGrpcHandler struct {
	orderpb.UnimplementedOrderTrackingServiceServer
	notifier *Notifier
}

func NewOrderGrpcHandler(notifier *Notifier) *OrderGrpcHandler {
	return &OrderGrpcHandler{
		notifier: notifier,
	}
}

func (h *OrderGrpcHandler) SubscribeToOrderUpdates(
	req *orderpb.OrderRequest,
	stream orderpb.OrderTrackingService_SubscribeToOrderUpdatesServer,
) error {

	orderID := req.GetOrderId()
	ch := h.notifier.Subscribe(orderID)

	for {
		select {
		case update := <-ch:
			err := stream.Send(&orderpb.OrderStatusUpdate{
				OrderId:   update.OrderID,
				OldStatus: update.OldStatus,
				NewStatus: update.NewStatus,
				Message:   update.Message,
				UpdatedAt: timestamppb.Now(),
			})
			if err != nil {
				return err
			}

		case <-stream.Context().Done():
			return nil
		}
	}
}
