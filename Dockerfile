FROM golang:1.24.0-bullseye AS builder

WORKDIR /app

# Install build dependencies for SQLite
RUN apt-get update && apt-get install -y \
    gcc \
    libc6-dev \
    sqlite3 \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o main ./cmd/api

FROM debian:bullseye-slim

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/config ./config
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/start.sh ./start.sh

RUN chmod +x start.sh

EXPOSE 8080

CMD ["./start.sh"]     