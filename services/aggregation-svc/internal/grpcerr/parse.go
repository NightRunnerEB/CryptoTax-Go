package grpcerr

import (
	"errors"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StatusFromErrorChain walks the wrapped error chain and returns the first gRPC status found.
func StatusFromErrorChain(err error) (codes.Code, *status.Status, bool) {
	for current := err; current != nil; current = errors.Unwrap(current) {
		if st, ok := status.FromError(current); ok {
			return st.Code(), st, true
		}
	}
	return codes.OK, nil, false
}

// ErrorInfo extracts google.rpc.ErrorInfo from gRPC status details, if present.
func ErrorInfo(st *status.Status) *errdetails.ErrorInfo {
	if st == nil {
		return nil
	}
	for _, detail := range st.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if ok {
			return info
		}
	}
	return nil
}
