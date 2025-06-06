package cmd

import (
	"context"
	"log"

	"github.com/koplec/sokoni/internal/service"
)

// ScanConnection は指定されたconnection IDのNAS/ローカルパスをスキャンして、
// 見つかったPDFファイルの情報をデータベースに保存する。
// - connectionID: データベースに登録されているconnection ID
// - scanConnectionFunc: 実際のスキャン処理を行う関数（依存性注入）
func ScanConnection(connectionID int, scanConnectionFunc service.ScanConnectionFunc) {
	ctx := context.Background()

	userID := -1 // 仮のユーザーID（開発用）
	
	err := scanConnectionFunc(ctx, connectionID, userID)
	if err != nil {
		log.Fatalf("failed to scan connection: %v", err)
	}
}