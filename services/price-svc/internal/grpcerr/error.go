package grpcerr

import (
	"context"
	"errors"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
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
		return wrapStatus(err, status.New(codes.Canceled, "request canceled"))
	case errors.Is(err, context.DeadlineExceeded):
		return wrapStatus(err, status.New(codes.DeadlineExceeded, "deadline exceeded"))
	}

	var ae *apperr.Error
	if !errors.As(err, &ae) {
		return wrapStatus(err, status.New(codes.Internal, "internal error"))
	}

	grpcCode := mapCodeToGRPC(ae.Code)

	if grpcCode == codes.Internal {
		st := status.New(codes.Internal, "internal error")
		st2, err2 := st.WithDetails(&errdetails.ErrorInfo{
			Reason: string(apperr.ErrInternal),
			Domain: domain,
		})
		if err2 == nil {
			return wrapStatus(err, st2)
		}
		return wrapStatus(err, st)
	}

	st := status.New(grpcCode, ae.Msg)

	details := toDetails(ae, domain)
	if len(details) > 0 {
		if st2, err2 := st.WithDetails(details...); err2 == nil {
			return wrapStatus(err, st2)
		}
	}

	return wrapStatus(err, st)
}

type statusError struct {
	err error
	st  *status.Status
}

func (e statusError) Error() string {
	return e.st.Err().Error()
}

func (e statusError) GRPCStatus() *status.Status {
	return e.st
}

func (e statusError) Unwrap() error {
	return e.err
}

func wrapStatus(err error, st *status.Status) error {
	if st == nil {
		return err
	}
	return statusError{err: err, st: st}
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
