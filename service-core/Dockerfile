FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.work ./
COPY query/ ./query/
COPY service-core/go.mod service-core/go.sum ./service-core/
RUN cd service-core && go mod download
COPY service-core/ ./service-core/
RUN CGO_ENABLED=0 go build -C service-core -o /service-core .

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /service-core .
EXPOSE 8080
CMD ["./service-core"]
