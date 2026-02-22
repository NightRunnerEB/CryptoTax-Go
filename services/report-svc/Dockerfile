FROM golang:1.25.1 AS build

WORKDIR /workspace
ENV CGO_ENABLED=0
ENV GOWORK=/workspace/go.work

COPY go.work go.work.sum ./
COPY gen/go.mod gen/go.sum ./gen/
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/aggregation-svc/go.mod ./services/aggregation-svc/
COPY services/price-svc/go.mod ./services/price-svc/
COPY services/tax-svc/go.mod ./services/tax-svc/
COPY services/report-svc/go.mod ./services/report-svc/

WORKDIR /workspace/services/report-svc
RUN go mod download

COPY pkg ./pkg
COPY services/report-svc ./services/report-svc

RUN go build -o /out/report-svc ./cmd/app

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app

COPY --from=build /out/report-svc /app/report-svc
USER nonroot:nonroot
EXPOSE 8098
ENTRYPOINT ["/app/report-svc"]
