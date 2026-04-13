package grpc

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

func LoggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {

	start := time.Now()

	resp, err := handler(ctx, req)

	duration := time.Since(start)

	log.Printf(
		"[gRPC] method=%s duration=%s error=%v",
		info.FullMethod,
		duration,
		err,
	)

	return resp, err
}
