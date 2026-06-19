# Multi-stage build. The workspace (go.work) spans the service-core module and the
# local query module, so the build context is the repo root.
FROM golang:1.26-alpine AS base
WORKDIR /app
COPY go.work ./
COPY query/ ./query/
COPY service-core/go.mod service-core/go.sum ./service-core/
RUN cd service-core && go mod download

# dev: live-reload via air. Source is bind-mounted in docker-compose.override.yml
# (go.work + query/ + service-core/); air rebuilds on change. Runs from /app so the
# workspace resolves the local query replace.
FROM base AS dev
RUN go install github.com/air-verse/air@v1.65.3
CMD ["air", "-c", "service-core/.air.toml"]

FROM base AS builder
COPY service-core/ ./service-core/
RUN CGO_ENABLED=0 go build -C service-core -o /service-core .

# prod is the default (last) stage, so `docker build` with no target produces the
# runtime image — keeping the swarm/CI image build unchanged.
FROM alpine:3.20 AS prod
WORKDIR /app
COPY --from=builder /service-core .
EXPOSE 8080
CMD ["./service-core"]
