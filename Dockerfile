FROM golang:1.23 AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application and migration binary
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -v -o main ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -v -o migrate ./cmd/migrate

# Verify the binaries exist
RUN ls -la main migrate

FROM debian:bullseye-slim

WORKDIR /app

# Install CA certificates and Go for seeding and other tasks
RUN apt-get update && apt-get install -y ca-certificates golang && update-ca-certificates

# Copy go.mod and go.sum files
COPY --from=builder /app/go.mod /app/go.sum ./

# Ensure go.mod has the correct version before downloading dependencies
RUN sed -i 's/go 1.23.0/go 1.23/' go.mod && \
    sed -i '/^toolchain/d' go.mod

# Download dependencies
RUN go mod download

# Copy the binaries and other necessary files
COPY --from=builder /app/main .
COPY --from=builder /app/migrate .
COPY --from=builder /app/config ./config
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/cmd/seed_recipes ./cmd/seed_recipes

# Verify files in the final image
RUN ls -la

EXPOSE 8080

CMD ["./main"] 