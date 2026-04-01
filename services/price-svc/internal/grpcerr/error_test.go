package grpcerr

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

func TestToGRPCStatus_InvalidArgument(t *testing.T) {
	err := apperr.InvalidArgument("bad input", nil, apperr.FieldViolation{
		Field:       "fiat_currency",
		Description: "required",
	})

	got := ToGRPCStatus(err, "price")
	st, ok := status.FromError(got)
	if !ok {
		t.Fatalf("expected grpc status error, got %T", got)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("code = %s, want %s", st.Code(), codes.InvalidArgument)
	}
}

func TestToGRPCStatus_ProviderUnavailable(t *testing.T) {
	err := apperr.ProviderUnavailable("provider down", "coingecko", errors.New("network down"), nil)

	got := ToGRPCStatus(err, "price")
	st, ok := status.FromError(got)
	if !ok {
		t.Fatalf("expected grpc status error, got %T", got)
	}
	if st.Code() != codes.Unavailable {
		t.Fatalf("code = %s, want %s", st.Code(), codes.Unavailable)
	}
}

func TestToGRPCStatus_InternalFallback(t *testing.T) {
	got := ToGRPCStatus(context.DeadlineExceeded, "price")
	st, ok := status.FromError(got)
	if !ok {
		t.Fatalf("expected grpc status error, got %T", got)
	}
	if st.Code() != codes.DeadlineExceeded {
		t.Fatalf("code = %s, want %s", st.Code(), codes.DeadlineExceeded)
	}
}

func TestToGRPCStatus_Passthrough(t *testing.T) {
	orig := status.Error(codes.NotFound, "already status")
	got := ToGRPCStatus(orig, "price")
	st, ok := status.FromError(got)
	if !ok {
		t.Fatalf("expected grpc status error, got %T", got)
	}
	if st.Code() != codes.NotFound {
		t.Fatalf("code = %s, want %s", st.Code(), codes.NotFound)
	}
}
