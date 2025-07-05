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
go test ./internal/service
go test ./internal/cmd

# Run tests with verbose output
go test -v ./...

# Run tests for specific functionality
go test ./internal/db -run ".*User.*" -v        # User repository tests
go test ./internal/service -run "TestUserService_" -v  # User service tests
go test ./internal/cmd -run "TestUserCommand_" -v      # User CLI tests

# Run specific individual tests
go test ./internal/db -run "TestCreateUser$" -v
go test ./internal/service -run "TestUserService_CreateUser$" -v
go test ./internal/cmd -run "TestUserCommand_CreateUser$" -v

# Short tests only (excludes slow integration tests)
go test ./... -short
```

### Testing Guidelines for New/Updated Code

When adding or modifying code, always create corresponding tests:

#### 1. Repository Layer Tests (`internal/db/*_test.go`)
- Test all CRUD operations (Create, Read, Update, Delete)
- Test error conditions (not found, duplicates, invalid input)
- Use random usernames/emails to avoid test data conflicts
- Example pattern:
```go
func TestCreateUser(t *testing.T) {
    testUsername := "testuser_create_" + randomString(8)
    defer cleanupUser(t, conn, testUsername)
    // ... test implementation
}
```

#### 2. Service Layer Tests (`internal/service/*_test.go`) 
- Test business logic and validation rules
- Test password hashing and verification
- Test error handling and edge cases
- Example validation tests for all input combinations:
```go
tests := []struct {
    name        string
    req         model.CreateUserRequest
    expectedErr error
}{
    {"empty username", CreateUserRequest{Username: ""}, ErrInvalidInput},
    // ... more test cases
}
```

#### 3. CLI Command Tests (`internal/cmd/*_test.go`)
- Test command parsing and execution
- Test output formatting and user interaction
- Use output capture for testing printed messages:
```go
output := captureOutput(t, func() {
    err := userCmd.Execute(ctx, args)
    // ... assertions
})
```

#### 4. Test Data Management
- Always use unique test data (random strings) to prevent conflicts
- Clean up test data in defer statements
- Use `test.env` for database configuration consistency
- Example cleanup pattern:
```go
defer cleanupUser(t, conn, testUsername)
```

#### 5. Running Tests During Development
```bash
# Test current changes
go test ./internal/db -run "TestUser.*" -v

# Quick feedback loop
go test ./internal/service -run "TestUserService_CreateUser$" -v

# Full regression test
go test ./... -v
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
- **model**: Defines data structures (`FileInfo`, `User`, `CreateUserRequest`, etc.) representing application entities
- **db**: PostgreSQL connection management and repository pattern for data operations (files, users, connections)
- **service**: Business logic layer with validation, password hashing, and error handling
- **cmd**: CLI command handlers for user management and application operations
- **api**: REST API handlers for web interface
- **tztime**: Timezone utilities for consistent time handling

### Database Schema

Three main tables:
- `users`: User accounts with authentication (username, email, password_hash)
- `connections`: Network share configurations (SMB/CIFS mount points) linked to users
- `files`: PDF file metadata linked to connections

### Key Patterns

- Uses `pgx/v5` for PostgreSQL connectivity
- Repository pattern in `internal/db` for data access
- Service layer for business logic and validation
- bcrypt for secure password hashing with automatic salt management
- CLI commands for administrative operations (user management)
- Functional approach with callback handlers (`ScanWith`)
- Environment-based configuration via `DATABASE_URL`
- Docker Compose for local PostgreSQL development
- Standard Go project layout with `cmd/` and `internal/` structure
- Comprehensive test coverage across all layers (repository, service, CLI)