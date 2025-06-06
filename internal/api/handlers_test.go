package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/koplec/sokoni/internal/db"
	"github.com/koplec/sokoni/internal/model"
)

func TestSearchFiles(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Connect(ctx)
	if err != nil {
		t.Skipf("Database connection failed: %v", err)
	}
	defer conn.Close(ctx)

	api := NewAPI(conn)

	insertTestData(t, ctx, conn)

	req := httptest.NewRequest("GET", "/search?q=test", nil)
	w := httptest.NewRecorder()

	api.SearchFiles(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var files []model.FileInfo
	err = json.Unmarshal(w.Body.Bytes(), &files)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(files) == 0 {
		t.Error("Expected to find test files")
	}
}

func TestSearchFilesNoQuery(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Connect(ctx)
	if err != nil {
		t.Skipf("Database connection failed: %v", err)
	}
	defer conn.Close(ctx)

	api := NewAPI(conn)

	req := httptest.NewRequest("GET", "/search", nil)
	w := httptest.NewRecorder()

	api.SearchFiles(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func insertTestData(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	conn.Exec(ctx, "DELETE FROM files")
	conn.Exec(ctx, "DELETE FROM connections")

	_, err := conn.Exec(ctx, `
		INSERT INTO connections (id, name, base_path, remote_path) 
		VALUES (1, 'test', '/test', '//test/share')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test connection: %v", err)
	}

	testFile := model.FileInfo{
		Path:    "/test/sample.pdf",
		Name:    "test-sample.pdf",
		Size:    1024,
		ModTime: time.Now(),
	}

	err = db.InsertFile(ctx, conn, 1, testFile)
	if err != nil {
		t.Fatalf("Failed to insert test file: %v", err)
	}
}