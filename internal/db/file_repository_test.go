package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/koplec/sokoni/internal/model"
	"github.com/koplec/sokoni/internal/tztime"
)

func setupTestConn(t *testing.T) *pgx.Conn {
	t.Helper()

	err := godotenv.Load(filepath.Join("..", "..", "test.env"))
	if err != nil {
		t.Fatalf("no test.env file found, using default env")
	}

	ctx := context.Background()

	url := os.Getenv("DATABASE_URL")
	conn, err := pgx.Connect(ctx, url)

	if err != nil {
		t.Fatalf("failed to connect:%v", err)
	}

	return conn
}

func TestInsertFile(t *testing.T) {
	conn := setupTestConn(t)
	defer conn.Close(context.Background())

	ctx := context.Background()

	file := model.FileInfo{
		Path:    "/test/sample.pdf",
		Name:    "sample.pdf",
		Size:    23456,
		ModTime: tztime.Now(),
	}
	file.ModTime = file.ModTime.Truncate(time.Second)
	connectionID := 999

	// connectionID := 1をconnectionsテーブルに入れる
	_, err := conn.Exec(ctx, `
    INSERT INTO connections (id, name, base_path, remote_path)
    VALUES ($1, 'test-connection-name', 'test-base-path', 'test-remote-path')
    ON CONFLICT (id) DO NOTHING
`, connectionID)

	if err != nil {
		t.Fatalf("failed to insert dummy connection: %v", err)
	}

	// Insert the file into the database
	err = InsertFile(ctx, conn, connectionID, file)
	if err != nil {
		t.Fatalf("failed to insert file: %v", err)
	}

	// Select確認
	var gotSize int64
	var gotModTime time.Time
	var gotName string

	err = conn.QueryRow(ctx, `
	SELECT size, mod_time, name
	FROM files
	WHERE connection_id = $1 AND path = $2
	`, connectionID, file.Path).Scan(&gotSize, &gotModTime, &gotName)
	if err != nil {
		t.Fatalf("failed to fetch inserted file for path=%s, connectionID=%d: %v", file.Path, connectionID, err)
	}
	if gotSize != file.Size || gotName != file.Name || !tztime.EqualTime(gotModTime, file.ModTime) {
		t.Errorf("inserted data mismatch:\n got (%d, %s, %v), want (%d, %s, %v)", gotSize, gotName, gotModTime, file.Size, file.Name, file.ModTime)
	}

	// DELETE確認
	_, err = conn.Exec(ctx, `
	DELETE FROM files WHERE connection_id = $1 AND path = $2
	`, connectionID, file.Path)
	if err != nil {
		t.Fatalf("failed to delete inserted file: %v", err)
	}

	//削除確認
	var count int
	err = conn.QueryRow(ctx, `
	SELECT COUNT(*) FROM files WHERE connection_id = $1 AND path = $2
	`, connectionID, file.Path).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count after delete: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 files, got %d", count)
	}

}
