package grpcserver

import (
	"context"

	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
	"google.golang.org/grpc/metadata"
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
	if len(md.Get(headerTenantID)) == 0 {
		return apperr.InvalidArgument("missing tenant header", nil, apperr.FieldViolation{
			Field:       "x-tenant-id",
			Description: "required",
		})
	}
	return nil
}
