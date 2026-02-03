package logger

import (
	"context"

	"go.uber.org/zap"
)

type Logger interface {
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

type ctxKey struct{}

func WithContext(ctx context.Context, l *zap.Logger) context.Context {
	if l == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, l)
}

func FromContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return zap.NewExample()
	}
	if l, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok && l != nil {
		return l
	}
	return zap.NewExample()
}

func NewLogger(level string, env string) (*zap.Logger, error) {
	if env == "dev" {
		cfg := zap.NewDevelopmentConfig()
		cfg.DisableStacktrace = true
		return cfg.Build()
	}

	cfg := zap.NewProductionConfig()

	if err := cfg.Level.UnmarshalText([]byte(level)); err != nil {
		return zap.NewExample(), err
	}

	return cfg.Build()
}
