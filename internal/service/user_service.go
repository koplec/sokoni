package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/koplec/sokoni/internal/db"
	"github.com/koplec/sokoni/internal/model"
)

var (
	ErrInvalidInput     = errors.New("invalid input")
	ErrUserNotFound     = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrWeakPassword     = errors.New("password is too weak")
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrInvalidUsername  = errors.New("invalid username format")
)

// UserService provides user management operations with business logic
type UserService struct {
	conn *pgx.Conn
}

// NewUserService creates a new user service instance
func NewUserService(conn *pgx.Conn) *UserService {
	return &UserService{conn: conn}
}

// CreateUser creates a new user with validation and password hashing
func (s *UserService) CreateUser(ctx context.Context, req model.CreateUserRequest) (*model.User, error) {
	// Validate input
	if err := s.validateCreateUserRequest(req); err != nil {
		return nil, err
	}

	// Hash password
	passwordHash, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user in database
	user, err := db.CreateUser(ctx, s.conn, req, passwordHash)
	if err != nil {
		if errors.Is(err, db.ErrUserAlreadyExists) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetAllUsers retrieves all users from the database
func (s *UserService) GetAllUsers(ctx context.Context) ([]*model.User, error) {
	users, err := db.GetAllUsers(ctx, s.conn)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	return users, nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(ctx context.Context, id int) (*model.User, error) {
	if id <= 0 {
		return nil, ErrInvalidInput
	}

	user, err := db.GetUserByID(ctx, s.conn, id)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	if username == "" {
		return nil, ErrInvalidInput
	}

	user, err := db.GetUserByUsername(ctx, s.conn, username)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// UpdateUser updates an existing user
func (s *UserService) UpdateUser(ctx context.Context, id int, req model.UpdateUserRequest) (*model.User, error) {
	if id <= 0 {
		return nil, ErrInvalidInput
	}

	// Validate input
	if err := s.validateUpdateUserRequest(req); err != nil {
		return nil, err
	}

	var passwordHash *string
	if req.Password != "" {
		hash, err := s.hashPassword(req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		passwordHash = &hash
	}

	user, err := db.UpdateUser(ctx, s.conn, id, req, passwordHash)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		if errors.Is(err, db.ErrUserAlreadyExists) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// DeleteUser deletes a user by ID
func (s *UserService) DeleteUser(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrInvalidInput
	}

	err := db.DeleteUser(ctx, s.conn, id)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// VerifyPassword verifies a user's password
func (s *UserService) VerifyPassword(ctx context.Context, username, password string) (*model.User, error) {
	if username == "" || password == "" {
		return nil, ErrInvalidInput
	}

	user, err := db.GetUserByUsername(ctx, s.conn, username)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			return nil, db.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, db.ErrInvalidCredentials
	}

	return user, nil
}

// validateCreateUserRequest validates the create user request
func (s *UserService) validateCreateUserRequest(req model.CreateUserRequest) error {
	if req.Username == "" {
		return fmt.Errorf("%w: username is required", ErrInvalidInput)
	}

	if req.Email == "" {
		return fmt.Errorf("%w: email is required", ErrInvalidInput)
	}

	if req.Password == "" {
		return fmt.Errorf("%w: password is required", ErrInvalidInput)
	}

	if err := s.validateUsername(req.Username); err != nil {
		return err
	}

	if err := s.validateEmail(req.Email); err != nil {
		return err
	}

	if err := s.validatePassword(req.Password); err != nil {
		return err
	}

	return nil
}

// validateUpdateUserRequest validates the update user request
func (s *UserService) validateUpdateUserRequest(req model.UpdateUserRequest) error {
	if req.Username != "" {
		if err := s.validateUsername(req.Username); err != nil {
			return err
		}
	}

	if req.Email != "" {
		if err := s.validateEmail(req.Email); err != nil {
			return err
		}
	}

	if req.Password != "" {
		if err := s.validatePassword(req.Password); err != nil {
			return err
		}
	}

	return nil
}

// validateUsername validates username format
func (s *UserService) validateUsername(username string) error {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 50 {
		return fmt.Errorf("%w: username must be between 3 and 50 characters", ErrInvalidUsername)
	}

	// Allow alphanumeric characters, underscores, and hyphens
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, username); !matched {
		return fmt.Errorf("%w: username can only contain letters, numbers, underscores, and hyphens", ErrInvalidUsername)
	}

	return nil
}

// validateEmail validates email format
func (s *UserService) validateEmail(email string) error {
	email = strings.TrimSpace(email)
	if len(email) > 254 {
		return fmt.Errorf("%w: email is too long", ErrInvalidEmail)
	}

	// Simple email validation regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("%w: invalid email format", ErrInvalidEmail)
	}

	return nil
}

// validatePassword validates password strength
func (s *UserService) validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("%w: password must be at least 8 characters long", ErrWeakPassword)
	}

	if len(password) > 128 {
		return fmt.Errorf("%w: password is too long", ErrWeakPassword)
	}

	// Check for at least one letter and one number
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)

	if !hasLetter || !hasNumber {
		return fmt.Errorf("%w: password must contain at least one letter and one number", ErrWeakPassword)
	}

	return nil
}

// hashPassword hashes a password using bcrypt
func (s *UserService) hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}