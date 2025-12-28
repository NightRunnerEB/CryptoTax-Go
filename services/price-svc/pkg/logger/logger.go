package logger

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

type Logger interface {
	Debug(message interface{}, args ...interface{})
	Info(message string, args ...interface{})
	Warn(message string, args ...interface{})
	Error(message interface{}, args ...interface{})
	Fatal(message interface{}, args ...interface{})
}

type ZeroLogger struct {
	logger *zerolog.Logger
}

var _ Logger = (*ZeroLogger)(nil)

// New -.
func New(level string) *ZeroLogger {
	var l zerolog.Level

	switch strings.ToLower(level) {
	case "error":
		l = zerolog.ErrorLevel
	case "warn":
		l = zerolog.WarnLevel
	case "info":
		l = zerolog.InfoLevel
	case "debug":
		l = zerolog.DebugLevel
	default:
		l = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(l)

	skipFrameCount := 3
	logger := zerolog.New(os.Stdout).With().Timestamp().CallerWithSkipFrameCount(zerolog.CallerSkipFrameCount + skipFrameCount).Logger()

	return &ZeroLogger{
		logger: &logger,
	}
}

func (l *ZeroLogger) Debug(message interface{}, args ...interface{}) {
	l.msg(zerolog.DebugLevel, message, args...)
}

func (l *ZeroLogger) Info(message string, args ...interface{}) {
	l.log(zerolog.InfoLevel, message, args...)
}

func (l *ZeroLogger) Warn(message string, args ...interface{}) {
	l.log(zerolog.WarnLevel, message, args...)
}

func (l *ZeroLogger) Error(message interface{}, args ...interface{}) {
	if l.logger.GetLevel() == zerolog.DebugLevel {
		l.Debug(message, args...)
	}

	l.msg(zerolog.ErrorLevel, message, args...)
}

func (l *ZeroLogger) Fatal(message interface{}, args ...interface{}) {
	l.msg(zerolog.FatalLevel, message, args...)

	os.Exit(1)
}

func (l *ZeroLogger) log(level zerolog.Level, message string, args ...interface{}) {
	if len(args) == 0 {
		l.logger.WithLevel(level).Msg(message)
	} else {
		l.logger.WithLevel(level).Msgf(message, args...)
	}
}

func (l *ZeroLogger) msg(level zerolog.Level, message interface{}, args ...interface{}) {
	switch msg := message.(type) {
	case error:
		l.log(level, msg.Error(), args...)
	case string:
		l.log(level, msg, args...)
	default:
		l.log(level, fmt.Sprintf("%s message %v has unknown type %v", level, message, msg), args...)
	}
}
