# Alchemorsel v2 Docker Compose Setup

This docker-compose configuration orchestrates the complete Alchemorsel v2 application stack.

## Services

- **Frontend**: Vue 3 application with Nginx (port 3000 in production, 5173 in development)
- **Backend**: Go API server (port 8080)
- **PostgreSQL**: Database with pgvector extension (port 5432)
- **Redis**: Caching and session storage (port 6379)

## Prerequisites

1. Docker and Docker Compose installed
2. Secrets files in `./secrets/` directory:
   - `db_password.txt`
   - `jwt_secret.txt` 
   - `redis_password.txt`

## Usage

### Production
```bash
docker-compose -f docker-compose.secrets.yml up -d
```

### Development
```bash
docker-compose -f docker-compose.secrets.yml -f docker-compose.override.yml up -d
```

### Stop Services
```bash
docker-compose down
```

### View Logs
```bash
docker-compose logs -f [service_name]
```

## Accessing Services

- Frontend: http://localhost:3000 (production) or http://localhost:5173 (development)
- Backend API: http://localhost:8080
- PostgreSQL: localhost:5432
- Redis: localhost:6379

## Volumes

- `postgres_data`: PostgreSQL data persistence
- `redis_data`: Redis data persistence

## Networks

All services communicate through the `alchemorsel-network` bridge network.
