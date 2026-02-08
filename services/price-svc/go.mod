module github.com/NightRunner/CryptoTax-Go/services/price-svc

go 1.25.1

require (
	github.com/NightRunner/CryptoTax-Go/gen v0.0.0
	github.com/NightRunner/CryptoTax-Go/pkg v0.0.0
	github.com/google/uuid v1.6.0
	github.com/ilyakaznacheev/cleanenv v1.5.0
	github.com/jackc/pgx/v5 v5.8.0
	github.com/joho/godotenv v1.5.1
	github.com/shopspring/decimal v1.4.0
	go.uber.org/zap v1.27.1
	golang.org/x/net v0.47.0
	golang.org/x/sync v0.19.0
	golang.org/x/time v0.14.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260128011058-8636f8732409
	google.golang.org/grpc v1.78.0
	google.golang.org/protobuf v1.36.11
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/NightRunner/CryptoTax-Go/gen => ../../gen

replace github.com/NightRunner/CryptoTax-Go/pkg => ../../pkg

require (
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/redis/go-redis/v9 v9.17.2 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	olympos.io/encoding/edn v0.0.0-20201019073823-d3554ca0b0a3 // indirect
)
