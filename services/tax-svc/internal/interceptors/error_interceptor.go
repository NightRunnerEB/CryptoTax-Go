package interceptors

import (
	"context"

	"google.golang.org/grpc"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/grpcerr"
)

func ErrorInterceptor(domain string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}
		return nil, grpcerr.ToGRPCStatus(err, domain)
	}
}
