# CryptoTax Platform — C4 (L1 + L2)

## L1 — System Context

```mermaid
flowchart LR
  user["User<br/>(Taxpayer / Operator)"]

  subgraph ct["CryptoTax Platform"]
    platform["CryptoTax Platform<br/>Tax accounting and reporting system"]
  end

  coingecko["CoinGecko API"]
  smtp["SMTP Provider"]
  pg["PostgreSQL"]
  redis["Redis"]
  rabbit["RabbitMQ"]
  s3["S3-Compatible Object Storage"]
  grafana["Grafana / Observability Stack"]

  user -->|"HTTPS REST"| platform
  platform -->|"HTTPS REST API"| coingecko
  platform -->|"SMTP / SMTPS"| smtp
  platform -->|"SQL (TCP)"| pg
  platform -->|"RESP (TCP)"| redis
  platform -->|"AMQP 0-9-1 (TCP)"| rabbit
  platform -->|"S3 API (HTTP/S)"| s3
  platform -->|"OTLP metrics/logs/traces"| grafana
```

## L2 — Container Diagram

```mermaid
flowchart LR
  user["User<br/>(Web Browser)"]

  subgraph edge["Edge / API Layer"]
    gateway["Istio Ingress Gateway<br/>Public HTTP entrypoint"]
    frontend["Frontend<br/>React + Vite"]
  end

  subgraph svc["CryptoTax Services"]
    auth["auth-svc<br/>Auth, JWT, profile bootstrap"]
    ledger["ledger-svc<br/>Exchange import / normalization"]
    aggregation["aggregation-svc<br/>Aggregation + enrichment"]
    price["price-svc<br/>Price/FX provider"]
    tax["tax-svc<br/>Tax engine + orchestration"]
    report["report-svc<br/>NDFL XML rendering"]
  end

  pg["PostgreSQL"]
  redis["Redis"]
  rabbit["RabbitMQ"]
  s3["S3-Compatible Object Storage"]
  coingecko["CoinGecko API"]
  smtp["SMTP Provider"]
  grafana["Grafana / Observability Stack"]

  user -->|"HTTPS"| frontend
  frontend -->|"HTTPS REST"| gateway

  gateway -->|"REST /auth/*"| auth
  gateway -->|"REST /mexc/csv, /exchanges/*"| ledger
  gateway -->|"REST /transactions, /imports/*"| aggregation
  gateway -->|"REST /tax/*"| tax

  auth -->|"HTTP REST (tax profile upsert)"| tax
  tax -->|"gRPC (calc input)"| aggregation
  tax -->|"gRPC (render report)"| report
  aggregation -->|"HTTP REST (imports/transactions)"| ledger
  aggregation -->|"gRPC (prices/fx)"| price

  price -->|"HTTPS REST"| coingecko
  auth -->|"SMTP / SMTPS"| smtp
  report -->|"S3 API (XML upload)"| s3
  tax -->|"S3 API (result/report access)"| s3

  ledger -->|"AMQP publish"| rabbit
  aggregation -->|"AMQP consume"| rabbit

  auth -->|"SQL"| pg
  ledger -->|"SQL"| pg
  aggregation -->|"SQL"| pg
  tax -->|"SQL"| pg
  price -->|"SQL"| pg

  auth -->|"RESP"| redis
  aggregation -->|"RESP"| redis
  price -->|"RESP"| redis

  auth -.->|"OTLP"| grafana
  ledger -.->|"OTLP"| grafana
  aggregation -.->|"OTLP"| grafana
  price -.->|"OTLP"| grafana
  tax -.->|"OTLP"| grafana
  report -.->|"OTLP"| grafana
```

## Scope Notes

- Диаграмма объединяет сервисы из двух репозиториев (`CryptoTax-Go` и `CryptoTax`) как единую систему.
- На L2 показаны контейнеры и внешние зависимости без детализации внутренних модулей сервисов.
- Для читаемости используются только основные пользовательские и межсервисные потоки.
