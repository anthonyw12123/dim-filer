package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// IngestJob represents one file's journey from SD to SSD
type IngestJob struct {
	Source    string
	DestRoot  string
	SubFolder string // e.g., "2026/2026-02-28"
	Filename  string
	Config    *Config
}

// EngineRecipe matches the Rust `Recipe` struct.
// We only populate the fields we need for the Fast-Path extraction.
type EngineRecipe struct {
	ExtractJpg     *bool          `json:"extract_jpg"`
	OutputPath     string         `json:"output_path"`
	OrientationMap map[string]int `json:"orientation_map"`
}

// Worker is a single concurrent processor
func Worker(id int, jobs <-chan IngestJob, results chan<- string, dryRun bool) {
	for job := range jobs {
		destDir := filepath.Join(job.DestRoot, job.SubFolder)
		previewDir := filepath.Join(destDir, ".previews")
		finalPath := filepath.Join(destDir, job.Filename)

		if dryRun {
			results <- fmt.Sprintf("Worker %d: [DRY RUN] Would ingest %s and generate recipe", id, job.Filename)
			continue
		}

		// 1. Create Directories
		os.MkdirAll(destDir, 0755)
		if job.Config.ExtractPreviews {
			os.MkdirAll(previewDir, 0755)
		}

		// 2. Perform Copy (SD -> SSD)
		if err := copyFile(job.Source, finalPath); err != nil {
			results <- fmt.Sprintf("Worker %d: ❌ Copy Failed %s: %v", id, job.Filename, err)
			continue
		}

		// 3. Verify Hash
		if job.Config.VerifyChecksums {
			match, err := verifyChecksum(job.Source, finalPath)
			if err != nil || !match {
				// If the copy corrupted, we delete the bad file and report the error
				os.Remove(finalPath)
				results <- fmt.Sprintf("Worker %d: ❌ Checksum Failed (Corrupted Copy) %s", id, job.Filename)
				continue
			}
			// Your requested console output!
			fmt.Printf("Worker %d: 🔒 Checksum verified for %s\n", id, job.Filename)
		}

		// 4. Generate JSON Recipe & Trigger Rust
		if job.Config.ExtractPreviews {
			previewPath := filepath.Join(previewDir, job.Filename+".jpg")
			recipePath := filepath.Join(previewDir, job.Filename+".json")

			// Build the payload
			extractFlag := true
			recipe := EngineRecipe{
				ExtractJpg:     &extractFlag,
				OutputPath:     previewPath,
				OrientationMap: job.Config.OrientationMap,
			}

			// Write the JSON to disk
			recipeData, _ := json.MarshalIndent(recipe, "", "  ")
			os.WriteFile(recipePath, recipeData, 0644)

			// Execute Rust: ./dim-engine /SSD/Path/RAW.arw /SSD/Path/.previews/RAW.json
			cmd := exec.Command(job.Config.EnginePath, finalPath, recipePath)

			// Capture the raw output from the Rust engine
			output, err := cmd.CombinedOutput()
			if err != nil {
				results <- fmt.Sprintf("Worker %d: ⚠️  Preview Failed for %s: %v\n--- RUST LOG ---\n%s----------------", id, job.Filename, err, string(output))
			}

			// --- Cleanup the temporary Recipe ---
			if job.Config.DeleteRecipes {
				os.Remove(recipePath)
			}
		}

		// --- Cleanup the Original SD Card File ---
		// We ONLY do this if the copy and hash verification were 100% successful.
		if job.Config.DeleteOriginals {
			if err := os.Remove(job.Source); err != nil {
				results <- fmt.Sprintf("Worker %d: ⚠️  Could not delete original %s: %v", id, job.Filename, err)
			} else {
				fmt.Printf("Worker %d: 🗑️  Deleted original %s\n", id, job.Filename)
			}
		}

		results <- fmt.Sprintf("Worker %d: ✅ Success %s", id, job.Filename)
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

// verifyChecksum compares the SHA256 hashes of two files
func verifyChecksum(src, dst string) (bool, error) {
	srcHash, err := hashFile(src)
	if err != nil {
		return false, err
	}

	dstHash, err := hashFile(dst)
	if err != nil {
		return false, err
	}

	return bytes.Equal(srcHash, dstHash), nil
}

// hashFile calculates the SHA256 hash of a file without loading it all into RAM
func hashFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}
