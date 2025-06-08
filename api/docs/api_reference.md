# API Reference

This document summarizes the available endpoints of the backend service. The full machine readable specification is located at [`api/docs/openapi.yaml`](openapi.yaml).

| Method | Path | Auth | Summary |
|-------|------|------|---------|
| GET | `/health` | None | Health check |
| POST | `/api/v1/auth/register` | None | Register a new user |
| POST | `/api/v1/auth/login` | None | Login user |
| GET | `/api/v1/profile` | Bearer | Get authenticated profile |
| PUT | `/api/v1/profile` | Bearer | Update profile |
| POST | `/api/v1/profile/logout` | Bearer | Logout user |
| GET | `/api/v1/recipes` | None | List recipes |
| POST | `/api/v1/recipes` | Bearer | Create recipe |
| GET | `/api/v1/recipes/{id}` | None | Get recipe by ID |
| PUT | `/api/v1/recipes/{id}` | Bearer | Update recipe |
| DELETE | `/api/v1/recipes/{id}` | Bearer | Delete recipe |
| POST | `/api/v1/recipes/{id}/favorite` | Bearer | Favorite recipe |
| DELETE | `/api/v1/recipes/{id}/favorite` | Bearer | Remove recipe from favorites |
| POST | `/api/v1/llm/query` | Bearer | Generate recipe using LLM |

Each endpoint's request and response bodies are defined in the OpenAPI file. To explore the API interactively during development, start the server and visit `http://localhost:8080/swagger`.
