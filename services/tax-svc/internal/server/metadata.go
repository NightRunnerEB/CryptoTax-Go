package grpcserver

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"

	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
)

const (
	headerUserID = "x-user-id"
	headerRole   = "x-role"
)

func userIDFromHeader(ctx context.Context) (uuid.UUID, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return uuid.Nil, apperr.InvalidArgument("missing headers", nil, apperr.FieldViolation{
			Field:       "x-user-id",
			Description: "required",
		})
	}

	values := md.Get(headerUserID)
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		return uuid.Nil, apperr.InvalidArgument("missing user header", nil, apperr.FieldViolation{
			Field:       "x-user-id",
			Description: "required",
		})
	}

	headerUserID := strings.TrimSpace(values[0])
	headerUserUUID, err := uuid.Parse(headerUserID)
	if err != nil {
		return uuid.Nil, apperr.InvalidArgument("invalid user header", err, apperr.FieldViolation{
			Field:       "x-user-id",
			Description: "invalid uuid",
		})
	}

	return headerUserUUID, nil
}
