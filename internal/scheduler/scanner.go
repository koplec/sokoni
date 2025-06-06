package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/koplec/sokoni/internal/collector"
	"github.com/koplec/sokoni/internal/db"
	"github.com/koplec/sokoni/internal/model"
)

type Scanner struct {
	conn *pgx.Conn
	ctx  context.Context
	done chan struct{}
}

func NewScanner(conn *pgx.Conn) *Scanner {
	return &Scanner{
		conn: conn,
		ctx:  context.Background(),
		done: make(chan struct{}),
	}
}

func (s *Scanner) Start() {
	ticker := time.NewTicker(6 * time.Hour) // 6時間ごとにチェック
	defer ticker.Stop()

	log.Println("Scanner started (checking every 6 hours)")

	// 起動時に1回チェック
	s.scanDueConnections()

	for {
		select {
		case <-ticker.C:
			s.scanDueConnections()
		case <-s.done:
			log.Println("Scanner stopped")
			return
		}
	}
}

func (s *Scanner) Stop() {
	close(s.done)
}

func (s *Scanner) scanDueConnections() {
	connections, err := s.getDueConnections()
	if err != nil {
		log.Printf("Error getting due connections: %v", err)
		return
	}

	if len(connections) == 0 {
		log.Println("No connections due for scanning")
		return
	}

	log.Printf("Found %d connections due for scanning", len(connections))

	for _, conn := range connections {
		log.Printf("Starting scan for connection: %s (ID: %d, Path: %s)", conn.Name, conn.ID, conn.BasePath)
		
		fileCount := 0
		err := s.scanConnection(conn, &fileCount)
		if err != nil {
			log.Printf("Error scanning connection %s: %v", conn.Name, err)
			continue
		}
		
		err = s.updateLastScan(conn.ID)
		if err != nil {
			log.Printf("Error updating last_scan for connection %s: %v", conn.Name, err)
		} else {
			log.Printf("Completed scan for %s: processed %d files", conn.Name, fileCount)
		}
	}
}

type Connection struct {
	ID       int
	Name     string
	BasePath string
	UserID   int
}

func (s *Scanner) getDueConnections() ([]Connection, error) {
	query := `
		SELECT id, name, base_path, user_id 
		FROM connections 
		WHERE auto_scan = true 
		AND (last_scan IS NULL OR last_scan + (scan_interval || ' seconds')::interval < now())
	`
	
	rows, err := s.conn.Query(s.ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []Connection
	for rows.Next() {
		var conn Connection
		err := rows.Scan(&conn.ID, &conn.Name, &conn.BasePath, &conn.UserID)
		if err != nil {
			return nil, err
		}
		connections = append(connections, conn)
	}

	return connections, rows.Err()
}

func (s *Scanner) scanConnection(conn Connection, fileCount *int) error {
	return collector.ScanWith(conn.BasePath, func(fileInfo model.FileInfo) error {
		*fileCount++
		return db.InsertFile(s.ctx, s.conn, conn.ID, fileInfo)
	})
}

func (s *Scanner) updateLastScan(connectionID int) error {
	_, err := s.conn.Exec(s.ctx, 
		"UPDATE connections SET last_scan = now() WHERE id = $1", 
		connectionID)
	return err
}