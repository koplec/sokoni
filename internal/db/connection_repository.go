package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type Connection struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	BasePath     string    `json:"base_path"`
	RemotePath   string    `json:"remote_path"`
	Username     *string   `json:"username,omitempty"`
	Password     *string   `json:"password,omitempty"`
	Options      *string   `json:"options,omitempty"`
	UserID       int       `json:"user_id"`
	LastScan     *time.Time `json:"last_scan,omitempty"`
	ScanInterval int       `json:"scan_interval"`
	AutoScan     bool      `json:"auto_scan"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateConnectionRequest struct {
	Name         string  `json:"name"`
	BasePath     string  `json:"base_path"`
	RemotePath   string  `json:"remote_path"`
	Username     *string `json:"username,omitempty"`
	Password     *string `json:"password,omitempty"`
	Options      *string `json:"options,omitempty"`
	UserID       int     `json:"user_id"`
	ScanInterval *int    `json:"scan_interval,omitempty"`
	AutoScan     *bool   `json:"auto_scan,omitempty"`
}

func GetConnectionsByUserID(ctx context.Context, conn *pgx.Conn, userID int) ([]Connection, error) {
	query := `
		SELECT id, name, base_path, remote_path, username, password, options,
		       user_id, last_scan, scan_interval, auto_scan, created_at, updated_at
		FROM connections
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := conn.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []Connection
	for rows.Next() {
		var c Connection
		err := rows.Scan(
			&c.ID, &c.Name, &c.BasePath, &c.RemotePath, &c.Username, &c.Password, &c.Options,
			&c.UserID, &c.LastScan, &c.ScanInterval, &c.AutoScan, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		connections = append(connections, c)
	}

	return connections, rows.Err()
}

func GetConnectionByID(ctx context.Context, conn *pgx.Conn, id int, userID int) (*Connection, error) {
	query := `
		SELECT id, name, base_path, remote_path, username, password, options,
		       user_id, last_scan, scan_interval, auto_scan, created_at, updated_at
		FROM connections
		WHERE id = $1 AND user_id = $2
	`

	var c Connection
	err := conn.QueryRow(ctx, query, id, userID).Scan(
		&c.ID, &c.Name, &c.BasePath, &c.RemotePath, &c.Username, &c.Password, &c.Options,
		&c.UserID, &c.LastScan, &c.ScanInterval, &c.AutoScan, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func CreateConnection(ctx context.Context, conn *pgx.Conn, req CreateConnectionRequest) (*Connection, error) {
	scanInterval := 604800 // 1週間デフォルト
	if req.ScanInterval != nil {
		scanInterval = *req.ScanInterval
	}

	autoScan := true
	if req.AutoScan != nil {
		autoScan = *req.AutoScan
	}

	query := `
		INSERT INTO connections (name, base_path, remote_path, username, password, options, user_id, scan_interval, auto_scan)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, name, base_path, remote_path, username, password, options,
		          user_id, last_scan, scan_interval, auto_scan, created_at, updated_at
	`

	var c Connection
	err := conn.QueryRow(ctx, query,
		req.Name, req.BasePath, req.RemotePath, req.Username, req.Password, req.Options,
		req.UserID, scanInterval, autoScan,
	).Scan(
		&c.ID, &c.Name, &c.BasePath, &c.RemotePath, &c.Username, &c.Password, &c.Options,
		&c.UserID, &c.LastScan, &c.ScanInterval, &c.AutoScan, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func UpdateConnection(ctx context.Context, conn *pgx.Conn, id int, userID int, req CreateConnectionRequest) (*Connection, error) {
	query := `
		UPDATE connections 
		SET name = $3, base_path = $4, remote_path = $5, username = $6, password = $7, options = $8,
		    scan_interval = COALESCE($9, scan_interval), auto_scan = COALESCE($10, auto_scan), updated_at = now()
		WHERE id = $1 AND user_id = $2
		RETURNING id, name, base_path, remote_path, username, password, options,
		          user_id, last_scan, scan_interval, auto_scan, created_at, updated_at
	`

	var c Connection
	err := conn.QueryRow(ctx, query,
		id, userID, req.Name, req.BasePath, req.RemotePath, req.Username, req.Password, req.Options,
		req.ScanInterval, req.AutoScan,
	).Scan(
		&c.ID, &c.Name, &c.BasePath, &c.RemotePath, &c.Username, &c.Password, &c.Options,
		&c.UserID, &c.LastScan, &c.ScanInterval, &c.AutoScan, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func DeleteConnection(ctx context.Context, conn *pgx.Conn, id int, userID int) error {
	result, err := conn.Exec(ctx, "DELETE FROM connections WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}