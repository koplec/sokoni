package main

import (
	"fmt"
	"log"

	"github.com/koplec/sokoni/internal/collector"
)

func main() {
	fmt.Println("Hello sokoni!")

	path := "/mnt/share"

	files, err := collector.Scan(path)
	if err != nil {
		log.Fatalf("failed to scan files: %v", err)
	}

	fmt.Printf("Found %d PDF files:\n", len(files))
	for _, f := range files {
		fmt.Printf("- %s (%d bytes)=\n", f.Path, f.Size)
	}
}
