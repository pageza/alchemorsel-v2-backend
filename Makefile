.PHONY: build run test lint clean debug debug-remote

# Build the application
build:
	go build -o bin/api ./cmd/api

# Run the application
run:
	go run ./cmd/api

# Run tests with required environment variables
test:
	DEEPSEEK_API_KEY="$$(cat ../secrets/deepseek_api_key.txt 2>/dev/null || echo 'test-key')" \
	OPENAI_API_KEY="$$(cat ../secrets/openai_api_key.txt 2>/dev/null || echo 'test-key')" \
	go test -v ./...

# Run tests without API key requirements (faster for development)
test-quick:
	go test -v ./... -short

# Run linter
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Install development tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/cosmtrek/air@latest
	go install github.com/go-delve/delve/cmd/dlv@latest

# Run with hot reload using Air
dev:
	air -c .air.toml

# Run with debugger
debug:
	dlv debug ./cmd/api --headless --listen=:2345 --api-version=2 --accept-multiclient

# Run with remote debugger
debug-remote:
	dlv debug ./cmd/api --headless --listen=:2345 --api-version=2 --accept-multiclient --continue 