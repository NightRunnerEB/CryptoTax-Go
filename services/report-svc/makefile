gen-proto:
	protoc \
		-I ../../api/proto \
		--go_out=../../gen --go_opt=paths=source_relative \
		--go-grpc_out=../../gen --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=../../gen --grpc-gateway_opt=paths=source_relative \
		../../api/proto/report/v1/report.proto

run-grpc:
	APP_ENV=dev \
	LOG_LEVEL=debug \
	go run ./cmd/app
