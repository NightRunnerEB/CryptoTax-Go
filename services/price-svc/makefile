MIGRATIONS_DIR=db/migrations

env:
	@set -a; source .env; set +a

migrate-version:
	@set -a; source .env; set +a; migrate -path $(MIGRATIONS_DIR) -database "$$DATABASE_URL" version

migrate-up:
	@set -a; source .env; set +a; migrate -verbose -path $(MIGRATIONS_DIR) -database "$$DATABASE_URL" up


migrate-down:
	@set -a; source .env; set +a; migrate -verbose -path $(MIGRATIONS_DIR) -database "$$DATABASE_URL" down -all

migrate-status:
	@set -a; source .env; set +a; migrate -path $(MIGRATIONS_DIR) -database "$$DATABASE_URL" status

gen-proto:
	protoc \
		-I api/proto \
		--go_out=internal/gen --go_opt=paths=source_relative \
		--go-grpc_out=internal/gen --go-grpc_opt=paths=source_relative \
		api/proto/price/v1/price.proto

gen-sqlc:
	sqlc generate

run-grpc:
	APP_ENV=dev \
	LOG_LEVEL=debug \
	go run ./cmd/app