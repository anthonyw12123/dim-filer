package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("🚀 Dim-Filer: Starting Ingest...")

	// 1. Load Configs
	cfg, err := LoadAndMerge("global.json", "import.json")
	if err != nil {
		log.Fatalf("❌ Configuration Error: %v", err)
	}

	// 2. Scan Source
	fmt.Printf("🔍 Scanning: %s\n", cfg.SourceDir)
	orders, err := ScanSource(cfg.SourceDir, cfg.AllowedTypes)
	if err != nil {
		log.Fatalf("❌ Scan Error: %v", err)
	}

	fmt.Printf("✅ Found %d RAW files to process.\n", len(orders))

	// 3. Dispatch (Coming next!)
	// For now, let's just print what we found
	for _, order := range orders {
		fmt.Printf("   Ready: %s\n", order.Filename)
	}
}