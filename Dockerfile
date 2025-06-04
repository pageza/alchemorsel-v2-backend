FROM golang:1.23-bullseye AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -v -o main ./cmd/api

# Verify the binary exists
RUN ls -la main

FROM debian:bullseye-slim

WORKDIR /app

# Install CA certificates for HTTPS requests
RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates

# Copy the binary and other necessary files
COPY --from=builder /app/main .
COPY --from=builder /app/config ./config
COPY --from=builder /app/migrations ./migrations

# Verify files in the final image
RUN ls -la

EXPOSE 8000

CMD ["./main"] 