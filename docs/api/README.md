# CryptoTax REST API Contracts

This directory contains hand-maintained OpenAPI contracts for **REST** endpoints only.

- `frontend.openapi.yaml` — endpoints intended for UI/frontend traffic (via Nginx + grpc-gateway).
- `internal.openapi.yaml` — REST endpoints intended for internal platform usage.

## Scope

OpenAPI is intentionally limited to REST. Service-to-service gRPC contracts remain in:

- `api/proto/aggregation/v1/aggregation.proto`
- `api/proto/tax/v1/tax.proto`
- `api/proto/price/v1/price.proto`
- `api/proto/report/v1/report.proto`

`auth-svc` and `ledger-svc` source routes/status mappings live in the Rust repository
`../CryptoTax`. Their endpoints are documented inside these two specs (frontend/internal)
and synchronized manually.

## Security model

- JWT is validated at Nginx.
- Backend services require `X-Tenant-Id` and validate tenant consistency for tenant-scoped endpoints.
- Optional headers: `X-User-Id`, `X-Roles`.

## Error model

- gRPC gateway endpoints use `{ code, message, details[] }` with google.rpc details
  (`ErrorInfo`, `BadRequest`, `ResourceInfo`).
- Native Rust Axum endpoints (`auth-svc`, `ledger-svc`) use service-specific JSON errors
  as implemented in `../CryptoTax`.
