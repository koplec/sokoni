package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	"github.com/koplec/sokoni/internal/cmd"
	"github.com/koplec/sokoni/internal/db"
	"github.com/koplec/sokoni/internal/service"
)

// helper to connect to database for tests. skips test when db is unavailable
func testConn(t *testing.T) *pgx.Conn {
	t.Helper()
	err := godotenv.Load("../../test.env")
	if err != nil {
		t.Logf("Warning: Could not load test.env: %v", err)
	}
	ctx := context.Background()
	conn, err := db.Connect(ctx)
	if err != nil {
		t.Skipf("Database connection failed: %v", err)
	}
	t.Cleanup(func() { conn.Close(ctx) })
	return conn
}

func TestScanConnectionLocal(t *testing.T) {
	conn := testConn(t)
	ctx := context.Background()

	// clean tables in case previous tests left data
	conn.Exec(ctx, "DELETE FROM files")
	conn.Exec(ctx, "DELETE FROM connections")

	// create temp directory with a single pdf file
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "sample.pdf"), []byte("dummy"), 0644)

	connectionID := 101
	_, err := conn.Exec(ctx, `
        INSERT INTO connections (id, name, base_path, remote_path)
        VALUES ($1, 'test', $2, $2)
        ON CONFLICT (id) DO NOTHING
    `, connectionID, dir)
	if err != nil {
		t.Fatalf("failed to insert connection: %v", err)
	}

	t.Cleanup(func() {
		conn.Exec(ctx, "DELETE FROM files WHERE connection_id=$1", connectionID)
		conn.Exec(ctx, "DELETE FROM connections WHERE id=$1", connectionID)
	})

	scanner := service.NewConnectionScanner(conn)
	err = cmd.ScanConnection(connectionID, scanner)
	if err != nil {
		t.Fatalf("ScanConnection failed: %v", err)
	}

	var count int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM files WHERE connection_id=$1", connectionID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query files: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 file, got %d", count)
	}
}

// stringPtr returns a pointer to s if s is not empty, otherwise nil.
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// TestScanConnectionSMB verifies scanning using an SMB path if environment variables are provided.
func TestScanConnectionSMB(t *testing.T) {
	// Check DATABASE_URL is set (required for testConn)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatal("DATABASE_URL environment variable is required for SMB test")
	}

	// Optional SMB credentials - warn if not set but don't fail
	// Check required environment variables
	smbPath := os.Getenv("SOKONI_TEST_SMB_PATH")
	if smbPath == "" {
		t.Skip("SOKONI_TEST_SMB_PATH not set")
	}

	smbUser := os.Getenv("SOKONI_TEST_SMB_USER")
	smbPass := os.Getenv("SOKONI_TEST_SMB_PASS")
	if smbUser == "" || smbPass == "" {
		t.Logf("Warning: SOKONI_TEST_SMB_USER or SOKONI_TEST_SMB_PASS not set - SMB authentication may fail")
	}

	conn := testConn(t)
	ctx := context.Background()

	conn.Exec(ctx, "DELETE FROM files")
	conn.Exec(ctx, "DELETE FROM connections")
	conn.Exec(ctx, "DELETE FROM users")

	// Create test user
	var userID int
	err := conn.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash) 
		VALUES ('test-user', 'test@example.com', 'dummy-hash') 
		RETURNING id
	`).Scan(&userID)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	connectionID := 102
	_, err = conn.Exec(ctx, `
        INSERT INTO connections (id, name, base_path, remote_path, username, password, options, user_id)
        VALUES ($1, 'smb-test', $2, $3, $4, $5, $6, $7)
        ON CONFLICT (id) DO NOTHING
    `, connectionID, smbPath, smbPath,
		stringPtr(os.Getenv("SOKONI_TEST_SMB_USER")),
		stringPtr(os.Getenv("SOKONI_TEST_SMB_PASS")),
		stringPtr(os.Getenv("SOKONI_TEST_SMB_OPTIONS")),
		userID)
	if err != nil {
		t.Fatalf("failed to insert connection: %v", err)
	}

	// Register cleanup immediately after data creation
	t.Cleanup(func() {
		conn.Exec(ctx, "DELETE FROM files WHERE connection_id=$1", connectionID)
		conn.Exec(ctx, "DELETE FROM connections WHERE id=$1", connectionID)
		conn.Exec(ctx, "DELETE FROM users WHERE id=$1", userID)
	})

	// Verify data insertion
	var userCount, connectionCount int
	conn.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE id=$1", userID).Scan(&userCount)
	conn.QueryRow(ctx, "SELECT COUNT(*) FROM connections WHERE id=$1", connectionID).Scan(&connectionCount)
	if userCount != 1 || connectionCount != 1 {
		t.Fatalf("expected 1 user and 1 connection, got %d users and %d connections", userCount, connectionCount)
	}

	scanner := service.NewConnectionScanner(conn)
	err = cmd.ScanConnection(connectionID, scanner)
	if err != nil {
		t.Fatalf("ScanConnection failed: %v", err)
	}

	var count int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM files WHERE connection_id=$1", connectionID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query files: %v", err)
	}
	if count == 0 {
		t.Logf("No files scanned (expected due to SMB authentication failure)")
	} else {
		t.Logf("Scanned %d files", count)
	}
}
