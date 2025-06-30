package service_test

import (
	"context"
	"errors"
	"fmt"
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
	conn.Exec(ctx, "DELETE FROM users")

	// create temp directory with a single pdf file
	baseDir := t.TempDir()
	remoteSub := "data"
	remoteDir := filepath.Join(baseDir, remoteSub)
	os.MkdirAll(remoteDir, 0755)
	os.WriteFile(filepath.Join(remoteDir, "sample.pdf"), []byte("dummy"), 0644)

	// create test user and connection
	var userID int
	err := conn.QueryRow(ctx, `
                INSERT INTO users (username, email, password_hash)
                VALUES ('local-user', 'local@example.com', 'x')
                RETURNING id
        `).Scan(&userID)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	connectionID := 101
	_, err = conn.Exec(ctx, `
        INSERT INTO connections (id, name, base_path, remote_path, user_id)
        VALUES ($1, 'test', $2, $3, $4)
        ON CONFLICT (id) DO NOTHING
    `, connectionID, baseDir, remoteSub, userID)
	if err != nil {
		t.Fatalf("failed to insert connection: %v", err)
	}

	t.Cleanup(func() {
		conn.Exec(ctx, "DELETE FROM files WHERE connection_id=$1", connectionID)
		conn.Exec(ctx, "DELETE FROM connections WHERE id=$1", connectionID)
		conn.Exec(ctx, "DELETE FROM users WHERE id=$1", userID)
	})

	scanner := service.NewConnectionScanner(conn)
	err = cmd.ScanConnection(connectionID, userID, scanner)
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

	// Check required environment variables
	smbBase := os.Getenv("SOKONI_TEST_SMB_BASE_PATH")
	if smbBase == "" {
		t.Skip("SOKONI_TEST_SMB_BASE_PATH not set")
	}
	smbRemote := os.Getenv("SOKONI_TEST_SMB_REMOTE_PATH")

	// Optional SMB credentials - warn if not set but don't fail
	smbUser := os.Getenv("SOKONI_TEST_SMB_USER")
	smbPass := os.Getenv("SOKONI_TEST_SMB_PASS")
	if smbUser == "" || smbPass == "" {
		t.Logf("Warning: SOKONI_TEST_SMB_USER or SOKONI_TEST_SMB_PASS not set - SMB authentication may fail")
	}

	// Validate expected PDF count format if set
	expectedPdfCountStr := os.Getenv("SOKONI_TEST_SMB_EXPECTED_PDF_COUNT")
	if expectedPdfCountStr == "" {
		t.Fatalf("SOKONI_TEST_SMB_EXPECTED_PDF_COUNT not set")
	}
	expectedPdfCount := 0
	if _, err := fmt.Sscanf(expectedPdfCountStr, "%d", &expectedPdfCount); err != nil {
		t.Fatalf("invalid SOKONI_TEST_SMB_EXPECTED_PDF_COUNT format: %v", err)
	}
	if expectedPdfCount < 0 {
		t.Fatalf("SOKONI_TEST_SMB_EXPECTED_PDF_COUNT must be non-negative, got: %d", expectedPdfCount)
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
    `, connectionID, smbBase, smbRemote,
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
	err = cmd.ScanConnection(connectionID, userID, scanner)
	if err != nil {
		t.Fatalf("ScanConnection failed: %v", err)
	}

	var actualPdfCount int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM files WHERE connection_id=$1", connectionID).Scan(&actualPdfCount)
	if err != nil {
		t.Fatalf("failed to query files: %v", err)
	}

	// Check expected PDF count if environment variable is set
	if expectedPdfCountStr != "" {
		expectedPdfCount := 0
		if _, err := fmt.Sscanf(expectedPdfCountStr, "%d", &expectedPdfCount); err != nil {
			t.Fatalf("invalid SOKONI_TEST_SMB_EXPECTED_PDF_COUNT format: %v", err)
		}
		if actualPdfCount != expectedPdfCount {
			t.Errorf("expected %d PDF files, but scanned %d files", expectedPdfCount, actualPdfCount)
		} else {
			t.Logf("Successfully scanned %d PDF files as expected", actualPdfCount)
		}
	} else {
		// Fallback to original behavior when expected count is not set
		if actualPdfCount == 0 {
			t.Logf("No PDF files scanned (expected due to SMB authentication failure)")
		} else {
			t.Logf("Scanned %d PDF files", actualPdfCount)
		}
	}
}

func TestScanConnectionUnauthorized(t *testing.T) {
	conn := testConn(t)
	ctx := context.Background()

	conn.Exec(ctx, "DELETE FROM files")
	conn.Exec(ctx, "DELETE FROM connections")
	conn.Exec(ctx, "DELETE FROM users")

	// create two users
	var user1, user2 int
	err := conn.QueryRow(ctx, `
                INSERT INTO users (username, email, password_hash)
                VALUES ('owner', 'owner@example.com', 'x')
                RETURNING id
        `).Scan(&user1)
	if err != nil {
		t.Fatalf("failed to create user1: %v", err)
	}
	err = conn.QueryRow(ctx, `
                INSERT INTO users (username, email, password_hash)
                VALUES ('other', 'other@example.com', 'x')
                RETURNING id
        `).Scan(&user2)
	if err != nil {
		t.Fatalf("failed to create user2: %v", err)
	}

	// temp directory and connection owned by user1
	baseDir := t.TempDir()
	remoteSub := "private"
	remoteDir := filepath.Join(baseDir, remoteSub)
	os.MkdirAll(remoteDir, 0755)
	os.WriteFile(filepath.Join(remoteDir, "sample.pdf"), []byte("dummy"), 0644)

	connectionID := 103
	_, err = conn.Exec(ctx, `
                INSERT INTO connections (id, name, base_path, remote_path, user_id)
                VALUES ($1, 'unauth', $2, $3, $4)
        `, connectionID, baseDir, remoteSub, user1)
	if err != nil {
		t.Fatalf("failed to insert connection: %v", err)
	}

	t.Cleanup(func() {
		conn.Exec(ctx, "DELETE FROM files WHERE connection_id=$1", connectionID)
		conn.Exec(ctx, "DELETE FROM connections WHERE id=$1", connectionID)
		conn.Exec(ctx, "DELETE FROM users WHERE id IN ($1,$2)", user1, user2)
	})

	scanner := service.NewConnectionScanner(conn)
	err = cmd.ScanConnection(connectionID, user2, scanner)
	if err == nil {
		t.Fatalf("expected error for unauthorized user")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("unexpected error: %v", err)
	}

	var count int
	conn.QueryRow(ctx, "SELECT COUNT(*) FROM files WHERE connection_id=$1", connectionID).Scan(&count)
	if count != 0 {
		t.Errorf("expected no files scanned, got %d", count)
	}
}
