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

func SearchFilesByName(ctx context.Context, conn *pgx.Conn, query string) ([]model.FileInfo, error) {
	rows, err := conn.Query(ctx, `
		SELECT path, name, size, mod_time 
		FROM files 
		WHERE name ILIKE '%' || $1 || '%'
		ORDER BY name
	`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []model.FileInfo
	for rows.Next() {
		var file model.FileInfo
		err := rows.Scan(&file.Path, &file.Name, &file.Size, &file.ModTime)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}

	return files, rows.Err()
}
