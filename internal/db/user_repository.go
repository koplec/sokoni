package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/koplec/sokoni/internal/model"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

func GetAllUsers(ctx context.Context, conn *pgx.Conn) ([]*model.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var u model.User
		err := rows.Scan(
			&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, &u)
	}

	return users, rows.Err()
}

func GetUserByID(ctx context.Context, conn *pgx.Conn, id int) (*model.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var u model.User
	err := conn.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &u, nil
}

func GetUserByUsername(ctx context.Context, conn *pgx.Conn, username string) (*model.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	var u model.User
	err := conn.QueryRow(ctx, query, username).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &u, nil
}

func GetUserByEmail(ctx context.Context, conn *pgx.Conn, email string) (*model.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var u model.User
	err := conn.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &u, nil
}

func CreateUser(ctx context.Context, conn *pgx.Conn, req model.CreateUserRequest, passwordHash string) (*model.User, error) {
	// Check if user already exists
	existingUser, err := GetUserByUsername(ctx, conn, req.Username)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	// Check if email already exists
	existingUser, err = GetUserByEmail(ctx, conn, req.Email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	query := `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, username, email, password_hash, created_at, updated_at
	`

	var u model.User
	err = conn.QueryRow(ctx, query, req.Username, req.Email, passwordHash).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func UpdateUser(ctx context.Context, conn *pgx.Conn, id int, req model.UpdateUserRequest, passwordHash *string) (*model.User, error) {
	// Check if user exists
	existingUser, err := GetUserByID(ctx, conn, id)
	if err != nil {
		return nil, err
	}

	// Check for username/email conflicts if they're being updated
	if req.Username != "" && req.Username != existingUser.Username {
		conflictUser, err := GetUserByUsername(ctx, conn, req.Username)
		if err != nil && !errors.Is(err, ErrUserNotFound) {
			return nil, err
		}
		if conflictUser != nil {
			return nil, ErrUserAlreadyExists
		}
	}

	if req.Email != "" && req.Email != existingUser.Email {
		conflictUser, err := GetUserByEmail(ctx, conn, req.Email)
		if err != nil && !errors.Is(err, ErrUserNotFound) {
			return nil, err
		}
		if conflictUser != nil {
			return nil, ErrUserAlreadyExists
		}
	}

	// Build update query based on provided fields
	query := `
		UPDATE users 
		SET username = COALESCE($2, username), 
		    email = COALESCE($3, email), 
		    password_hash = COALESCE($4, password_hash),
		    updated_at = now()
		WHERE id = $1
		RETURNING id, username, email, password_hash, created_at, updated_at
	`

	var username, email *string
	if req.Username != "" {
		username = &req.Username
	}
	if req.Email != "" {
		email = &req.Email
	}

	var u model.User
	err = conn.QueryRow(ctx, query, id, username, email, passwordHash).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func DeleteUser(ctx context.Context, conn *pgx.Conn, id int) error {
	result, err := conn.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}