package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/koplec/sokoni/internal/db"
	"github.com/koplec/sokoni/internal/model"
	"github.com/koplec/sokoni/internal/service"
)

func createScanTestUser(t *testing.T, conn *pgx.Conn) *model.User {
	ctx := context.Background()
	
	// Create user using repository
	userReq := model.CreateUserRequest{
		Username: fmt.Sprintf("testuser_scan_%d", time.Now().UnixNano()),
		Email:    fmt.Sprintf("testuser_scan_%d@example.com", time.Now().UnixNano()),
	}
	
	user, err := db.CreateUser(ctx, conn, userReq, "dummy_hash")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	
	return user
}

func createScanTestConnection(t *testing.T, conn *pgx.Conn, userID int, testDir string) *db.Connection {
	ctx := context.Background()
	
	// Create connection using repository
	connReq := db.CreateConnectionRequest{
		UserID:     userID,
		Name:       fmt.Sprintf("Test Connection %d", time.Now().UnixNano()),
		BasePath:   testDir,
		RemotePath: "", // Leave empty for local path
	}
	
	connResp, err := db.CreateConnection(ctx, conn, connReq)
	if err != nil {
		t.Fatalf("Failed to create test connection: %v", err)
	}
	
	// Get the full connection record
	connection, err := db.GetConnectionByID(ctx, conn, connResp.ID)
	if err != nil {
		t.Fatalf("Failed to get created connection: %v", err)
	}
	
	return connection
}

func TestScanConnection(t *testing.T) {
	// Load test environment
	err := godotenv.Load("../../test.env")
	if err != nil {
		t.Logf("Warning: Could not load test.env file: %v", err)
	}

	ctx := context.Background()
	conn, err := db.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	// Create test directory structure
	testDir := "/tmp/test_scan"
	defer os.RemoveAll(testDir)

	err = os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create test PDF files
	testFiles := []string{
		"document1.pdf",
		"document2.pdf",
		"report.pdf",
	}

	for _, filename := range testFiles {
		filePath := filepath.Join(testDir, filename)
		err = os.WriteFile(filePath, []byte("fake pdf content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create test user and connection with real database IDs
	testUser := createScanTestUser(t, conn)
	testConnection := createScanTestConnection(t, conn, testUser.ID, testDir)

	// Cleanup after test
	defer func() {
		_, _ = conn.Exec(ctx, "DELETE FROM files WHERE connection_id = $1", testConnection.ID)
		_, _ = conn.Exec(ctx, "DELETE FROM connections WHERE id = $1", testConnection.ID)
		_, _ = conn.Exec(ctx, "DELETE FROM users WHERE id = $1", testUser.ID)
	}()

	// Test successful scan
	err = ScanConnection(testConnection.ID, testUser.ID, service.NewConnectionScanner(conn))
	if err != nil {
		t.Fatalf("ScanConnection failed: %v", err)
	}

	// Verify files were inserted
	rows, err := conn.Query(ctx, "SELECT COUNT(*) FROM files WHERE connection_id = $1", testConnection.ID)
	if err != nil {
		t.Fatalf("Failed to query files: %v", err)
	}
	defer rows.Close()

	var count int
	if rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			t.Fatalf("Failed to scan count: %v", err)
		}
	}

	if count != len(testFiles) {
		t.Errorf("Expected %d files, got %d", len(testFiles), count)
	}
}

func TestScanConnection_InvalidConnectionID(t *testing.T) {
	// Load test environment
	err := godotenv.Load("../../test.env")
	if err != nil {
		t.Logf("Warning: Could not load test.env file: %v", err)
	}

	ctx := context.Background()
	conn, err := db.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	// Create test user for valid user ID
	testUser := createScanTestUser(t, conn)
	defer func() {
		_, _ = conn.Exec(ctx, "DELETE FROM users WHERE id = $1", testUser.ID)
	}()

	// Test with invalid connection ID
	invalidConnectionID := 99999
	err = ScanConnection(invalidConnectionID, testUser.ID, service.NewConnectionScanner(conn))
	if err == nil {
		t.Error("Expected error for invalid connection ID, got nil")
	}

	expectedError := fmt.Sprintf("failed to scan connection %d", invalidConnectionID)
	if err.Error()[:len(expectedError)] != expectedError {
		t.Errorf("Expected error to start with %q, got %q", expectedError, err.Error())
	}
}

func TestScanConnection_InvalidUserID(t *testing.T) {
	// Load test environment
	err := godotenv.Load("../../test.env")
	if err != nil {
		t.Logf("Warning: Could not load test.env file: %v", err)
	}

	ctx := context.Background()
	conn, err := db.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	// Create test user and connection for valid connection ID
	testUser := createScanTestUser(t, conn)
	testDir := "/tmp/test_scan_invalid_user"
	defer os.RemoveAll(testDir)
	err = os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	
	testConnection := createScanTestConnection(t, conn, testUser.ID, testDir)
	
	defer func() {
		_, _ = conn.Exec(ctx, "DELETE FROM connections WHERE id = $1", testConnection.ID)
		_, _ = conn.Exec(ctx, "DELETE FROM users WHERE id = $1", testUser.ID)
	}()

	// Test with invalid user ID
	invalidUserID := 88888
	err = ScanConnection(testConnection.ID, invalidUserID, service.NewConnectionScanner(conn))
	if err == nil {
		t.Error("Expected error for invalid user ID, got nil")
	}

	expectedError := fmt.Sprintf("failed to scan connection %d", testConnection.ID)
	if err.Error()[:len(expectedError)] != expectedError {
		t.Errorf("Expected error to start with %q, got %q", expectedError, err.Error())
	}
}