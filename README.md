# CryptoTax-Go

Backend infrastructure for a platform that prepares tax reports based on cryptocurrency transactions.

## What this system does

- Resolves asset symbols to canonical identifiers.
- Fetches historical crypto prices and converts them to fiat.
- Provides gRPC APIs for valuation and analytics.
- Emits production‑ready telemetry (metrics + logs).

## Services

- **price-svc** — gRPC service for historical price lookup and fiat valuation.
- **transactions-query-svc** — transaction analytics/querying (stub).
- **tax-svc** — tax computation logic (stub).

## Architecture

- **Transport**: gRPC + protobuf.
- **Storage**: PostgreSQL (sqlc).
- **Cache**: Redis.
- **Market data**: CoinGecko (USD prices).
- **FX rates**:
  - RUB: CBR XML
  - KZT: NBRK XML + CSV cache
- **Observability**: OpenTelemetry SDK → OTel Collector → Prometheus (metrics) + Loki (logs) → Grafana (UI).

## price-svc flow (high level)

1. Resolve asset symbols → CoinGecko IDs.
2. Fetch historical USD prices from CoinGecko.
3. Convert USD → target fiat via FX provider.
4. Multiply by transaction amounts and return values.

Key files:
- `services/price-svc/internal/server/server.go` — gRPC handlers
- `services/price-svc/internal/usecases/historical_price_usecase.go` — pricing logic
- `services/price-svc/internal/fiatfx` — FX sources and provider
- `services/price-svc/internal/domain/error` — domain error model

## Observability (MVP)

### Metrics
- gRPC server metrics via OTel semconv:
  - `rpc.server.call.duration` (histogram; used for p50/p90/p99)
  - `rpc_server_call_duration_count` (requests_total)
  - attributes: `rpc.method`, `rpc.service`, `rpc.response.status_code`
- Go runtime/process metrics via OTel runtime instrumentation:
  - goroutines, memory usage, GC goal
  - GC pauses/count via `OTEL_GO_X_DEPRECATED_RUNTIME_METRICS=true`
- **No high‑cardinality labels** in metrics.

### Logs
- zap JSON logs to stdout.
- Dual‑write to OTel Logs (OTLP) → Collector → Loki.
- One access log per RPC with duration and status code.
- Panic recovery logs stacktrace once (server‑side).

Grafana dashboards and Prometheus alert rules are provisioned automatically.

## Configuration

### YAML config
`services/price-svc/config.yaml`:
- gRPC address
- CoinGecko settings (rate limit, granularity policy)
- Resolver settings (`assets.yaml`)
- Postgres / Redis pool settings

### Environment variables
Required:
- `APP_VERSION`
- `DATABASE_URL`
- `REDIS_URL`
- `COINGECKO_API_KEY`

Optional:
- `APP_ENV` (default: `prod`)
- `OTEL_EXPORTER_OTLP_ENDPOINT` (default: `otel-collector:4317`)
- `OTEL_EXPORTER_OTLP_INSECURE` (default: `true`)
- `OTEL_GO_X_DEPRECATED_RUNTIME_METRICS` (set to `true` for GC pause/count metrics)

## Running locally (Docker Compose)

```sh
docker compose up --build
```

If you do not have Postgres/Redis inside compose, provide them via env:

```sh
DATABASE_URL=postgres://user:pass@host:5432/price?sslmode=disable
REDIS_URL=redis://host:6379
APP_VERSION=0.0.0
```

Ports:
- price-svc: `localhost:8093`
- Prometheus: `localhost:9090`
- Grafana: `localhost:3000` (admin/admin)
- Loki: `localhost:3100`

## Development

```sh
cd services/price-svc

go test ./...
```

## API

Proto definitions live in `api/proto/price/v1/price.proto` and generated code in `gen/`.

## Troubleshooting

- **No metrics in Prometheus**: check `otel-collector` container is up and scrape target is healthy.
- **No logs in Loki**: verify `OTEL_EXPORTER_OTLP_ENDPOINT` and that logs pipeline is enabled in collector.
- **High error rate**: check access logs in Loki for `rpc.response.status_code`.

## Roadmap

- Distributed tracing (OTel traces + trace_id/span_id in logs)
- Host/CPU metrics via collector receivers
- Full tax computation engine and report generation
