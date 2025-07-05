package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/koplec/sokoni/internal/db"
	"github.com/koplec/sokoni/internal/model"
	"github.com/koplec/sokoni/internal/service"
)

func randomStringAPI(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

func setupAPITestConn(t *testing.T) *pgx.Conn {
	t.Helper()

	err := godotenv.Load(filepath.Join("..", "..", "test.env"))
	if err != nil {
		t.Logf("no test.env file found, using default env")
	}

	ctx := context.Background()

	url := os.Getenv("DATABASE_URL")
	conn, err := pgx.Connect(ctx, url)

	if err != nil {
		t.Fatalf("failed to connect:%v", err)
	}

	return conn
}

func cleanupAPIUser(t *testing.T, conn *pgx.Conn, username string) {
	t.Helper()
	_, err := conn.Exec(context.Background(), "DELETE FROM users WHERE username = $1", username)
	if err != nil {
		t.Logf("failed to cleanup user %s: %v", username, err)
	}
}

func cleanupAPIConnection(t *testing.T, conn *pgx.Conn, connectionID int) {
	t.Helper()
	_, err := conn.Exec(context.Background(), "DELETE FROM connections WHERE id = $1", connectionID)
	if err != nil {
		t.Logf("failed to cleanup connection %d: %v", connectionID, err)
	}
}

func createTestUser(t *testing.T, conn *pgx.Conn) *model.User {
	t.Helper()

	userService := service.NewUserService(conn)
	testUsername := "testapi_" + randomStringAPI(8)

	req := model.CreateUserRequest{
		Username: testUsername,
		Email:    testUsername + "@example.com",
		Password: "password123",
	}

	user, err := userService.CreateUser(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	t.Cleanup(func() {
		cleanupAPIUser(t, conn, testUsername)
	})

	return user
}

func TestAPI_CreateConnection_ValidUserID(t *testing.T) {
	conn := setupAPITestConn(t)
	defer conn.Close(context.Background())

	api := NewAPI(conn)
	user := createTestUser(t, conn)

	reqBody := db.CreateConnectionRequest{
		Name:         "Test Connection",
		BasePath:     "/tmp/test",
		RemotePath:   "//test/share",
		UserID:       user.ID,
		ScanInterval: &[]int{86400}[0],
		AutoScan:     &[]bool{true}[0],
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest("POST", "/connections", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.CreateConnection(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
		t.Logf("response body: %s", w.Body.String())
	}

	var resp db.ConnectionResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Name != reqBody.Name {
		t.Errorf("expected name %s, got %s", reqBody.Name, resp.Name)
	}
	if resp.UserID != user.ID {
		t.Errorf("expected user_id %d, got %d", user.ID, resp.UserID)
	}
	if resp.ID == 0 {
		t.Error("expected connection ID to be set")
	}

	// Cleanup
	t.Cleanup(func() {
		cleanupAPIConnection(t, conn, resp.ID)
	})
}

func TestAPI_CreateConnection_InvalidUserID(t *testing.T) {
	conn := setupAPITestConn(t)
	defer conn.Close(context.Background())

	api := NewAPI(conn)

	reqBody := db.CreateConnectionRequest{
		Name:       "Test Connection",
		BasePath:   "/tmp/test",
		RemotePath: "//test/share",
		UserID:     999999, // Non-existent user
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest("POST", "/connections", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.CreateConnection(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("User not found")) {
		t.Errorf("expected 'User not found' in response, got: %s", w.Body.String())
	}
}

func TestAPI_CreateConnection_MissingUserID(t *testing.T) {
	conn := setupAPITestConn(t)
	defer conn.Close(context.Background())

	api := NewAPI(conn)

	reqBody := db.CreateConnectionRequest{
		Name:       "Test Connection",
		BasePath:   "/tmp/test",
		RemotePath: "//test/share",
		UserID:     0, // Missing user_id
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest("POST", "/connections", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.CreateConnection(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("user_id is required")) {
		t.Errorf("expected 'user_id is required' in response, got: %s", w.Body.String())
	}
}

func TestAPI_CreateConnection_InvalidJSON(t *testing.T) {
	conn := setupAPITestConn(t)
	defer conn.Close(context.Background())

	api := NewAPI(conn)

	req := httptest.NewRequest("POST", "/connections", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.CreateConnection(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("Invalid JSON")) {
		t.Errorf("expected 'Invalid JSON' in response, got: %s", w.Body.String())
	}
}

func TestAPI_GetConnections_ValidUserID(t *testing.T) {
	conn := setupAPITestConn(t)
	defer conn.Close(context.Background())

	api := NewAPI(conn)
	user := createTestUser(t, conn)

	// Create a test connection first
	reqBody := db.CreateConnectionRequest{
		Name:         "Test Connection for List",
		BasePath:     "/tmp/test-list",
		RemotePath:   "//test/share-list",
		UserID:       user.ID,
		ScanInterval: &[]int{86400}[0],
		AutoScan:     &[]bool{true}[0],
	}

	connection, err := db.CreateConnection(context.Background(), conn, reqBody)
	if err != nil {
		t.Fatalf("failed to create test connection: %v", err)
	}

	t.Cleanup(func() {
		cleanupAPIConnection(t, conn, connection.ID)
	})

	// Test GetConnections
	req := httptest.NewRequest("GET", "/connections?user_id="+strconv.Itoa(user.ID), nil)
	w := httptest.NewRecorder()

	api.GetConnections(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		t.Logf("response body: %s", w.Body.String())
	}

	var resp []*db.ConnectionResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp) == 0 {
		t.Error("expected at least one connection in response")
	}

	// Check if our created connection is in the list
	found := false
	for _, c := range resp {
		if c.ID == connection.ID {
			found = true
			if c.UserID != user.ID {
				t.Errorf("expected user_id %d, got %d", user.ID, c.UserID)
			}
			break
		}
	}

	if !found {
		t.Error("created connection not found in response")
	}
}

func TestAPI_GetConnections_InvalidUserID(t *testing.T) {
	conn := setupAPITestConn(t)
	defer conn.Close(context.Background())

	api := NewAPI(conn)

	req := httptest.NewRequest("GET", "/connections?user_id=999999", nil)
	w := httptest.NewRecorder()

	api.GetConnections(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("User not found")) {
		t.Errorf("expected 'User not found' in response, got: %s", w.Body.String())
	}
}

func TestAPI_GetConnections_MissingUserID(t *testing.T) {
	conn := setupAPITestConn(t)
	defer conn.Close(context.Background())

	api := NewAPI(conn)

	req := httptest.NewRequest("GET", "/connections", nil)
	w := httptest.NewRecorder()

	api.GetConnections(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("user_id query parameter is required")) {
		t.Errorf("expected 'user_id query parameter is required' in response, got: %s", w.Body.String())
	}
}