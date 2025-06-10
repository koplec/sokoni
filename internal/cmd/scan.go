package cmd

import (
	"context"
	"fmt"

	"github.com/koplec/sokoni/internal/service"
)

// ScanConnection は指定されたconnection IDのNAS/ローカルパスをスキャンして、
// 見つかったPDFファイルの情報をデータベースに保存する。
// - connectionID: データベースに登録されているconnection ID
// - scanner: 実際のスキャン処理を行うスキャナー（依存性注入）
func ScanConnection(connectionID int, scanner service.ConnectionScanner) error {
	ctx := context.Background()

	userID := -1 // 仮のユーザーID（開発用）
	
	err := scanner(ctx, connectionID, userID)
	if err != nil {
		return fmt.Errorf("failed to scan connection %d: %w", connectionID, err)
	}
	return nil
}