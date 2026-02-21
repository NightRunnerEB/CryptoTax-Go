package interceptors

import (
	"context"
	"strings"
	"time"

	"github.com/NightRunner/CryptoTax-Go/pkg/logger"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/grpcerr"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type LogConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
}

func LogInterceptor(log *zap.Logger, cfg LogConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		baseFields := make([]zap.Field, 0, 10)
		baseFields = append(baseFields,
			zap.String("rpc.system.name", "grpc"),
			zap.String("rpc.method", info.FullMethod),
		)

		if svc, method := splitFullMethod(info.FullMethod); svc != "" {
			baseFields = append(baseFields,
				zap.String("rpc.service", svc),
				zap.String("rpc.method_name", method),
			)
		}

		if cfg.ServiceName != "" {
			baseFields = append(baseFields, zap.String("service.name", cfg.ServiceName))
		}
		if cfg.ServiceVersion != "" {
			baseFields = append(baseFields, zap.String("service.version", cfg.ServiceVersion))
		}
		if cfg.Environment != "" {
			baseFields = append(baseFields, zap.String("deployment.environment", cfg.Environment))
		}

		if requestID := requestIDFromContext(ctx); requestID != "" {
			baseFields = append(baseFields, zap.String("request_id", requestID))
		}

		if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
			baseFields = append(baseFields,
				zap.String("trace_id", sc.TraceID().String()),
				zap.String("span_id", sc.SpanID().String()),
			)
		}

		reqLogger := log.With(baseFields...)
		ctx = logger.WithContext(ctx, reqLogger)

		resp, err := handler(ctx, req)

		durationMs := time.Since(start).Milliseconds()
		fields := []zap.Field{
			zap.Int64("duration_ms", durationMs),
		}

		code := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				code = st.Code()
			} else {
				code = codes.Unknown
			}
			fields = append(fields, grpcerr.LogFields(err)...)
		}
		fields = append(fields, zap.String("rpc.response.status_code", code.String()))

		switch {
		case err == nil:
			reqLogger.Info("grpc request", fields...)
		case code == codes.Internal:
			reqLogger.Error("grpc request", fields...)
		default:
			reqLogger.Warn("grpc request", fields...)
		}

		return resp, err
	}
}

func splitFullMethod(fullMethod string) (string, string) {
	fullMethod = strings.TrimPrefix(fullMethod, "/")
	parts := strings.Split(fullMethod, "/")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func requestIDFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	for _, key := range []string{"x-request-id", "request-id"} {
		if values := md.Get(key); len(values) > 0 {
			return values[0]
		}
	}
	return ""
}
