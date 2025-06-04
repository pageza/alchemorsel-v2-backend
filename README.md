# Backend Application

This is the backend application built with Go.

## Prerequisites

- Go 1.21 or higher
- PostgreSQL 15 or higher
- Redis 7 or higher

## Getting Started

1. Install dependencies:
```bash
go mod download
```

2. Set up environment variables:
Create a `.env` file in the root directory with the following variables:
```
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=your_database
REDIS_HOST=localhost
REDIS_PORT=6379
```

3. Run the application:
```bash
go run main.go
```

## Development

- The server runs on `http://localhost:8080` by default
- Hot reload is enabled using `air` (optional)
- API documentation is available at `/swagger` when running in development mode

## Project Structure

```
backend/
├── cmd/              # Application entry points
├── internal/         # Private application code
│   ├── api/         # API handlers
│   ├── config/      # Configuration
│   ├── models/      # Data models
│   └── services/    # Business logic
├── pkg/             # Public library code
└── main.go         # Main application entry point
```

## Available Commands

- `go run main.go` - Run the application
- `go test ./...` - Run all tests
- `go mod tidy` - Clean up dependencies
- `go fmt ./...` - Format code
- `go vet ./...` - Check for common errors

## API Documentation

The API documentation is generated using Swagger/OpenAPI. To view the documentation:

1. Start the server
2. Visit `http://localhost:8080/swagger`

### Recipes Endpoint

`GET /api/v1/recipes` supports optional query parameters:

- `q` - search term matched against recipe name and description
- `category` - filter by category

### LLM Endpoint

`POST /api/v1/llm/query` generates a recipe using the language model. This route
requires a valid `Authorization` header with a bearer token. The response
includes the persisted recipe with the authenticated user ID attached.

## Contributing

1. Create a new branch for your feature
2. Make your changes
3. Run tests: `go test ./...`
4. Submit a pull request

## License

MIT 