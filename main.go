package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	// Define the -dryrun flag (defaults to false)
	dryRun := flag.Bool("dryrun", false, "Scan and log operations without moving files")
	flag.Parse()

	if *dryRun {
		fmt.Println("⚠️  DRY RUN MODE: No files will be moved or modified.")
	}

	// 1. Load Configs
	cfg, err := LoadAndMerge("global.json", "import.json")
	if err != nil {
		log.Fatalf("❌ Configuration Error: %v", err)
	}

	// 2. Scan Source
	orders, err := ScanSource(cfg.SourceDir, cfg.AllowedTypes)
	if err != nil {
		log.Fatalf("❌ Scan Error: %v", err)
	}

	// 3. Launch Worker Pool
	// We use a channel to send jobs and a channel to receive results
	jobs := make(chan IngestJob, len(orders))
	results := make(chan string, len(orders))

	// Start 4 workers (The Heavy Lifters)
	for w := 1; w <= 4; w++ {
		go Worker(w, jobs, results, *dryRun)
	}

	// 4. Fill the Conveyor Belt
	for _, order := range orders {
		// Get the actual date from the RAW file
        datePath, err := GetImgDate(order.SourcePath)
        if err != nil {
            fmt.Printf("⚠️  Warning: Could not read EXIF for %s, using fallback.\n", order.Filename)
            datePath = "Unknown/Unknown-Date"
        }

		jobs <- IngestJob{
            Source:    order.SourcePath,
            DestRoot:  cfg.Destinations[0],
            SubFolder: datePath,
            Filename:  order.Filename,
            Config:    cfg,
        }
	}
	close(jobs) // Tell workers no more jobs are coming

	// 5. Collect Results
	for i := 0; i < len(orders); i++ {
		fmt.Println(<-results)
	}

	fmt.Println("🏁 Operation complete.")
}
