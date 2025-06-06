# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Database Setup
```bash
# Reset and initialize database
docker compose down
docker volume ls 
docker volume rm sokoni_sokoni_pgadata
docker compose up -d

export DATABASE_URL="postgres://sokoni:sokoni@localhost:5432/sokoni?sslmode=disable"
migrate -path db/migrations -database "$DATABASE_URL" up
```

### Testing
```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/collector
go test ./internal/db

# Run tests with verbose output
go test -v ./...
```

### Building and Running
```bash
# Build the application
go build ./cmd/sokoni

# Run the application
go run ./cmd/sokoni

# Build for production
go build -o sokoni ./cmd/sokoni
```

## Architecture Overview

**Sokoni** is a Go application for scanning and cataloging PDF files from network shares, storing metadata in PostgreSQL.

### Core Components

- **collector**: Scans filesystem paths for PDF files, provides both batch (`Scan`) and streaming (`ScanWith`) interfaces
- **model**: Defines `FileInfo` struct representing file metadata (path, name, size, modification time)
- **db**: PostgreSQL connection management and repository pattern for file operations
- **tztime**: Timezone utilities for consistent time handling

### Database Schema

Two main tables:
- `connections`: Network share configurations (SMB/CIFS mount points)  
- `files`: PDF file metadata linked to connections

### Key Patterns

- Uses `pgx/v5` for PostgreSQL connectivity
- Repository pattern in `internal/db` for data access
- Functional approach with callback handlers (`ScanWith`)
- Environment-based configuration via `DATABASE_URL`
- Docker Compose for local PostgreSQL development
- Standard Go project layout with `cmd/` and `internal/` structure