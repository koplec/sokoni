package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/koplec/sokoni/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func randomStringService(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

func setupUserServiceTestConn(t *testing.T) *pgx.Conn {
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

func cleanupUserService(t *testing.T, conn *pgx.Conn, username string) {
	t.Helper()
	_, err := conn.Exec(context.Background(), "DELETE FROM users WHERE username = $1", username)
	if err != nil {
		t.Logf("failed to cleanup user %s: %v", username, err)
	}
}

func TestUserService_CreateUser(t *testing.T) {
	conn := setupUserServiceTestConn(t)
	defer conn.Close(context.Background())

	service := NewUserService(conn)
	ctx := context.Background()
	testUsername := "testuser_service_create_" + randomStringService(6)
	defer cleanupUserService(t, conn, testUsername)

	req := model.CreateUserRequest{
		Username: testUsername,
		Email:    testUsername + "@example.com",
		Password: "password123",
	}

	user, err := service.CreateUser(ctx, req)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if user.Username != req.Username {
		t.Errorf("expected username %s, got %s", req.Username, user.Username)
	}
	if user.Email != req.Email {
		t.Errorf("expected email %s, got %s", req.Email, user.Email)
	}
	if user.ID == 0 {
		t.Error("expected user ID to be set")
	}

	// Verify password was hashed
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		t.Error("password was not hashed correctly")
	}
}

func TestUserService_CreateUserValidation(t *testing.T) {
	conn := setupUserServiceTestConn(t)
	defer conn.Close(context.Background())

	service := NewUserService(conn)
	ctx := context.Background()

	tests := []struct {
		name        string
		req         model.CreateUserRequest
		expectedErr error
	}{
		{
			name: "empty username",
			req: model.CreateUserRequest{
				Username: "",
				Email:    "test@example.com",
				Password: "password123",
			},
			expectedErr: ErrInvalidInput,
		},
		{
			name: "empty email",
			req: model.CreateUserRequest{
				Username: "testuser",
				Email:    "",
				Password: "password123",
			},
			expectedErr: ErrInvalidInput,
		},
		{
			name: "empty password",
			req: model.CreateUserRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "",
			},
			expectedErr: ErrInvalidInput,
		},
		{
			name: "invalid email format",
			req: model.CreateUserRequest{
				Username: "testuser",
				Email:    "invalid-email",
				Password: "password123",
			},
			expectedErr: ErrInvalidEmail,
		},
		{
			name: "weak password",
			req: model.CreateUserRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "weak",
			},
			expectedErr: ErrWeakPassword,
		},
		{
			name: "password without number",
			req: model.CreateUserRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "passwordonly",
			},
			expectedErr: ErrWeakPassword,
		},
		{
			name: "password without letter",
			req: model.CreateUserRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "12345678",
			},
			expectedErr: ErrWeakPassword,
		},
		{
			name: "invalid username format",
			req: model.CreateUserRequest{
				Username: "te",
				Email:    "test@example.com",
				Password: "password123",
			},
			expectedErr: ErrInvalidUsername,
		},
		{
			name: "username with invalid characters",
			req: model.CreateUserRequest{
				Username: "test@user",
				Email:    "test@example.com",
				Password: "password123",
			},
			expectedErr: ErrInvalidUsername,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateUser(ctx, tt.req)
			if err == nil {
				t.Error("expected error but got none")
			}
			if !isErrorType(err, tt.expectedErr) {
				t.Errorf("expected error type %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

func TestUserService_GetUserByID(t *testing.T) {
	conn := setupUserServiceTestConn(t)
	defer conn.Close(context.Background())

	service := NewUserService(conn)
	ctx := context.Background()
	testUsername := "testuser_service_getbyid_" + randomStringService(8)
	defer cleanupUserService(t, conn, testUsername)

	req := model.CreateUserRequest{
		Username: testUsername,
		Email:    testUsername + "@example.com",
		Password: "password123",
	}

	createdUser, err := service.CreateUser(ctx, req)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	user, err := service.GetUserByID(ctx, createdUser.ID)
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

func TestUserService_GetUserByIDNotFound(t *testing.T) {
	conn := setupUserServiceTestConn(t)
	defer conn.Close(context.Background())

	service := NewUserService(conn)
	ctx := context.Background()

	_, err := service.GetUserByID(ctx, 999999)
	if err == nil {
		t.Error("expected error when getting non-existent user")
	}
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserService_GetUserByIDInvalidInput(t *testing.T) {
	conn := setupUserServiceTestConn(t)
	defer conn.Close(context.Background())

	service := NewUserService(conn)
	ctx := context.Background()

	_, err := service.GetUserByID(ctx, 0)
	if err == nil {
		t.Error("expected error when getting user with invalid ID")
	}
	if err != ErrInvalidInput {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUserService_GetAllUsers(t *testing.T) {
	conn := setupUserServiceTestConn(t)
	defer conn.Close(context.Background())

	service := NewUserService(conn)
	ctx := context.Background()
	testUsername1 := "testuser_service_getall1"
	testUsername2 := "testuser_service_getall2"
	defer cleanupUserService(t, conn, testUsername1)
	defer cleanupUserService(t, conn, testUsername2)

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

	_, err := service.CreateUser(ctx, req1)
	if err != nil {
		t.Fatalf("failed to create user 1: %v", err)
	}
	_, err = service.CreateUser(ctx, req2)
	if err != nil {
		t.Fatalf("failed to create user 2: %v", err)
	}

	users, err := service.GetAllUsers(ctx)
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

func TestUserService_UpdateUser(t *testing.T) {
	conn := setupUserServiceTestConn(t)
	defer conn.Close(context.Background())

	service := NewUserService(conn)
	ctx := context.Background()
	testUsername := "testuser_service_update"
	defer cleanupUserService(t, conn, testUsername)

	req := model.CreateUserRequest{
		Username: testUsername,
		Email:    "old@example.com",
		Password: "password123",
	}

	createdUser, err := service.CreateUser(ctx, req)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	updateReq := model.UpdateUserRequest{
		Username: "newusername",
		Email:    "new@example.com",
		Password: "newpassword123",
	}

	updatedUser, err := service.UpdateUser(ctx, createdUser.ID, updateReq)
	if err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	if updatedUser.Username != updateReq.Username {
		t.Errorf("expected username %s, got %s", updateReq.Username, updatedUser.Username)
	}
	if updatedUser.Email != updateReq.Email {
		t.Errorf("expected email %s, got %s", updateReq.Email, updatedUser.Email)
	}

	// Verify new password was hashed correctly
	err = bcrypt.CompareHashAndPassword([]byte(updatedUser.PasswordHash), []byte(updateReq.Password))
	if err != nil {
		t.Error("new password was not hashed correctly")
	}

	// Cleanup with new username
	cleanupUserService(t, conn, "newusername")
}

func TestUserService_DeleteUser(t *testing.T) {
	conn := setupUserServiceTestConn(t)
	defer conn.Close(context.Background())

	service := NewUserService(conn)
	ctx := context.Background()
	testUsername := "testuser_service_delete_" + randomStringService(8)

	req := model.CreateUserRequest{
		Username: testUsername,
		Email:    testUsername + "@example.com",
		Password: "password123",
	}

	createdUser, err := service.CreateUser(ctx, req)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	err = service.DeleteUser(ctx, createdUser.ID)
	if err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	// Verify user is deleted
	_, err = service.GetUserByID(ctx, createdUser.ID)
	if err == nil {
		t.Error("expected error when getting deleted user")
	}
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserService_VerifyPassword(t *testing.T) {
	conn := setupUserServiceTestConn(t)
	defer conn.Close(context.Background())

	service := NewUserService(conn)
	ctx := context.Background()
	testUsername := "testuser_service_verify_" + randomStringService(8)
	defer cleanupUserService(t, conn, testUsername)

	password := "password123"
	req := model.CreateUserRequest{
		Username: testUsername,
		Email:    testUsername + "@example.com",
		Password: password,
	}

	_, err := service.CreateUser(ctx, req)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Test correct password
	user, err := service.VerifyPassword(ctx, testUsername, password)
	if err != nil {
		t.Fatalf("failed to verify correct password: %v", err)
	}
	if user.Username != testUsername {
		t.Errorf("expected username %s, got %s", testUsername, user.Username)
	}

	// Test incorrect password
	_, err = service.VerifyPassword(ctx, testUsername, "wrongpassword")
	if err == nil {
		t.Error("expected error when verifying incorrect password")
	}

	// Test non-existent user
	_, err = service.VerifyPassword(ctx, "nonexistentuser", password)
	if err == nil {
		t.Error("expected error when verifying password for non-existent user")
	}
}

func TestUserService_PasswordHashing(t *testing.T) {
	conn := setupUserServiceTestConn(t)
	defer conn.Close(context.Background())

	service := NewUserService(conn)

	password := "testpassword123"
	hash, err := service.hashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	// Verify the hash starts with bcrypt prefix
	if len(hash) < 20 {
		t.Error("hash seems too short")
	}

	// Verify we can verify the password against the hash
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		t.Error("password verification failed")
	}

	// Verify different passwords produce different hashes
	hash2, err := service.hashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password second time: %v", err)
	}

	if hash == hash2 {
		t.Error("same password should produce different hashes due to salt")
	}
}

// isErrorType checks if an error is of a specific type or contains that error type
func isErrorType(err, target error) bool {
	if err == target {
		return true
	}
	// Check if error message contains the target error message
	if err != nil && target != nil {
		return strings.Contains(err.Error(), target.Error())
	}
	return false
}