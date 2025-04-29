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
		ModTime: time.Now(),
	}
	connectionID := 1

	err := InsertFile(ctx, conn, connectionID, file)
	if err != nil {
		t.Fatalf("failed to insert file: %v", err)
	}
}
