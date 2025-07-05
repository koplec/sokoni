package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/koplec/sokoni/internal/model"
)

func randomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

func setupUserTestConn(t *testing.T) *pgx.Conn {
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

func cleanupUser(t *testing.T, conn *pgx.Conn, username string) {
	t.Helper()
	_, err := conn.Exec(context.Background(), "DELETE FROM users WHERE username = $1", username)
	if err != nil {
		t.Logf("failed to cleanup user %s: %v", username, err)
	}
}

func TestCreateUser(t *testing.T) {
	conn := setupUserTestConn(t)
	defer conn.Close(context.Background())

	ctx := context.Background()
	testUsername := "testuser_create_" + randomString(8)
	defer cleanupUser(t, conn, testUsername)

	req := model.CreateUserRequest{
		Username: testUsername,
		Email:    testUsername + "@example.com",
		Password: "password123",
	}
	passwordHash := "hashedpassword123"

	user, err := CreateUser(ctx, conn, req, passwordHash)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if user.Username != req.Username {
		t.Errorf("expected username %s, got %s", req.Username, user.Username)
	}
	if user.Email != req.Email {
		t.Errorf("expected email %s, got %s", req.Email, user.Email)
	}
	if user.PasswordHash != passwordHash {
		t.Errorf("expected password hash %s, got %s", passwordHash, user.PasswordHash)
	}
	if user.ID == 0 {
		t.Error("expected user ID to be set")
	}
}

func TestCreateUserAlreadyExists(t *testing.T) {
	conn := setupUserTestConn(t)
	defer conn.Close(context.Background())

	ctx := context.Background()
	testUsername := "testuser_duplicate"
	defer cleanupUser(t, conn, testUsername)

	req := model.CreateUserRequest{
		Username: testUsername,
		Email:    "test1@example.com",
		Password: "password123",
	}

	// First user creation
	_, err := CreateUser(ctx, conn, req, "hashedpassword123")
	if err != nil {
		t.Fatalf("failed to create first user: %v", err)
	}

	// Second user creation with same username should fail
	req.Email = "test2@example.com"
	_, err = CreateUser(ctx, conn, req, "hashedpassword456")
	if err == nil {
		t.Error("expected error when creating user with duplicate username")
	}
	if err != ErrUserAlreadyExists {
		t.Errorf("expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestCreateUserDuplicateEmail(t *testing.T) {
	conn := setupUserTestConn(t)
	defer conn.Close(context.Background())

	ctx := context.Background()
	testEmail := "duplicate@example.com"
	defer func() {
		conn.Exec(context.Background(), "DELETE FROM users WHERE email = $1", testEmail)
	}()

	req1 := model.CreateUserRequest{
		Username: "testuser1",
		Email:    testEmail,
		Password: "password123",
	}

	// First user creation
	_, err := CreateUser(ctx, conn, req1, "hashedpassword123")
	if err != nil {
		t.Fatalf("failed to create first user: %v", err)
	}

	// Second user creation with same email should fail
	req2 := model.CreateUserRequest{
		Username: "testuser2",
		Email:    testEmail,
		Password: "password456",
	}
	_, err = CreateUser(ctx, conn, req2, "hashedpassword456")
	if err == nil {
		t.Error("expected error when creating user with duplicate email")
	}
	if err != ErrUserAlreadyExists {
		t.Errorf("expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestGetUserByID(t *testing.T) {
	conn := setupUserTestConn(t)
	defer conn.Close(context.Background())

	ctx := context.Background()
	testUsername := "testuser_getbyid_" + randomString(8)
	defer cleanupUser(t, conn, testUsername)

	req := model.CreateUserRequest{
		Username: testUsername,
		Email:    testUsername + "@example.com",
		Password: "password123",
	}

	createdUser, err := CreateUser(ctx, conn, req, "hashedpassword123")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	user, err := GetUserByID(ctx, conn, createdUser.ID)
	if err != nil {
		t.Fatalf("failed to get user by ID: %v", err)
	}

	if user.ID != createdUser.ID {
		t.Errorf("expected ID %d, got %d", createdUser.ID, user.ID)
	}
	if user.Username != createdUser.Username {
		t.Errorf("expected username %s, got %s", createdUser.Username, user.Username)
	}
}

func TestGetUserByIDNotFound(t *testing.T) {
	conn := setupUserTestConn(t)
	defer conn.Close(context.Background())

	ctx := context.Background()

	_, err := GetUserByID(ctx, conn, 999999)
	if err == nil {
		t.Error("expected error when getting non-existent user")
	}
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGetUserByUsername(t *testing.T) {
	conn := setupUserTestConn(t)
	defer conn.Close(context.Background())

	ctx := context.Background()
	testUsername := "testuser_getbyusername_" + randomString(8)
	defer cleanupUser(t, conn, testUsername)

	req := model.CreateUserRequest{
		Username: testUsername,
		Email:    testUsername + "@example.com",
		Password: "password123",
	}

	createdUser, err := CreateUser(ctx, conn, req, "hashedpassword123")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	user, err := GetUserByUsername(ctx, conn, testUsername)
	if err != nil {
		t.Fatalf("failed to get user by username: %v", err)
	}

	if user.ID != createdUser.ID {
		t.Errorf("expected ID %d, got %d", createdUser.ID, user.ID)
	}
	if user.Username != testUsername {
		t.Errorf("expected username %s, got %s", testUsername, user.Username)
	}
}

func TestGetAllUsers(t *testing.T) {
	conn := setupUserTestConn(t)
	defer conn.Close(context.Background())

	ctx := context.Background()
	testUsername1 := "testuser_getall1"
	testUsername2 := "testuser_getall2"
	defer cleanupUser(t, conn, testUsername1)
	defer cleanupUser(t, conn, testUsername2)

	req1 := model.CreateUserRequest{
		Username: testUsername1,
		Email:    "test1@example.com",
		Password: "password123",
	}
	req2 := model.CreateUserRequest{
		Username: testUsername2,
		Email:    "test2@example.com",
		Password: "password456",
	}

	_, err := CreateUser(ctx, conn, req1, "hashedpassword123")
	if err != nil {
		t.Fatalf("failed to create user 1: %v", err)
	}
	_, err = CreateUser(ctx, conn, req2, "hashedpassword456")
	if err != nil {
		t.Fatalf("failed to create user 2: %v", err)
	}

	users, err := GetAllUsers(ctx, conn)
	if err != nil {
		t.Fatalf("failed to get all users: %v", err)
	}

	if len(users) < 2 {
		t.Errorf("expected at least 2 users, got %d", len(users))
	}

	// Check that our created users are in the list
	foundUser1 := false
	foundUser2 := false
	for _, user := range users {
		if user.Username == testUsername1 {
			foundUser1 = true
		}
		if user.Username == testUsername2 {
			foundUser2 = true
		}
	}

	if !foundUser1 {
		t.Error("user 1 not found in all users list")
	}
	if !foundUser2 {
		t.Error("user 2 not found in all users list")
	}
}

func TestUpdateUser(t *testing.T) {
	conn := setupUserTestConn(t)
	defer conn.Close(context.Background())

	ctx := context.Background()
	testUsername := "testuser_update"
	defer cleanupUser(t, conn, testUsername)

	req := model.CreateUserRequest{
		Username: testUsername,
		Email:    "old@example.com",
		Password: "password123",
	}

	createdUser, err := CreateUser(ctx, conn, req, "hashedpassword123")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	updateReq := model.UpdateUserRequest{
		Username: "newusername",
		Email:    "new@example.com",
	}
	newPasswordHash := "newhashedpassword"

	updatedUser, err := UpdateUser(ctx, conn, createdUser.ID, updateReq, &newPasswordHash)
	if err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	if updatedUser.Username != updateReq.Username {
		t.Errorf("expected username %s, got %s", updateReq.Username, updatedUser.Username)
	}
	if updatedUser.Email != updateReq.Email {
		t.Errorf("expected email %s, got %s", updateReq.Email, updatedUser.Email)
	}
	if updatedUser.PasswordHash != newPasswordHash {
		t.Errorf("expected password hash %s, got %s", newPasswordHash, updatedUser.PasswordHash)
	}

	// Cleanup with new username
	cleanupUser(t, conn, "newusername")
}

func TestDeleteUser(t *testing.T) {
	conn := setupUserTestConn(t)
	defer conn.Close(context.Background())

	ctx := context.Background()
	testUsername := "testuser_delete_" + randomString(8)

	req := model.CreateUserRequest{
		Username: testUsername,
		Email:    testUsername + "@example.com",
		Password: "password123",
	}

	createdUser, err := CreateUser(ctx, conn, req, "hashedpassword123")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	err = DeleteUser(ctx, conn, createdUser.ID)
	if err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	// Verify user is deleted
	_, err = GetUserByID(ctx, conn, createdUser.ID)
	if err == nil {
		t.Error("expected error when getting deleted user")
	}
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestDeleteUserNotFound(t *testing.T) {
	conn := setupUserTestConn(t)
	defer conn.Close(context.Background())

	ctx := context.Background()

	err := DeleteUser(ctx, conn, 999999)
	if err == nil {
		t.Error("expected error when deleting non-existent user")
	}
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}