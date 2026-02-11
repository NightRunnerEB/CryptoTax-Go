package grpcerr

import (
	"errors"

	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
	"go.uber.org/zap"
)

func LogFields(err error) []zap.Field {
	fields := []zap.Field{
		zap.Error(err),
	}

	var ae *apperr.Error
	if errors.As(err, &ae) {
		fields = append(fields,
			zap.String("op", ae.Op),
			zap.String("error_code", string(ae.Code)),
			zap.String("message", ae.Msg),
		)

		if len(ae.Meta) > 0 {
			fields = append(fields, zap.Any("meta", ae.Meta))
		}

		if len(ae.Details) > 0 {
			fields = append(fields, zap.Any("details", ae.Details))
		}
	}

	return fields
}
