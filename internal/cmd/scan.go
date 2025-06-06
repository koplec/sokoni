package cmd

import (
	"context"
	"log"
	"strconv"

	"github.com/koplec/sokoni/internal/service"
)

func ScanConnection(connectionIDStr string, scanConnectionFunc service.ScanConnectionFunc) {
	ctx := context.Background()

	connectionID, err := strconv.Atoi(connectionIDStr)
	if err != nil {
		log.Fatalf("invalid connection ID: %v", err)
	}

	userID := -1 // 仮のユーザーID（開発用）
	
	err = scanConnectionFunc(ctx, connectionID, userID)
	if err != nil {
		log.Fatalf("failed to scan connection: %v", err)
	}
}