package grpcerr

import (
	"errors"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
	"go.uber.org/zap"
)

func LogFields(err error) []zap.Field {
	fields := []zap.Field{
		zap.Error(err),
		// zap.Strings("cause_chain", causeChain(err)),
		// zap.Any("root_cause", rootCauseString(err)),
		// zap.String("root_cause_type", rootCauseType(err)),
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

// В данной модели ошибок цепочек вызовов ошибок не должно быть!
// func causeChain(err error) []string {
// 	var out []string
// 	for err != nil {
// 		out = append(out, err.Error())
// 		err = errors.Unwrap(err)
// 	}
// 	return out
// }

// func rootCauseString(err error) string {
// 	rc := rootCause(err)
// 	if rc == nil {
// 		return ""
// 	}
// 	return rc.Error()
// }

// func rootCauseType(err error) string {
// 	rc := rootCause(err)
// 	if rc == nil {
// 		return ""
// 	}
// 	return fmt.Sprintf("%T", rc)
// }

// func rootCause(err error) error {
// 	if err == nil {
// 		return nil
// 	}
// 	for {
// 		u := errors.Unwrap(err)
// 		if u == nil {
// 			return err
// 		}
// 		err = u
// 	}
// }
