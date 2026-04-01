package interceptors

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestAccessLogInterceptor_AddsRequestID(t *testing.T) {
	core, observed := observer.New(zapcore.DebugLevel)
	log := zap.New(core)

	interceptor := AccessLogInterceptor(log, AccessLogConfig{
		ServiceName:    "price-svc",
		ServiceVersion: "1.0.0",
		Environment:    "dev",
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-request-id", "req-123"))
	_, err := interceptor(
		ctx,
		struct{}{},
		&grpc.UnaryServerInfo{FullMethod: "/price.v1.Price/ValuateTransactionsBatch"},
		func(context.Context, any) (any, error) {
			return struct{}{}, nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := observed.All()
	if len(entries) == 0 {
		t.Fatal("expected at least one log entry")
	}

	foundRequestID := false
	for _, f := range entries[0].Context {
		if f.Key == "request_id" && f.String == "req-123" {
			foundRequestID = true
			break
		}
	}
	if !foundRequestID {
		t.Fatalf("request_id field not found in log entry: %+v", entries[0].Context)
	}
}
