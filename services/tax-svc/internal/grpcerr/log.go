package grpcerr

import (
	"errors"

	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"go.uber.org/zap"
)

func LogFields(err error) []zap.Field {
	if err == nil {
		return nil
	}

	var ae *apperr.Error
	if errors.As(err, &ae) {
		fields := []zap.Field{
			zap.String("error.code", string(ae.Code)),
			zap.String("error.message", ae.Msg),
		}
		if ae.Op != "" {
			fields = append(fields, zap.String("error.op", ae.Op))
		}
		if ae.Cause != nil {
			fields = append(fields, zap.Error(ae.Cause))
		}
		if len(ae.Meta) > 0 {
			fields = append(fields, zap.Any("error.meta", ae.Meta))
		}
		return fields
	}

	return []zap.Field{zap.Error(err)}
}
