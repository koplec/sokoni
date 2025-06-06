package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/koplec/sokoni/internal/collector"
	"github.com/koplec/sokoni/internal/db"
	"github.com/koplec/sokoni/internal/model"
)

type ScanConnectionFunc func(ctx context.Context, connectionID int, userID int) error

// NewScanConnectionFunc は指定されたデータベース接続を使用して、
// connectionをスキャンする関数を作成する。
// 
// この関数は以下の処理を行う：
// 1. connection情報をDBから取得
// 2. SMB/CIFS または ローカルファイルシステムから PDFファイルをスキャン
// 3. 見つかったファイルを100件ずつバッチでDBに保存
// 4. 進捗状況をログ出力
//
// - conn: PostgreSQL データベース接続
// 戻り値: ScanConnectionFunc (connectionID, userIDを受け取りスキャンを実行する関数)
func NewScanConnectionFunc(conn *pgx.Conn) ScanConnectionFunc {
	return func(ctx context.Context, connectionID int, userID int) error {
		connection, err := db.GetConnectionByID(ctx, conn, connectionID, userID)
		if err != nil {
			return fmt.Errorf("failed to get connection: %w", err)
		}

		fmt.Printf("Scanning connection: %s (%s)\n", connection.Name, connection.BasePath)

		const batchSize = 100
		var batch []model.FileInfo
		var totalCount int

		err = collector.ScanConnectionWith(connection, func(file model.FileInfo) error {
			batch = append(batch, file)
			totalCount++

			if len(batch) >= batchSize {
				if err := insertFileBatch(ctx, conn, connectionID, batch); err != nil {
					return err
				}
				batch = batch[:0] // clear slice
				fmt.Printf("Processed %d files...\n", totalCount)
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to scan files: %w", err)
		}

		// Insert remaining files in batch
		if len(batch) > 0 {
			if err := insertFileBatch(ctx, conn, connectionID, batch); err != nil {
				return err
			}
		}

		fmt.Printf("Successfully stored %d files for connection %s\n", totalCount, connection.Name)
		return nil
	}
}

func insertFileBatch(ctx context.Context, conn *pgx.Conn, connectionID int, files []model.FileInfo) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, f := range files {
		_, err := tx.Exec(ctx, `
			INSERT into files (connection_id, path, size, name, mod_time)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (path) DO UPDATE
			SET size = EXCLUDED.size, 
				mod_time = EXCLUDED.mod_time,
				updated_at = now()
		`, connectionID, f.Path, f.Size, f.Name, f.ModTime)
		if err != nil {
			return fmt.Errorf("failed to insert file %s: %w", f.Path, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}