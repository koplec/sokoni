package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/koplec/sokoni/internal/model"
)

func InsertFile(ctx context.Context, conn *pgx.Conn, connectionID int, file model.FileInfo) error {
	_, err := conn.Exec(ctx, `
	INSERT into files (connection_id, path, size, name, mod_time)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (path) DO UPDATE
	SET size = EXCLUDED.size, 
		mod_time = EXCLUDED.mod_time,
		updated_at = now()
	`, connectionID, file.Path, file.Size, file.Name, file.ModTime)
	return err
}
