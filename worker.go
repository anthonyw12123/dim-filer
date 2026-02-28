package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// IngestJob represents one file's journey from SD to SSD
type IngestJob struct {
	Source      string
	DestRoot    string
	SubFolder   string // e.g., "2026/2026-02-28"
	Filename    string
	Config      *Config
}

// Worker is a single concurrent processor
// Update the signature to accept dryRun
func Worker(id int, jobs <-chan IngestJob, results chan<- string, dryRun bool) {
	for job := range jobs {
		destDir := filepath.Join(job.DestRoot, job.SubFolder)
		finalPath := filepath.Join(destDir, job.Filename)

		if dryRun {
			results <- fmt.Sprintf("Worker %d: [DRY RUN] Would move %s -> %s", id, job.Filename, finalPath)
			continue
		}

		// ACTUAL EXECUTION
		os.MkdirAll(destDir, 0755)
		err := copyFile(job.Source, finalPath)
		if err != nil {
			results <- fmt.Sprintf("Worker %d: ❌ Failed %s: %v", id, job.Filename, err)
			continue
		}

		// 3. Trigger Rust Engine for Preview (Stub for now)
		// triggerEngine(job.Config.EnginePath, finalPath)

		results <- fmt.Sprintf("Worker %d: ✅ Completed %s", id, job.Filename)
	}
}

// copyFile is a simple, efficient stream-based copy
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}