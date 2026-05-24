FROM golang:1.26.1-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/go/pkg/mod

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/service ./cmd/trend-service

FROM alpine:latest
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/bin/service ./main

CMD ["./main"]

