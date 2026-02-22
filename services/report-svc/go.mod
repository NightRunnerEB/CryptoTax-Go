module github.com/NightRunner/CryptoTax-Go/services/report-svc

go 1.25.1

require (
	github.com/NightRunner/CryptoTax-Go/pkg v0.0.0
	github.com/google/uuid v1.6.0
	github.com/ilyakaznacheev/cleanenv v1.5.0
	github.com/jackc/pgx/v5 v5.8.0
	github.com/joho/godotenv v1.5.1
	github.com/minio/minio-go/v7 v7.0.97
	github.com/wagslane/go-rabbitmq v0.15.0
	github.com/klauspost/cpuid/v2 v2.2.11
	go.opentelemetry.io/contrib/bridges/otelzap v0.15.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.65.0
	go.opentelemetry.io/otel/trace v1.40.0
	go.uber.org/zap v1.27.1
	golang.org/x/crypto v0.39.0
	golang.org/x/sync v0.19.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260128011058-8636f8732409
	google.golang.org/grpc v1.78.0
	google.golang.org/protobuf v1.36.11
)

replace github.com/NightRunner/CryptoTax-Go/pkg => ../../pkg
