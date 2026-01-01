# OpenAnki Backend

A self-hosted Go backend for OpenAnki, designed to run on low-cost VPS instances.

## Tech Stack
- **Language**: Go 1.21+
- **Database**: SQLite
- **Framework**: Chi (HTTP Router)

## Local Development

### Prerequisites
- Go 1.21+ (if running locally)
- Docker (recommended)

### Running with Docker (Recommended)
Since the `go` command might not be installed on your host machine, using Docker is the easiest way to run the backend.

1. Build the image:
   ```bash
   cd backend/openanki-backend
   docker build -t openanki-backend .
   ```

2. Run the container:
   ```bash
   docker run -p 8080:8080 openanki-backend
   ```
   The API will be available at `http://localhost:8080`.

### Running Locally (if Go is installed)
```bash
cd backend/openanki-backend
# Install dependencies
go mod tidy
# Run
go run cmd/server/main.go
```

## Structure
- `cmd/server`: Entry point (`main.go`)
- `internal/api`: HTTP Handlers
- `internal/database`: Database connection and queries
- `internal/auth`: Authentication logic
