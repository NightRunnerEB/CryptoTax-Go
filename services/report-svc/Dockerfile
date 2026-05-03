# syntax=docker/dockerfile:1.7
FROM golang:1.25.1 AS build

WORKDIR /workspace
ENV CGO_ENABLED=0
ENV GOWORK=off

COPY go.work go.work.sum ./
COPY gen/go.mod gen/go.sum ./gen/
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/report-svc/go.mod services/report-svc/go.sum ./services/report-svc/

WORKDIR /workspace/services/report-svc
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY gen /workspace/gen
COPY pkg /workspace/pkg
COPY services/report-svc /workspace/services/report-svc

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /out/report-svc ./cmd/app

FROM debian:bookworm-slim
WORKDIR /app

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates libxml2-utils \
	&& rm -rf /var/lib/apt/lists/*
RUN useradd --system --uid 10001 --create-home appuser

COPY --from=build /out/report-svc /app/report-svc
COPY --from=build /workspace/services/report-svc/NDFL3.xsd.xml /app/NDFL3.xsd.xml
COPY services/report-svc/config.yaml /app/config.yaml

USER appuser
EXPOSE 8098
ENTRYPOINT ["/app/report-svc"]
