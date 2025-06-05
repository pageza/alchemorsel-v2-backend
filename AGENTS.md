# AGENTS.md

## 📦 Project Overview
- Language: Go (Golang)
- Minimum Go Version: 1.20
- Module Path: github.com/your-org/your-project
- Build Tool: go build
- Dependency Management: Go Modules
- Continuous Integration: GitHub Actions
- Deployment: Docker, Kubernetes

## 🎯 Coding Standards
- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines.
- Use `gofmt` for code formatting.
- Enforce `golint` for linting.
- Maintain idiomatic Go naming conventions.
- Limit cyclomatic complexity to 15 per function.

## 📁 Project Structure
- `/cmd`: Application entry points.
- `/internal`: Private application and library code.
- `/pkg`: Public libraries intended for use by external applications.
- `/api`: API definitions and related files.
- `/configs`: Configuration files.
- `/scripts`: Scripts for automation and tooling.
- `/build`: Packaging and Continuous Integration.

## 🧪 Testing Guidelines
- Use `testing` package for unit tests.
- Place test files in the same package with `_test.go` suffix.
- Achieve minimum 90% code coverage.
- Use `go test -race` to detect race conditions.
- Employ `testify` for assertions and mocks.

## 🔐 Security Practices
- Regularly run `gosec` to detect security issues.
- Avoid hardcoding secrets; use environment variables or secret management tools.
- Validate all inputs to prevent injection attacks.
- Use HTTPS for all external communications.
- Keep dependencies up to date to mitigate known vulnerabilities.

## 🛠️ Build Instructions
- Compile the application using:

```go build -o bin/your-app ./cmd/your-app```
- For cross-compilation:

```GOOS=linux GOARCH=amd64 go build -o bin/your-app-linux ./cmd/your-app```


## 🐳 Docker Configuration
- Base Image: `golang:1.20-alpine`
- Multi-stage builds to minimize image size.
- Expose necessary ports (e.g., 8080).
- Use non-root user for running the application.

## 🚀 Deployment
- Deploy using Kubernetes manifests located in `/deploy/k8s`.
- Use Helm charts for templating and managing deployments.
- Store secrets in Kubernetes Secrets or use external secret managers.

## 📈 Monitoring & Logging
- Integrate with Prometheus for metrics collection.
- Use structured logging with `logrus` or `zap`.
- Centralize logs using ELK stack or similar solutions.

## 📄 Documentation
- Generate documentation using `godoc`.
- Maintain API documentation with Swagger/OpenAPI in `/api/docs`.
- Update README.md with setup and usage instructions.

## 🔄 Continuous Integration/Continuous Deployment
- Use GitHub Actions workflows defined in `.github/workflows/`.
- On pull requests:
- Run tests and linters.
- Build Docker image.
- On merge to `main`:
- Push Docker image to registry.
- Deploy to staging environment.

## 🧰 Tooling
- Code Formatting: `gofmt`
- Linting: `golint`, `staticcheck`
- Dependency Management: `go mod`
- Security Scanning: `gosec`
- Testing: `go test`, `testify`

## 📝 Pull Request Guidelines
- Title format: `[Component] Brief description`
- Include:
- Summary of changes.
- Related issues.
- Testing performed.
- Any breaking changes.

## 👥 Code Review Process
- At least one approval required before merging.
- Use GitHub's review features for comments and suggestions.
- Discuss significant changes in team meetings.

## 📅 Release Management
- Follow Semantic Versioning: MAJOR.MINOR.PATCH.
- Tag releases in Git with annotated tags.
- Update CHANGELOG.md with each release.

## 🧾 License
- This project is licensed under the MIT License. See LICENSE file for details.

