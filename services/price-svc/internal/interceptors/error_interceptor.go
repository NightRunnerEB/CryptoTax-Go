package interceptors

import (
	"context"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/grpcerr"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ErrorInterceptor(logger *zap.Logger, domain string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}

		fields := []zap.Field{
			zap.String("grpc.method", info.FullMethod),
		}
		fields = append(fields, grpcerr.LogFields(err)...)

		grpcErr := grpcerr.ToGRPCStatus(err, domain)
		st, _ := status.FromError(grpcErr)

		fields = append(fields, zap.String("grpc.code", st.Code().String()))

		if st.Code() == codes.Internal {
			logger.Error("request failed", fields...)
		} else {
			logger.Warn("request failed", fields...)
		}

		return nil, grpcErr
	}
}
