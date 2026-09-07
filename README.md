# AlcheMorsel v2 Backend


Go API for AlcheMorsel, a functioning but incomplete AI-powered recipe application. The backend provides authentication, recipe workflows, relational and vector-backed persistence, external AI-service integration, API documentation, and automated tests.


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
