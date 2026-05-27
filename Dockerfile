FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata sqlite-libs wget

WORKDIR /app
COPY --from=builder /app/server .
COPY config.yaml .
COPY internal/database/schema.sql ./internal/database/

RUN mkdir -p data && addgroup -S portal && adduser -S -G portal portal && chown portal:portal /app/data

EXPOSE 8081

USER portal

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8081/api/health || exit 1

CMD ["./server"]
