package interceptors

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

func TestErrorInterceptor_MapsDomainError(t *testing.T) {
	interceptor := ErrorInterceptor("price")

	_, err := interceptor(
		context.Background(),
		struct{}{},
		&grpc.UnaryServerInfo{FullMethod: "/price.v1.Price/ValuateTransactionsBatch"},
		func(context.Context, any) (any, error) {
			return nil, apperr.InvalidArgument("bad input", nil, apperr.FieldViolation{
				Field:       "fiat_currency",
				Description: "required",
			})
		},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %T", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("code = %s, want %s", st.Code(), codes.InvalidArgument)
	}
}

func TestErrorInterceptor_PassesSuccess(t *testing.T) {
	interceptor := ErrorInterceptor("price")
	resp, err := interceptor(
		context.Background(),
		struct{}{},
		&grpc.UnaryServerInfo{FullMethod: "/price.v1.Price/ValuateTransactionsBatch"},
		func(context.Context, any) (any, error) {
			return "ok", nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.(string) != "ok" {
		t.Fatalf("unexpected response: %v", resp)
	}
}
