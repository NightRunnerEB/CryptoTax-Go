package logger

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

type ctxKey struct{}

type Option func(*options)

type options struct {
	extraCores []zapcore.Core
}

func WithCore(core zapcore.Core) Option {
	if core == nil {
		return func(*options) {}
	}
	return func(o *options) {
		o.extraCores = append(o.extraCores, core)
	}
}

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

func NewLogger(level string, env string, opts ...Option) (*zap.Logger, error) {
	var cfg zapcore.EncoderConfig
	if env == "dev" {
		cfg = zap.NewDevelopmentEncoderConfig()
	} else {
		cfg = zap.NewProductionEncoderConfig()
	}

	cfg.TimeKey = "timestamp"
	cfg.LevelKey = "level"
	cfg.MessageKey = "message"
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncodeLevel = zapcore.LowercaseLevelEncoder

	encoder := zapcore.NewJSONEncoder(cfg)

	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		return zap.NewExample(), err
	}

	cores := []zapcore.Core{
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapLevel),
	}

	var options options
	for _, opt := range opts {
		opt(&options)
	}
	if len(options.extraCores) > 0 {
		cores = append(cores, options.extraCores...)
	}

	return zap.New(
		zapcore.NewTee(cores...),
		zap.AddCaller(),
		zap.IncreaseLevel(zapLevel),
	), nil
}
