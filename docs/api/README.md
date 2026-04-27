# CryptoTax REST API Contracts

This directory contains hand-maintained OpenAPI contracts for **REST** endpoints only.

- `openapi.yaml` — unified contract with **all** REST methods (frontend + internal).

## Scope

OpenAPI is intentionally limited to REST. Service-to-service gRPC contracts remain in:

- `api/proto/aggregation/v1/aggregation.proto`
- `api/proto/tax/v1/tax.proto`
- `api/proto/price/v1/price.proto`
- `api/proto/report/v1/report.proto`

`auth-svc` and `ledger-svc` source routes/status mappings live in the Rust repository
`../CryptoTax`. Their endpoints are documented in `openapi.yaml` and synchronized manually.

## Security model

- JWT is validated at Nginx.
- Backend services require `X-User-Id` and validate user consistency for user-scoped endpoints.
- Optional headers: `X-User-Id`, `X-Roles`.

## Error model

- gRPC gateway endpoints use `{ code, message, details[] }` with google.rpc details
  (`ErrorInfo`, `BadRequest`, `ResourceInfo`).
- Native Rust Axum endpoints (`auth-svc`, `ledger-svc`) use service-specific JSON errors
  as implemented in `../CryptoTax`.
