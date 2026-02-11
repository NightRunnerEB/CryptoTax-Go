package interceptors

import (
	"context"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/grpcerr"
	"google.golang.org/grpc"
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
