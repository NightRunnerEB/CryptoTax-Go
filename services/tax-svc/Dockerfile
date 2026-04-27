# syntax=docker/dockerfile:1.7
FROM golang:1.25.1 AS build

WORKDIR /workspace
ENV CGO_ENABLED=0
ENV GOWORK=off

COPY go.work go.work.sum ./
COPY gen/go.mod gen/go.sum ./gen/
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/tax-svc/go.mod services/tax-svc/go.sum ./services/tax-svc/

WORKDIR /workspace/services/tax-svc
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY gen /workspace/gen
COPY pkg /workspace/pkg
COPY services/tax-svc /workspace/services/tax-svc

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /out/tax-svc ./cmd/app

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app

COPY --from=build /out/tax-svc /app/tax-svc
COPY services/tax-svc/config.yaml /app/config.yaml
USER nonroot:nonroot
EXPOSE 8096 8097
ENTRYPOINT ["/app/tax-svc"]
