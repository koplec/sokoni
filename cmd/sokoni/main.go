package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/koplec/sokoni/internal/api"
	"github.com/koplec/sokoni/internal/collector"
	"github.com/koplec/sokoni/internal/db"
)

func main() {
	err := godotenv.Load("test.env")
	if err != nil {
		log.Printf("Warning: Could not load test.env file: %v", err)
	}

	ctx := context.Background()
	conn, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	if len(os.Args) > 1 && os.Args[1] == "scan" {
		runScan()
		return
	}

	apiHandler := api.NewAPI(conn)

	http.HandleFunc("/search", apiHandler.SearchFiles)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting server on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
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
