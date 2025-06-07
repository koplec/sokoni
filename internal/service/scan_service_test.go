package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	"github.com/koplec/sokoni/internal/cmd"
	"github.com/koplec/sokoni/internal/db"
)

// helper to connect to database for tests. skips test when db is unavailable
func testConn(t *testing.T) *pgx.Conn {
	t.Helper()
	_ = godotenv.Load(filepath.Join("..", "..", "test.env"))
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

	scanner := NewConnectionScanner(conn)
	cmd.ScanConnection(connectionID, scanner)

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
	smbPath := os.Getenv("SOKONI_TEST_SMB_PATH")
	if smbPath == "" {
		t.Skip("SOKONI_TEST_SMB_PATH not set")
	}

	conn := testConn(t)
	ctx := context.Background()

	conn.Exec(ctx, "DELETE FROM files")
	conn.Exec(ctx, "DELETE FROM connections")

	connectionID := 102
	_, err := conn.Exec(ctx, `
        INSERT INTO connections (id, name, base_path, remote_path, username, password, options)
        VALUES ($1, 'smb-test', $2, $3, $4, $5, $6)
        ON CONFLICT (id) DO NOTHING
    `, connectionID, smbPath, smbPath,
		stringPtr(os.Getenv("SOKONI_TEST_SMB_USER")),
		stringPtr(os.Getenv("SOKONI_TEST_SMB_PASS")),
		stringPtr(os.Getenv("SOKONI_TEST_SMB_OPTIONS")))
	if err != nil {
		t.Fatalf("failed to insert connection: %v", err)
	}

	t.Cleanup(func() {
		conn.Exec(ctx, "DELETE FROM files WHERE connection_id=$1", connectionID)
		conn.Exec(ctx, "DELETE FROM connections WHERE id=$1", connectionID)
	})

	scanner := NewConnectionScanner(conn)
	cmd.ScanConnection(connectionID, scanner)

	var count int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM files WHERE connection_id=$1", connectionID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query files: %v", err)
	}
	if count == 0 {
		t.Error("expected at least 1 file scanned")
	}
}
