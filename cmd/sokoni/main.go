package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/koplec/sokoni/internal/api"
	"github.com/koplec/sokoni/internal/cmd"
	"github.com/koplec/sokoni/internal/collector"
	"github.com/koplec/sokoni/internal/db"
	"github.com/koplec/sokoni/internal/scheduler"
	"github.com/koplec/sokoni/internal/service"
)

func main() {
	err := godotenv.Load("test.env")
	if err != nil {
		log.Printf("Warning: Could not load test.env file: %v", err)
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "scan":
			if len(os.Args) > 2 {
				connectionID, err := strconv.Atoi(os.Args[2])
				if err != nil {
					log.Fatalf("invalid connection ID: %v", err)
				}
				withDB(func(conn *pgx.Conn) {
					err := cmd.ScanConnection(connectionID, service.NewConnectionScanner(conn))
					if err != nil {
						log.Fatalf("scan failed: %v", err)
					}
				})
			} else {
				runScan()
			}
		case "scheduler":
			runScheduler()
		case "api":
			runAPI()
		default:
			showUsage()
		}
	} else {
		runAPI() // デフォルトはAPI
	}
}

func withDB(fn func(*pgx.Conn)) {
	ctx := context.Background()
	conn, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)
	fn(conn)
}

func runAPI() {
	ctx := context.Background()
	conn, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	apiHandler := api.NewAPI(conn)

	http.HandleFunc("/search", apiHandler.SearchFiles)
	http.HandleFunc("/connections", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/connections" && r.Method == "GET" {
			apiHandler.GetConnections(w, r)
		} else if r.URL.Path == "/connections" && r.Method == "POST" {
			apiHandler.CreateConnection(w, r)
		} else {
			apiHandler.GetConnection(w, r)
		}
	})
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting API server on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func runScheduler() {
	ctx := context.Background()
	conn, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	scanner := scheduler.NewScanner(conn)
	
	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-c
		log.Println("Shutting down scheduler...")
		scanner.Stop()
		os.Exit(0)
	}()

	fmt.Println("Starting scheduler daemon...")
	scanner.Start()
}

func runScan() {
	fmt.Println("Scanning files...")

	path := "/mnt/share"
	files, err := collector.Scan(path)
	if err != nil {
		log.Fatalf("failed to scan files: %v", err)
	}

	fmt.Printf("Found %d PDF files:\n", len(files))
	for _, f := range files {
		fmt.Printf("- %s (%d bytes)\n", f.Path, f.Size)
	}
}


func showUsage() {
	fmt.Println("Usage: sokoni [command]")
	fmt.Println("Commands:")
	fmt.Println("  api              Start REST API server (default)")
	fmt.Println("  scheduler        Start background file scanner")
	fmt.Println("  scan             Run one-time file scan")
	fmt.Println("  scan <conn_id>   Scan specific connection")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ./sokoni api       # Start API on port 8080")
	fmt.Println("  ./sokoni scheduler # Start background scanner")
	fmt.Println("  ./sokoni scan      # Manual scan of /mnt/share")
	fmt.Println("  ./sokoni scan 1    # Scan connection ID 1")
}
