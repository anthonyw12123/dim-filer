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
func Worker(id int, jobs <-chan IngestJob, results chan<- string) {
	for job := range jobs {
		fmt.Printf("Worker %d: Processing %s\n", id, job.Filename)

		// 1. Create Destination Directory
		destDir := filepath.Join(job.DestRoot, job.SubFolder)
		os.MkdirAll(destDir, 0755)

		// 2. Perform the Copy
		finalPath := filepath.Join(destDir, job.Filename)
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