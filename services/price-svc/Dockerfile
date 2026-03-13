FROM golang:1.25.1 AS build

WORKDIR /workspace
ENV CGO_ENABLED=0
ENV GOWORK=off
# TODO: убрать GOWORK, когда снова потребуется сборка с go.work

COPY go.work go.work.sum ./
COPY gen/go.mod gen/go.sum ./gen/
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/price-svc/go.mod services/price-svc/go.sum ./services/price-svc/

WORKDIR /workspace/services/price-svc
RUN go mod download

COPY gen /workspace/gen
COPY pkg /workspace/pkg
COPY services/price-svc /workspace/services/price-svc

RUN go build -o /out/price-svc ./cmd/app

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app

COPY --from=build /out/price-svc /app/price-svc
COPY services/price-svc/config.yaml /app/config.yaml
COPY services/price-svc/assets.yaml /app/assets.yaml
COPY --chown=nonroot:nonroot services/price-svc/KZT-USD.csv /app/KZT-USD.csv
COPY --chown=nonroot:nonroot services/price-svc/RUB-USD.csv /app/RUB-USD.csv

USER nonroot:nonroot
EXPOSE 8093
ENTRYPOINT ["/app/price-svc"]
