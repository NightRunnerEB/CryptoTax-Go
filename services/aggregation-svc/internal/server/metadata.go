package grpcserver

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"

	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
)

const (
	headerTenantID = "x-tenant-id"
	headerUserID   = "x-user-id"
	headerRole     = "x-role"
)

func requireTenantHeader(ctx context.Context) error {
	_, err := tenantIDFromHeader(ctx)
	return err
}

func tenantIDFromHeader(ctx context.Context) (uuid.UUID, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return uuid.Nil, apperr.InvalidArgument("missing headers", nil, apperr.FieldViolation{
			Field:       "x-tenant-id",
			Description: "required",
		})
	}

	values := md.Get(headerTenantID)
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		return uuid.Nil, apperr.InvalidArgument("missing tenant header", nil, apperr.FieldViolation{
			Field:       "x-tenant-id",
			Description: "required",
		})
	}

	headerTenantID := strings.TrimSpace(values[0])
	headerTenantUUID, err := uuid.Parse(headerTenantID)
	if err != nil {
		return uuid.Nil, apperr.InvalidArgument("invalid tenant header", err, apperr.FieldViolation{
			Field:       "x-tenant-id",
			Description: "invalid uuid",
		})
	}

	return headerTenantUUID, nil
}
