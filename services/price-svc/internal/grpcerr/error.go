package grpcerr

import (
	"context"
	"errors"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

func ToGRPCStatus(err error, domain string) error {
	if err == nil {
		return nil
	}

	if _, ok := status.FromError(err); ok {
		return err
	}

	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	}

	var ae *apperr.Error
	if !errors.As(err, &ae) {
		return status.Error(codes.Internal, "internal error")
	}

	grpcCode := mapCodeToGRPC(ae.Code)

	if grpcCode == codes.Internal {
		st := status.New(codes.Internal, "internal error")
		st2, err2 := st.WithDetails(&errdetails.ErrorInfo{
			Reason: string(apperr.ErrInternal),
			Domain: domain,
		})
		if err2 == nil {
			return st2.Err()
		}
		return st.Err()
	}

	st := status.New(grpcCode, ae.Msg)

	details := toDetails(ae, domain)
	if len(details) > 0 {
		if st2, err2 := st.WithDetails(details...); err2 == nil {
			return st2.Err()
		}
	}

	return st.Err()
}

func mapCodeToGRPC(c apperr.ErrorCode) codes.Code {
	switch c {
	case apperr.ErrInvalidArgument, apperr.ErrUnsupportedFiat, apperr.ErrUnsupportedSource:
		return codes.InvalidArgument
	case apperr.ErrNotFound:
		return codes.NotFound
	case apperr.ErrConflict:
		return codes.AlreadyExists
	case apperr.ErrProviderUnavailable, apperr.ErrFXUnavailable:
		return codes.Unavailable
	case apperr.ErrProviderBadResponse, apperr.ErrPriceUnavailable:
		return codes.Internal
	default:
		return codes.Internal
	}
}
