# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN GOPROXY=https://goproxy.io,direct go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o cisco-exporter ./cmd/exporter

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /app/cisco-exporter .
COPY config.example.yaml .
COPY metric.example.yaml .

RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 9427

ENTRYPOINT ["./cisco-exporter"]
