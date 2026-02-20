FROM golang:1.25.1 AS build

WORKDIR /workspace
ENV CGO_ENABLED=0
ENV GOWORK=/workspace/go.work

COPY go.work go.work.sum ./
COPY gen/go.mod gen/go.sum ./gen/
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/tax-svc/go.mod ./services/tax-svc/

WORKDIR /workspace/services/tax-svc
RUN go mod download

COPY gen ./gen
COPY pkg ./pkg
COPY services/tax-svc ./services/tax-svc

RUN go build -o /out/tax-svc ./cmd/app

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app

COPY --from=build /out/tax-svc /app/tax-svc
USER nonroot:nonroot
EXPOSE 8096 8097
ENTRYPOINT ["/app/tax-svc"]
