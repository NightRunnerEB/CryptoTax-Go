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

gen-sqlc:
	sqlc generate

run-grpc:
	APP_ENV=dev \
	LOG_LEVEL=debug \
	go run ./cmd/app

