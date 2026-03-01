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
	headerRoles    = "x-roles"
)

func requireTenantHeader(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return apperr.InvalidArgument("missing headers", nil, apperr.FieldViolation{
			Field:       "x-tenant-id",
			Description: "required",
		})
	}
	values := md.Get(headerTenantID)
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		return apperr.InvalidArgument("missing tenant header", nil, apperr.FieldViolation{
			Field:       "x-tenant-id",
			Description: "required",
		})
	}
	return nil
}

func requireTenantHeaderMatch(ctx context.Context, tenantID uuid.UUID) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return apperr.InvalidArgument("missing headers", nil, apperr.FieldViolation{
			Field:       "x-tenant-id",
			Description: "required",
		})
	}

	values := md.Get(headerTenantID)
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		return apperr.InvalidArgument("missing tenant header", nil, apperr.FieldViolation{
			Field:       "x-tenant-id",
			Description: "required",
		})
	}

	headerTenantID := strings.TrimSpace(values[0])
	headerTenantUUID, err := uuid.Parse(headerTenantID)
	if err != nil {
		return apperr.InvalidArgument("invalid tenant header", err, apperr.FieldViolation{
			Field:       "x-tenant-id",
			Description: "invalid uuid",
		})
	}
	if headerTenantUUID != tenantID {
		return apperr.InvalidArgument("tenant mismatch", nil, apperr.FieldViolation{
			Field:       "tenant_id",
			Description: "must match x-tenant-id header",
		})
	}

	return nil
}
