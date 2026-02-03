package interceptors

import (
	"context"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func LoggerInterceptor(log *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		return handler(logger.WithContext(ctx, log), req)
	}
}
