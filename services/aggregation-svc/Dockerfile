FROM golang:1.25.1 AS build

WORKDIR /workspace
ENV CGO_ENABLED=0
ENV GOWORK=/workspace/go.work

COPY go.work go.work.sum ./
COPY gen/go.mod gen/go.sum ./gen/
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/aggregation-svc/go.mod services/aggregation-svc/go.sum ./services/aggregation-svc/

WORKDIR /workspace/services/aggregation-svc
RUN go mod download

COPY gen ./gen
COPY pkg ./pkg
COPY services/aggregation-svc ./services/aggregation-svc

RUN go build -o /out/aggregation-svc ./cmd/app

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app

COPY --from=build /out/aggregation-svc /app/aggregation-svc
COPY services/aggregation-svc/config.yaml /app/config.yaml

USER nonroot:nonroot
EXPOSE 8094 8095
ENTRYPOINT ["/app/aggregation-svc"]
