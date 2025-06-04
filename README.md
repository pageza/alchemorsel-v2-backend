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
# API authentication (one of these is required)
DEEPSEEK_API_KEY=your_deepseek_key
DEEPSEEK_API_KEY_FILE=path/to/keyfile
DEEPSEEK_API_URL=https://api.deepseek.com/v1/chat/completions
# S3 configuration for profile pictures
AWS_REGION=us-east-1
S3_BUCKET_NAME=alchemorsel-profile-pictures
```

3. Run the application:
```bash
go run ./cmd/api
```

## Development

- The server runs on `http://localhost:8080` by default
- Hot reload is enabled using `air` (optional)
- API documentation is available at `/swagger` when running in development mode

## Project Structure

```
backend/
├── cmd/
│   └── api/         # Application entry point
├── config/          # Configuration helpers
├── internal/        # Private application code
│   ├── api/         # HTTP handlers
│   ├── database/    # Database utilities
│   ├── middleware/  # HTTP middleware
│   ├── model/       # Recipe models
│   ├── models/      # User and profile models
│   ├── server/      # Server setup
│   └── service/     # Business logic
├── migrations/      # Database migrations
└── scripts/         # Utility scripts
```

## Available Commands

- `go run ./cmd/api` - Run the application
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
- `POST /api/v1/recipes/:id/favorite` - add a recipe to the authenticated user's favorites
- `DELETE /api/v1/recipes/:id/favorite` - remove a recipe from the authenticated user's favorites

Favorites are stored in the `recipe_favorites` table created by the database migrations.

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